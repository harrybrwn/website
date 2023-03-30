use std::{env, io, net::IpAddr, path::Path, sync::RwLock};

use actix_web::http::header;
use actix_web::middleware::{self, DefaultHeaders};
use actix_web::{get, web, App, HttpResponse, HttpServer, Responder};
use actix_web_prom::PrometheusMetricsBuilder;
use anyhow::bail;
use clap::Parser;
use maxminddb::geoip2;
use serde::Serialize;

mod ip;
mod locale;
mod logging;
mod models;
mod s3;
#[cfg(test)]
mod tests;

use ip::ClientIP;
use locale::{Language, Locales};
use logging::new_logger;
use models::{city_to_response, ErrorResponse, LocationError, LocationResponse};
use s3::open_from_s3;

type GeoIpDB = RwLock<maxminddb::Reader<Vec<u8>>>;

fn lookup(
    db: &web::Data<GeoIpDB>,
    ip: IpAddr,
    locales: &Locales,
) -> Result<LocationResponse, LocationError> {
    match db.try_read() {
        Ok(db) => match db.lookup::<geoip2::City>(ip) {
            Ok(c) => city_to_response(c, locales),
            Err(e) => Err(LocationError::from(e)),
        },
        Err(e) => Err(LocationError::from(e)),
    }
}

trait IpDb {
    fn lookup(&self, ip: IpAddr, lang: &Language) -> Result<LocationResponse, LocationError>;
}

async fn index(ip: ClientIP) -> impl Responder {
    // Adds newline for curl users
    format!("{}\n", ip.ip())
}

//#[get("/{address}")]
async fn lookup_address(
    address: web::Path<IpAddr>,
    db: web::Data<GeoIpDB>,
    locales: Locales,
) -> HttpResponse {
    match lookup(&db, *address, &locales) {
        Err(e) => e.into(),
        Ok(r) => HttpResponse::Ok()
            .insert_header((header::CONTENT_LANGUAGE, r.locale.clone()))
            .json(r),
    }
}

async fn lookup_self(ip: ClientIP, db: web::Data<GeoIpDB>, locales: Locales) -> HttpResponse {
    match lookup(&db, ip.ip(), &locales) {
        Err(e) => e.into(),
        Ok(mut r) => {
            r.ip = Some(ip.ip());
            let locale = r.locale.clone();
            HttpResponse::Ok()
                .insert_header((header::CONTENT_LANGUAGE, locale))
                .json(r)
        }
    }
}

async fn lookup_self_languages(ip: ClientIP, db: web::Data<GeoIpDB>) -> HttpResponse {
    let db = match db.try_read() {
        Ok(db) => db,
        Err(e) => return LocationError::from(e).into(),
    };
    match db.lookup::<geoip2::City>(ip.ip()) {
        Ok(city) => {
            match city
                .country
                .and_then(|c| c.names)
                .map(|n| n.iter().map(|(k, _)| *k).collect::<Vec<_>>())
            {
                Some(langs) => HttpResponse::Ok().json(langs),
                None => HttpResponse::NotFound().finish(),
            }
        }
        Err(e) => LocationError::from(e).into(),
    }
}

async fn lookup_address_languages(
    address: web::Path<IpAddr>,
    db: web::Data<GeoIpDB>,
) -> HttpResponse {
    let db = match db.try_read() {
        Ok(db) => db,
        Err(e) => return LocationError::from(e).into(),
    };
    match db.lookup::<geoip2::City>(*address) {
        Ok(city) => {
            match city
                .country
                .and_then(|c| c.names)
                .map(|n| n.iter().map(|(k, _)| *k).collect::<Vec<_>>())
            {
                Some(langs) => HttpResponse::Ok().json(langs),
                None => HttpResponse::NotFound().finish(),
            }
        }
        Err(e) => LocationError::from(e).into(),
    }
}

#[derive(Serialize)]
struct DebugResponse<'a> {
    geolocation: geoip2::City<'a>,
    response: Option<LocationResponse>,
    error: Option<ErrorResponse>,
    ip: IpAddr,
    client_ip: IpAddr,
    locales: Vec<String>,
}

#[cfg(debug_assertions)]
#[get("/{address}/debug")]
async fn lookup_address_debug(
    address: web::Path<IpAddr>,
    db: web::Data<GeoIpDB>,
    ip: ClientIP,
    locales: Locales,
) -> HttpResponse {
    let (response, error) = match lookup(&db, *address, &locales) {
        Ok(r) => (Some(r), None),
        Err(e) => (None, Some(e.into())),
    };
    let db = match db.try_read() {
        Ok(db) => db,
        Err(e) => return LocationError::from(e).into(),
    };
    let langs: Vec<_> = locales.iter().map(|l| l.to_string()).collect();
    match db.lookup::<geoip2::City>(*address) {
        Ok(geolocation) => HttpResponse::Ok().json(DebugResponse {
            client_ip: ip.ip(),
            ip: *address,
            locales: langs,
            response,
            error,
            geolocation,
        }),
        Err(e) => LocationError::from(e).into(),
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
            fs::read(&filepath)?
        }
    };
    match maxminddb::Reader::from_source(body) {
        Ok(db) => Ok(db),
        Err(e) => bail!("failed to parse GeoIP database from buffer: {:?}", e),
    }
}

async fn metrics_handler() -> impl Responder {
    ""
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

#[derive(clap::Parser, Debug)]
pub(crate) struct Cli {
    /// File path for GeoIP or GeoLite2 database file
    #[arg(short, long, env = "GEOIP_DB_FILE")]
    file: String,
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

    let geoip_db = match open_mmdb(args.file).await {
        Ok(db) => db,
        Err(e) => {
            return Err(io::Error::new(io::ErrorKind::Other, e.to_string()));
        }
    };
    let langs = geoip_db.metadata.languages.clone();
    let db = web::Data::new(GeoIpDB::new(geoip_db));

    let prometheus = PrometheusMetricsBuilder::new("")
        .endpoint("/metrics")
        .build()
        .unwrap();

    let origin = args.allowed_origin;
    HttpServer::new(move || {
        let app = App::new()
            // The prometheus client will record hits to the '/metrics' endpoint
            // as hits to '/{address}' if these is no route for /metrics. Must
            // be added before all other routes.
            .service(web::resource("/metrics").to(metrics_handler))
            .wrap(prometheus.clone())
            .wrap(logging::AutoLog::new(log.clone()))
            .wrap(middleware::NormalizePath::trim());

        app.app_data(web::Data::clone(&db))
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
