mod client;
mod link;

use actix_web::{
    http::header,
    web::{self, Json},
    FromRequest, HttpRequest, HttpResponse,
};
use askama::Template;
use clap::{Args, Parser as CliParser, Subcommand};
use redis::ConnectionInfo;
use serde_derive::{Deserialize, Serialize};
use std::path;
use std::{io, net::IpAddr};

#[derive(CliParser, Debug)]
#[command(author, version, about, long_about = None)]
struct Cli {
    #[arg(long, short = 'H', default_value = "0.0.0.0", env)]
    host: String,
    #[arg(long, short, default_value_t = 8088, env)]
    port: u16,
    #[arg(long, default_value_t = 6, env)]
    redis_db: i64,
    #[arg(long, env, default_value = "127.0.0.1")]
    redis_host: String,
    #[arg(long, env, default_value_t = 6379)]
    redis_port: u16,
    #[arg(long, env)]
    redis_username: Option<String>,
    #[arg(long, env, default_value = "testbed01")]
    redis_password: Option<String>,
    #[command(subcommand)]
    command: CliCommands,
}

#[derive(Args, Debug)]
struct Server {
    /// Number of worker threads
    #[arg(short, long, default_value_t = 6, env = "SERVER_WORKERS")]
    workers: usize,
    /// URL Prefix to serve all requests from.
    #[arg(long, default_value = "l", env = "SERVER_URL_PREFIX")]
    url_prefix: String,
    /// The domain name that this server will receive requests on.
    #[arg(long, default_value = "localhost", env = "SERVER_DOMAIN")]
    domain: String,
}

#[derive(Subcommand, Debug)]
#[command(author = "")]
enum CliCommands {
    /// Run the web server
    Server(Server),
    /// Create a new link by querying the server.
    Put {
        #[arg()]
        url: String,
        #[arg(short, long, value_parser = duration_str::parse)]
        expires: Option<std::time::Duration>,
    },
    /// Get a url by querying the server.
    Get {
        #[arg()]
        id: String,
    },
    /// Delete a url by querying the server.
    Del {
        #[arg()]
        id: String,
    },
}

impl Cli {
    fn redis(&self) -> Result<redis::Client, io::Error> {
        let addr = match dns_lookup::lookup_host(&self.redis_host) {
            Ok(addrs) => {
                if addrs.len() == 0 {
                    log::warn!("dns lookup failed on {}", self.redis_host);
                    self.redis_host.clone()
                } else {
                    addrs[0].to_string()
                }
            }
            Err(e) => {
                log::warn!("dns lookup failed on {}: {}", self.redis_host, e);
                self.redis_host.clone()
            }
        };
        let client = match redis::Client::open(redis::ConnectionInfo {
            // addr: redis::ConnectionAddr::Tcp(self.redis_host.clone(), self.redis_port),
            addr: redis::ConnectionAddr::Tcp(addr, self.redis_port),
            redis: redis::RedisConnectionInfo {
                db: self.redis_db,
                username: self.redis_username.clone(),
                password: self.redis_password.clone(),
            },
        }) {
            Ok(client) => client,
            Err(e) => return Err(io::Error::new(io::ErrorKind::Other, e)),
        };
        Ok(client)
    }
}

enum Accept {
    None,
    PlainText,
    Json,
    Html,
}

impl FromRequest for Accept {
    type Error = io::Error;
    type Future = std::pin::Pin<Box<dyn core::future::Future<Output = Result<Self, Self::Error>>>>;

    fn from_request(req: &HttpRequest, _payload: &mut actix_web::dev::Payload) -> Self::Future {
        let acc = match req.headers().get(header::ACCEPT) {
            None => Ok(Self::None),
            Some(val) => match val.to_str() {
                Ok(v) => Ok(if v.starts_with("application/json") {
                    Self::Json
                } else if v.starts_with("text/plain") {
                    Self::PlainText
                } else if v.starts_with("text/html") {
                    Self::Html
                } else {
                    Self::None
                }),
                Err(e) => Err(io::Error::new(io::ErrorKind::NotFound, e)),
            },
        };
        Box::pin(async move { acc })
    }
}

#[derive(Serialize, Deserialize, Debug)]
struct LinkResponse {
    url: String,
    id: String,
}

#[derive(Template)]
#[template(path = "new.html")]
struct NewLinkTemplate {
    id: String,
    url: String,
}

async fn link_create_post(
    store: web::Data<link::Store>,
    Json(req): Json<link::CreateRequest>,
    accept: Accept,
) -> actix_web::Result<HttpResponse> {
    let id = store.create(&req).await?;
    Ok(match accept {
        Accept::None | Accept::PlainText => HttpResponse::Ok().body(id),
        Accept::Json => HttpResponse::Ok().json(LinkResponse { url: req.url, id }),
        Accept::Html => {
            let tmpl = NewLinkTemplate { id, url: req.url };
            match tmpl.render() {
                Err(e) => HttpResponse::InternalServerError().body(e.to_string()),
                Ok(contents) => HttpResponse::Ok().content_type("text/html").body(contents),
            }
        }
    })
}

async fn link_get(
    store: web::Data<link::Store>,
    id: web::Path<String>,
) -> actix_web::error::Result<HttpResponse> {
    let link = store.get(id.into_inner()).await?;
    Ok(HttpResponse::TemporaryRedirect()
        .insert_header(("Location", link.url.clone()))
        .finish())
}

async fn link_del(
    store: web::Data<link::Store>,
    id: web::Path<String>,
) -> Result<HttpResponse, actix_web::Error> {
    store.del(id.into_inner()).await?;
    Ok(HttpResponse::Ok().finish())
}

fn log_connection_info(info: &ConnectionInfo) {
    log::info!(
        "connecting to redis addr={} db={}",
        info.addr.to_string(),
        info.redis.db
    );
}

async fn server(args: &Cli, server: &Server) -> Result<(), io::Error> {
    use actix_web::{App, HttpServer};

    std::env::set_var("RUST_LOG", "debug");
    env_logger::init();

    let prometheus = actix_web_prom::PrometheusMetricsBuilder::new("")
        .endpoint("/metrics")
        .build()
        .map_err(|e| io::Error::new(io::ErrorKind::Other, e))?;
    let rd = args.redis()?;
    log_connection_info(rd.get_connection_info());
    rd.get_connection().map_err(|e| {
        io::Error::new(
            io::ErrorKind::ConnectionRefused,
            format!("could not connect to redis: {}", e),
        )
    })?;
    let store = link::Store::new(server.domain.clone(), rd);

    let mut path = path::PathBuf::from("/");
    if !server.url_prefix.is_empty() && server.url_prefix != "/" {
        path.push(server.url_prefix.clone());
    }

    log::info!("starting server at {}:{}", args.host, args.port);
    HttpServer::new(move || {
        let prefix = path.to_str().unwrap();
        log::info!("using url prefix \"{prefix}\"");
        let app = App::new()
            // The prometheus client will record hits to the '/metrics' endpoint
            // as hits to '/{some_variable}' if these is no route for /metrics. Must
            // be added before all other routes.
            .service(web::resource("/metrics").to(HttpResponse::Ok))
            .wrap(prometheus.clone())
            .wrap(actix_web::middleware::NormalizePath::trim());
        app.app_data(web::Data::new(store.clone()))
            .wrap(actix_web::middleware::Logger::default())
            .route(prefix, web::post().to(link_create_post))
            .service(
                web::resource(
                    path::PathBuf::from_iter(&[prefix, "{id}"])
                        .to_str()
                        .unwrap(),
                )
                .route(web::get().to(link_get))
                .route(web::delete().to(link_del)),
            )
    })
    .workers(server.workers)
    .bind((args.host.clone(), args.port))?
    .run()
    .await
}

#[actix_web::main]
async fn main() -> Result<(), io::Error> {
    let args = Cli::parse_from(std::env::args());
    let mut client_url: url::Url = "http://localhost:8088"
        .parse()
        .map_err(|e| io::Error::new(io::ErrorKind::Other, e))?;
    if let Ok(mut path) = client_url.path_segments_mut() {
        path.push("l");
    }

    match &args.command {
        CliCommands::Server(s) => server(&args, s).await?,
        CliCommands::Put { url, expires } => {
            let client = client::Client::new(client_url);
            let id = client
                .put(&link::CreateRequest {
                    url: url.clone(),
                    expires: expires.map_or(None, |d| Some(d.as_secs())),
                    access_limit: None,
                })
                .await?;
            println!("{id}");
        }
        CliCommands::Get { id } => {
            let client = client::Client::new(client_url);
            let (loc, status) = client.get(id.clone()).await?;
            println!("status:   {}", status);
            println!("location: {}", loc);
        }
        CliCommands::Del { id } => {
            let client = client::Client::new(client_url);
            client.del(id.clone()).await?;
        }
    };
    Ok(())
}

#[cfg(test)]
mod main_tests {
    #[test]
    fn test() {}

    #[test]
    fn test_path_joining() {
        let p = std::path::PathBuf::from_iter(&["/x/y/z", "a", "b", "c"]);
        assert_eq!("/x/y/z/a/b/c", p.to_str().unwrap());
    }

    #[actix_web::test]
    async fn test_put() {}

    fn resolve() {}

    #[test]
    fn test_dns_lookup() {
        use dns_lookup::lookup_host;
        use std::net::{SocketAddr, ToSocketAddrs};
        let h = lookup_host("localhost").unwrap();
        println!("{:?}", h);
        // let addrs = "localhost".to_socket_addrs().unwrap();
        // println!("{:?}", addrs);
    }
}
