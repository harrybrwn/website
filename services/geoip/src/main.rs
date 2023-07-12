use std::{env, io, net::IpAddr, path::Path, sync::RwLock};

use actix_web::http::header;
use actix_web::middleware::{self, DefaultHeaders};
use actix_web::{get, web, App, HttpResponse, HttpServer, Responder};
use anyhow::bail;
use clap::Parser;
use maxminddb::geoip2;
use serde::Serialize;

mod geoip;
mod ip;
mod locale;
mod logging;
mod s3;
#[cfg(test)]
mod tests;

use geoip::{GeoDB, GeoLoc, GeoError, ErrorResponse};
use ip::ClientIP;
use locale::{Locales};
use logging::new_logger;
use s3::open_from_s3;

async fn index(ip: ClientIP) -> impl Responder {
    // Adds newline for curl users
    format!("{}\n", ip.ip())
}

async fn lookup_address(
    address: web::Path<IpAddr>,
    db: web::Data<RwLock<GeoDB>>,
    locales: Locales,
) -> HttpResponse {
    let db = match db.try_read() {
        Ok(db) => db,
        Err(e) => return GeoError::from(e).into(),
    };
    match db.lookup(*address, &locales) {
        Ok(r) => HttpResponse::Ok()
            .insert_header((header::CONTENT_LANGUAGE, r.locale.clone()))
            .json(r),
        Err(e) => e.into(),
    }
}

async fn lookup_self(ip: ClientIP, db: web::Data<RwLock<GeoDB>>, locales: Locales) -> HttpResponse {
    let db = match db.try_read() {
        Ok(db) => db,
        Err(e) => return GeoError::from(e).into(),
    };
    match db.lookup(ip.ip(), &locales) {
        Ok(mut r) => {
            r.ip = Some(ip.ip());
            HttpResponse::Ok()
                .insert_header((header::CONTENT_LANGUAGE, r.locale.clone()))
                .json(r)
        }
        Err(e) => e.into(),
    }
}

async fn lookup_self_languages(ip: ClientIP, db: web::Data<RwLock<GeoDB>>) -> HttpResponse {
    let db = match db.try_read() {
        Ok(db) => db,
        Err(e) => return GeoError::from(e).into(),
    };
    match db.languages(ip.ip()) {
        Ok(langs) => HttpResponse::Ok().json(langs),
        Err(e) => e.into(),
    }
}

async fn lookup_address_languages(
    address: web::Path<IpAddr>,
    db: web::Data<RwLock<GeoDB>>,
) -> HttpResponse {
    let db = match db.try_read() {
        Ok(db) => db,
        Err(e) => return GeoError::from(e).into(),
    };
    match db.languages(*address) {
        Ok(langs) => HttpResponse::Ok().json(langs),
        Err(e) => e.into(),
    }
}

#[derive(Serialize)]
struct DebugResponse<'a> {
    geolocation: geoip2::City<'a>,
    response: Option<GeoLoc<'a>>,
    error: Option<ErrorResponse>,
    ip: IpAddr,
    client_ip: IpAddr,
    locales: Vec<String>,
}

#[cfg(debug_assertions)]
#[get("/{address}/debug")]
async fn lookup_address_debug(
    address: web::Path<IpAddr>,
    db: web::Data<RwLock<GeoDB>>,
    ip: ClientIP,
    locales: Locales,
) -> HttpResponse {
    let db = match db.try_read() {
        Ok(db) => db,
        Err(e) => return GeoError::from(e).into(),
    };
    let (response, error) = match db.lookup(*address, &locales) {
        Ok(r) => (Some(r), None),
        Err(e) => (None, Some(e.into())),
    };
    let langs: Vec<_> = locales.iter().map(|l| l.to_string()).collect();
    match db.city.lookup::<geoip2::City>(*address) {
        Ok(geolocation) => HttpResponse::Ok().json(DebugResponse {
            client_ip: ip.ip(),
            ip: *address,
            locales: langs,
            response,
            error,
            geolocation,
        }),
        Err(e) => GeoError::from(e).into(),
    }
}

#[cfg(not(debug_assertions))]
#[get("{address}/debug")]
async fn lookup_address_debug() -> HttpResponse {
    HttpResponse::NotFound().finish()
}

async fn languages(langs: web::Data<Vec<String>>) -> HttpResponse {
    HttpResponse::Ok().json(langs)
}

async fn open_mmdb<P>(path: P) -> anyhow::Result<maxminddb::Reader<Vec<u8>>>
where
    P: AsRef<Path>,
{
    use std::fs;
    let filepath = match path.as_ref().to_str() {
        None => bail!("invalid file path"),
        Some(p) => p,
    };
    let body = match url::Url::parse(filepath) {
        Ok(url) => match url.scheme() {
            "http" | "https" => todo!("Download from url"),
            "file" => todo!("handle files"),
            "s3" => open_from_s3(&url).await?,
            _ => panic!("unknown url schema"),
        },
        Err(_) => {
            log::debug!("getting GeoIP database file from local fs");
            fs::read(filepath)?
        }
    };
    match maxminddb::Reader::from_source(body) {
        Ok(db) => Ok(db),
        Err(e) => bail!("failed to parse GeoIP database from buffer: {:?}", e),
    }
}

macro_rules! cors_route {
    ($resource:expr, $handler_fn:ident) => {
        cors_route!($resource, $handler_fn, "*")
    };
    ($resource:expr, $handler_fn:ident, $origin:expr) => {
        $resource
            .route(
                web::route()
                    .method(actix_web::http::Method::GET)
                    .to($handler_fn),
            )
            .route(
                web::route()
                    .method(actix_web::http::Method::HEAD)
                    .to(|| HttpResponse::Ok()),
            )
            .route(
                web::route()
                    .method(actix_web::http::Method::OPTIONS)
                    .to(|| HttpResponse::Ok()),
            )
            .wrap(
                DefaultHeaders::new()
                    .add((header::ACCESS_CONTROL_ALLOW_ORIGIN, $origin))
                    .add((header::ACCESS_CONTROL_ALLOW_METHODS, "GET, OPTIONS"))
                    .add((
                        header::ACCESS_CONTROL_ALLOW_HEADERS,
                        "Content-Type, Accept-Language",
                    )),
            )
    };
}

fn prometheus() -> actix_web_prom::PrometheusMetrics {
    actix_web_prom::PrometheusMetricsBuilder::new("")
        .endpoint("/metrics")
        .build()
        .unwrap()
}

#[derive(clap::Parser, Debug)]
pub(crate) struct Cli {
    /// File path for GeoIP or GeoLite2 database file
    #[arg(long, env = "GEOIP_CITY_FILE")]
    city_file: Vec<String>,
    #[arg(long, env = "GEOIP_ASN_FILE")]
    asn_file: String,
    /// Server host
    #[arg(long, short = 'H', default_value = "0.0.0.0", env = "GEOIP_HOST")]
    host: String,
    /// Server port
    #[arg(short, long, default_value_t = 8084, env = "GEOIP_PORT")]
    port: u16,
    /// Concurrent workers
    #[arg(short, long, default_value_t = 8)]
    workers: usize,
    /// Allowed origin
    #[arg(long, default_value = "https://hrry.me", env = "GEOIP_ALLOWED_ORIGIN")]
    allowed_origin: String,
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    let args = Cli::parse_from(env::args());
    let log = new_logger("geoip")?;
    let prometheus = prometheus();

    let geoip_db = match open_mmdb(&args.city_file[0]).await {
        Ok(db) => db,
        Err(e) => {
            return Err(io::Error::new(io::ErrorKind::Other, e.to_string()));
        }
    };
    let city_db = match open_mmdb(&args.city_file[0]).await {
        Ok(db) => db,
        Err(e) => {
            return Err(io::Error::new(io::ErrorKind::Other, e.to_string()));
        }
    };
    let asn_db = match open_mmdb(&args.asn_file).await {
        Ok(db) => db,
        Err(e) => {
            return Err(io::Error::new(io::ErrorKind::Other, e.to_string()));
        }
    };
    let langs = geoip_db.metadata.languages.clone();
    let database = web::Data::new(RwLock::new(GeoDB::new(city_db, asn_db)));
    log::info!(
        "loaded geoip databases {} {}",
        args.city_file[0],
        args.asn_file
    );

    let origin = args.allowed_origin;
    HttpServer::new(move || {
        let app = App::new()
            // The prometheus client will record hits to the '/metrics' endpoint
            // as hits to '/{address}' if these is no route for /metrics. Must
            // be added before all other routes.
            .service(web::resource("/metrics").to(HttpResponse::Ok))
            .wrap(prometheus.clone())
            .wrap(logging::AutoLog::new(log.clone()))
            .wrap(middleware::NormalizePath::trim());

        app.app_data(web::Data::clone(&database))
            .app_data(web::Data::new(langs.clone()))
            .service(cors_route!(web::resource("/"), index))
            .service(cors_route!(web::resource("/languages"), languages))
            .service(cors_route!(
                web::resource("/self"),
                lookup_self,
                origin.clone()
            ))
            .service(cors_route!(
                web::resource("/self/languages"),
                lookup_self_languages,
                origin.clone()
            ))
            .service(cors_route!(
                web::resource("/{address}"),
                lookup_address,
                origin.clone()
            ))
            .service(cors_route!(
                web::resource("/{address}/languages"),
                lookup_address_languages,
                origin.clone()
            ))
            .service(lookup_address_debug)
    })
    .workers(args.workers)
    .bind(("0.0.0.0", args.port))?
    .run()
    .await
}
