mod client;
mod link;
mod nanoid;

use actix_web::{
    web::{self, Data, Form, Json, Path},
    HttpResponse,
};
use askama::Template;
use clap::{Args, Parser as CliParser, Subcommand};
use redis::ConnectionInfo;
use serde_derive::{Deserialize, Serialize};
use std::io;

use actixutil_headers::Accept;

#[cfg(debug_assertions)]
const DEFAULT_CLIENT_URL: &'static str = "http://localhost:8088";
#[cfg(not(debug_assertions))]
const DEFAULT_CLIENT_URL: &'static str = "https://l.hrry.me";

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
    #[arg(long, env, default_value = DEFAULT_CLIENT_URL)]
    client_url: String,
    #[arg(long, short, env, default_value_t = log::Level::Info)]
    pub(crate) log_level: log::Level,
    #[arg(long, env, default_value_t = flog::Format::LogFmt)]
    pub(crate) log_format: flog::Format,
    #[command(subcommand)]
    command: CliCommands,
}

#[derive(Args, Debug)]
struct Server {
    /// Number of worker threads
    #[arg(short, long, default_value_t = 6, env = "SERVER_WORKERS")]
    workers: usize,
    /// The domain name that this server will receive requests on.
    #[arg(long, default_value = "localhost", env = "SERVER_DOMAIN")]
    domain: String,
    #[arg(long, default_value = "http://localhost:8088", env = "SERVER_BASE")]
    base: String,
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
    /// List all the saved links by connecting to redis.
    List,
    /// Open the url in a browser given the ID.
    Open {
        #[arg()]
        id: String,
    },
    #[command(hide = true)]
    Test,
}

struct ServerConfig {
    base: String,
}

impl Cli {
    fn redis(&self) -> Result<redis::Client, io::Error> {
        let addr = match dns_lookup::lookup_host(&self.redis_host) {
            Ok(addrs) => {
                if addrs.is_empty() {
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

#[derive(Serialize, Deserialize, Debug)]
struct LinkResponse {
    url: String,
    id: String,
}

#[derive(Template)]
#[template(path = "create.html")]
struct CreateLinkTemplate {
    title: String,
}

#[derive(Template)]
#[template(path = "link.html")]
struct NewLinkTemplate {
    title: String,
    id: String,
    url: String,
    server: String,
}

async fn link_create_post(
    store: Data<link::Store>,
    sc: Data<ServerConfig>,
    Json(req): Json<link::CreateRequest>,
    accept: Accept,
) -> actix_web::Result<HttpResponse> {
    let id = store.create(&req).await?;
    log::info!(accept = format!("{:?}", accept); "run handler for {id}");
    Ok(match accept {
        Accept::None | Accept::Any | Accept::PlainText => HttpResponse::Ok().body(id),
        Accept::Json => HttpResponse::Ok().json(LinkResponse { url: req.url, id }),
        Accept::Html => {
            let tmpl = NewLinkTemplate {
                title: "New Link".into(),
                id,
                url: req.url,
                server: sc.base.clone(),
            };
            match tmpl.render() {
                Err(e) => HttpResponse::InternalServerError().body(e.to_string()),
                Ok(contents) => HttpResponse::Ok().content_type("text/html").body(contents),
            }
        }
        _ => HttpResponse::BadRequest().body("bad accept header"),
    })
}

async fn link_create_form(
    store: Data<link::Store>,
    sc: Data<ServerConfig>,
    Form(form): Form<link::CreateRequest>,
    accept: Accept,
) -> actix_web::Result<NewLinkTemplate> {
    let id = store.create(&form).await?;
    log::info!(accept = format!("{:?}", accept); "run handler for {id}");
    let tmpl = NewLinkTemplate {
        title: "New Link".into(),
        id,
        url: form.url,
        server: sc.base.clone(),
    };
    Ok(tmpl)
}

async fn link_info(
    store: Data<link::Store>,
    sc: Data<ServerConfig>,
    _accept: Accept,
    id: Path<String>,
) -> actix_web::Result<NewLinkTemplate> {
    let link = store.get(&id).await?;
    Ok(NewLinkTemplate {
        title: "New Link".into(),
        id: id.to_string(),
        url: link.url,
        server: sc.base.clone(),
    })
}

async fn create_link_page() -> actix_web::Result<HttpResponse> {
    let tmpl = CreateLinkTemplate {
        title: "New Link".into(),
    };
    Ok(match tmpl.render() {
        Err(e) => HttpResponse::InternalServerError().body(e.to_string()),
        Ok(contents) => HttpResponse::Ok().content_type("text/html").body(contents),
    })
}

async fn link_get(
    store: Data<link::Store>,
    id: Path<String>,
) -> actix_web::error::Result<HttpResponse> {
    let link = store.get(&id).await?;
    Ok(HttpResponse::TemporaryRedirect()
        .insert_header(("Location", link.url.clone()))
        .finish())
}

async fn link_del(
    store: Data<link::Store>,
    id: Path<String>,
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

fn configure() -> impl Fn(&mut web::ServiceConfig) {
    return |cfg| {
        cfg.route("/new", web::post().to(link_create_form))
            .route("/info/{id}", web::get().to(link_info))
            .route("/", web::post().to(link_create_post))
            .route("/", web::get().to(create_link_page))
            .service(
                web::resource("/{id}")
                    .route(web::get().to(link_get))
                    .route(web::delete().to(link_del)),
            );
    };
}

async fn server(args: &Cli, server: &Server) -> Result<(), io::Error> {
    use actix_web::{App, HttpServer};
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
    let base = server.base.clone();

    log::info!(host = args.host, port = args.port; "starting server at {}:{}", args.host, args.port);
    HttpServer::new(move || {
        let base = base.clone();
        let app = App::new()
            // The prometheus client will record hits to the '/metrics' endpoint
            // as hits to '/{some_variable}' if these is no route for /metrics. Must
            // be added before all other routes.
            .service(web::resource("/metrics").to(HttpResponse::Ok))
            .wrap(prometheus.clone())
            .wrap(actix_request_logger::RequestLogger)
            .wrap(actix_web::middleware::NormalizePath::trim());
        app.app_data(Data::new(store.clone()))
            .app_data(Data::new(ServerConfig { base }))
            .configure(configure())
    })
    .workers(server.workers)
    .bind((args.host.clone(), args.port))?
    .run()
    .await
}

#[actix_web::main]
async fn main() -> Result<(), io::Error> {
    let args = Cli::parse_from(std::env::args());
    flog::Config::new()
        .format(args.log_format)
        .level(args.log_level)
        .load_env()
        .init()
        .map_err(|e| io::Error::new(io::ErrorKind::Other, e))?;
    let client_url: url::Url = args
        .client_url
        .parse()
        .map_err(|e| io::Error::new(io::ErrorKind::Other, e))?;

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
            println!(
                "status:   {} {:?}",
                status.as_str(),
                status.canonical_reason().unwrap_or("unknown status code")
            );
            println!("location: {}", loc);
        }
        CliCommands::Del { id } => {
            let client = client::Client::new(client_url);
            client.del(id.clone()).await?;
        }
        CliCommands::List => {
            let rd = args.redis()?;
            log_connection_info(rd.get_connection_info());
            rd.get_connection().map_err(|e| {
                io::Error::new(
                    io::ErrorKind::ConnectionRefused,
                    format!("could not connect to redis: {}", e),
                )
            })?;
            let store = link::Store::new(String::new(), rd);
            let links = store
                .list()
                .await
                .map_err(|e| io::Error::new(io::ErrorKind::Other, e.to_string()))?;
            for l in links {
                println!("{} -> {} {:?}", l.key, l.link.url, l.expires);
            }
        }
        CliCommands::Open { id } => {
            use std::os::unix::process::CommandExt;
            let client = client::Client::new(client_url);
            let (loc, _) = client.get(id.clone()).await?;
            let err = std::process::Command::new("xdg-open").args([&loc]).exec();
            println!("{:?}", err);
        }
        CliCommands::Test => {}
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

    #[test]
    fn test_dns_lookup() {
        use dns_lookup::lookup_host;
        let h = lookup_host("localhost").unwrap();
        println!("{:?}", h);
        // let addrs = "localhost".to_socket_addrs().unwrap();
        // println!("{:?}", addrs);
    }
}
