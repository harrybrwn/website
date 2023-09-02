use actix_web::{web, App, HttpResponse, HttpServer};
use clap::Parser;
use serde_derive::Serialize;
use std::error;
use std::io;
use std::str;
use tera::{Context, Tera};

fn prometheus() -> actix_web_prom::PrometheusMetrics {
    actix_web_prom::PrometheusMetricsBuilder::new("")
        .endpoint("/metrics")
        .build()
        .unwrap()
}

fn to_err<E>(e: E) -> io::Error
where
    E: error::Error,
{
    io::Error::new(io::ErrorKind::Other, e.to_string())
}

#[derive(Clone, Serialize)]
struct Package<'a> {
    name: &'a str,
    vcs: &'a str,
}

#[derive(Clone)]
enum RepoKind {
    Git,
    #[allow(unused)]
    Subversion,
    #[allow(unused)]
    Bazaar,
}

impl RepoKind {
    fn as_str(&self) -> &str {
        match self {
            Self::Git => "git",
            Self::Subversion => "svn",
            Self::Bazaar => "bzr",
        }
    }
}

impl serde::Serialize for RepoKind {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.collect_str(self.as_str())
    }
}

#[derive(Clone, Serialize)]
struct Repo<'a> {
    kind: RepoKind,
    domain: &'a str,
    user: &'a str,
}

#[derive(Clone, Serialize)]
struct Data<'a> {
    domain: &'a str,
    docs_url: &'a str,
    repo: Repo<'a>,
}

async fn index<'a>(
    config: web::Data<Data<'a>>,
    template: web::Data<Tera>,
    path: web::Path<String>,
) -> HttpResponse {
    let mut ctx = match Context::from_serialize(config) {
        Err(_) => return HttpResponse::InternalServerError().finish(),
        Ok(c) => c,
    };
    ctx.insert(
        "package",
        &Package {
            vcs: "git",
            name: path.as_str(),
        },
    );
    match template.render("index.html", &ctx) {
        Err(e) => {
            println!("error: {}", e);
            HttpResponse::InternalServerError().finish()
        }
        Ok(res) => HttpResponse::Ok().body(res),
    }
}

#[derive(clap::Parser, Debug)]
pub(crate) struct Cli {
    /// Server host
    #[arg(long, short = 'H', default_value = "0.0.0.0", env = "HOST")]
    host: String,
    /// Server port
    #[arg(short, long, default_value_t = 8080, env = "PORT")]
    port: u16,
    /// Concurrent workers
    #[arg(short, long, default_value_t = 4)]
    workers: usize,
}

#[actix_web::main]
async fn main() -> Result<(), io::Error> {
    std::env::set_var("RUST_LOG", "debug");
    env_logger::init();
    let args = Cli::parse_from(std::env::args());
    let prometheus = prometheus();
    let file = include_bytes!("index.html");
    let s = str::from_utf8(file).map_err(to_err)?;
    let mut tt = Tera::default();
    tt.set_escape_fn(|s| String::from(s));
    tt.add_raw_template("index.html", s).map_err(to_err)?;

    let data = Data {
        domain: "gopkg.hrry.dev",
        docs_url: "pkg.go.dev",
        repo: Repo {
            kind: RepoKind::Git,
            domain: "github.com",
            user: "harrybrwn",
        },
    };

    HttpServer::new(move || {
        let app = App::new()
            // The prometheus client will record hits to the '/metrics' endpoint
            // as hits to '/{some_variable}' if these is no route for /metrics. Must
            // be added before all other routes.
            .service(web::resource("/metrics").to(HttpResponse::Ok))
            .wrap(prometheus.clone())
            //.wrap(logging::AutoLog::new(log.clone()))
            .wrap(actix_web::middleware::NormalizePath::trim());
        app.app_data(web::Data::new(tt.clone()))
            .app_data(web::Data::new(data.clone()))
            .service(web::resource("/{path:.*}").route(web::get().to(index)))
    })
    .workers(args.workers)
    .bind(("0.0.0.0", args.port))?
    .run()
    .await
}
