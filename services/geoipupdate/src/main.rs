#![allow(dead_code)]

use std::io::{Error, ErrorKind};
use std::str::FromStr;

use aws_sdk_s3::config::Region;
use base64::{engine::general_purpose::STANDARD as b64, Engine as _};
use clap::Parser;

#[derive(Clone, Debug)]
struct Strings<const S: char>(Vec<String>);

impl<const S: char> FromStr for Strings<S> {
    type Err = Error;
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        Ok(Self(s.split(S).map(|s| s.to_owned()).collect()))
    }
}

impl<const S: char> Strings<S> {
    #[inline]
    pub fn iter(&self) -> std::slice::Iter<'_, String> {
        self.0.iter()
    }
}

impl<const S: char> IntoIterator for Strings<S> {
    type Item = String;
    type IntoIter = std::vec::IntoIter<Self::Item>;
    fn into_iter(self) -> Self::IntoIter {
        self.0.into_iter()
    }
}

#[derive(Clone, Debug)]
enum Output {
    Dir,
    S3,
}

impl Output {
    fn to_str<'a>(&self) -> &'a str {
        match self {
            Self::Dir => "dir",
            Self::S3 => "s3",
        }
    }
}

impl FromStr for Output {
    type Err = Error;
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        let o = s.trim().to_lowercase();
        match o.as_str() {
            "s3" => Ok(Self::S3),
            "dir" => Ok(Self::Dir),
            _ => Err(Error::new(
                std::io::ErrorKind::InvalidInput,
                "invalid output name",
            )),
        }
    }
}

impl std::fmt::Display for Output {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.to_str())
    }
}

impl Default for Output {
    fn default() -> Self {
        Self::S3
    }
}

// Allows clap help message to output all the possible values.
impl clap::ValueEnum for Output {
    fn value_variants<'a>() -> &'a [Self] {
        &[Self::S3, Self::Dir]
    }

    fn to_possible_value(&self) -> Option<clap::builder::PossibleValue> {
        let s = clap::builder::Str::from(self.to_str());
        Some(clap::builder::PossibleValue::new(s))
    }
}

// So we can pretend to be the official geoipupdate program.
const VERSION: &str = "5.1.1";

const _EDITION_IDS: [&str; 6] = [
    "GeoLite2-ASN",
    "GeoLite2-ASN-CSV",
    "GeoLite2-City",
    "GeoLite2-City-CSV",
    "GeoLite2-Country",
    "GeoLite2-Country-CSV",
];

#[derive(clap::Parser, Debug, Clone)]
struct Config {
    #[arg(
        long,
        short = 'H',
        env = "GEOIPUPDATE_HOST",
        default_value = "updates.maxmind.com"
    )]
    host: String,
    #[arg(long, env = "GEOIPUPDATE_ACCOUNT_ID")]
    account_id: i32,
    #[arg(long, env = "GEOIPUPDATE_LICENSE_KEY")]
    license_key: String,
    #[arg(long, env = "GEOIPUPDATE_EDITION_IDS")]
    edition_ids: Strings<','>,
    #[arg(long, env = "GEOIPUPDATE_S3_BUCKET", default_value = "geoip")]
    s3_bucket: String,
    /// Write the database files to either s3 or the local file system.
    #[arg(long, short, default_value_t = Output::default())]
    output: Output,
    #[arg(long, short, default_value_t = false)]
    test: bool,
}

struct S3Writer {
    client: aws_sdk_s3::Client,
    bucket: String,
}

impl S3Writer {
    fn write<T>(&self, _path: T) -> std::io::Result<bool>
    where
        T: AsRef<std::path::Path>,
    {
        todo!("write mmdb to s3 object")
    }
}

fn auth(config: &Config) -> String {
    format!(
        "Basic {}",
        b64.encode(format!("{}:{}", config.account_id, config.license_key))
    )
}

async fn s3_client() -> aws_sdk_s3::Client {
    let region = Region::new(match std::env::var("AWS_REGION") {
        Ok(v) => v,
        Err(_) => "us-east-1".to_string(), // MinIO default region
    });
    let mut loader = aws_config::from_env().region(region);
    if let Ok(endpoint) = std::env::var("AWS_ENDPOINT_URL") {
        loader = loader.endpoint_url(endpoint);
    }
    let builder = aws_sdk_s3::config::Builder::from(&loader.load().await).force_path_style(true);
    let s3 = aws_sdk_s3::Client::from_conf(builder.build());
    s3
}

async fn object_exists<T1, T2>(client: &aws_sdk_s3::Client, bucket: T1, key: &String) -> bool
where
    String: From<T1> + From<T2>,
{
    client
        .head_object()
        .bucket(bucket)
        .key(key)
        .send()
        .await
        .map_or(false, |_o| true)
}

async fn print_exists<T>(client: &aws_sdk_s3::Client, bucket: T, edition: T) -> bool
where
    String: From<T>,
{
    let now = "2023-04-27";
    // let now = chrono::offset::Local::now().date_naive();
    let id: String = edition.into();
    let key = format!("{now}/{}.mmdb", id);
    let exists = object_exists(client, bucket, &key).await;
    if exists {
        println!("object {key} exists");
    } else {
        println!("object {key} does not exist");
    }
    exists
}

const ZERO_MD5: &str = "00000000000000000000000000000000";

#[derive(Debug)]
struct Blob<'a> {
    body: hyper::Body,
    length: Option<usize>,
    md5: String,
    edition: &'a str,
}

struct Downloader {
    config: Config,
    client: hyper::Client<hyper_rustls::HttpsConnector<hyper::client::HttpConnector>, hyper::Body>,
    auth: String,
}

impl Downloader {
    fn new(config: Config) -> Self {
        let https = hyper_rustls::HttpsConnectorBuilder::new()
            .with_native_roots()
            .https_or_http()
            .enable_http1()
            .enable_http2()
            .build();
        let client = hyper::Client::builder()
            .http2_max_frame_size(Some((1 << 16) - 1))
            .http2_only(false)
            .build::<_, hyper::Body>(https);
        let auth = auth(&config);
        Self {
            config,
            client,
            auth,
        }
    }

    async fn download_database<'a>(&self, edition: &'a str) -> Result<Blob<'a>, Error> {
        let req = self.db_request(edition)?;
        match self.client.request(req).await {
            Ok(res) => {
                if !res.status().is_success() {
                    return Err(Error::new(
                        ErrorKind::Other,
                        format!("couldn't find {edition}"),
                    ));
                }
                let (parts, body) = res.into_parts();
                let l: Option<usize> = parts
                    .headers
                    .get("content-length")
                    .and_then(|l| l.to_str().ok())
                    .and_then(|l| l.parse().ok());
                let md5: String = parts
                    .headers
                    .get("x-database-md5")
                    .and_then(|h| Some(String::from(h.to_str().unwrap_or(""))))
                    .unwrap_or(String::new());
                Ok(Blob {
                    length: l,
                    body,
                    md5,
                    edition,
                })
            }
            Err(e) => Err(Error::new(ErrorKind::Other, e.to_string())),
        }
    }

    fn db_request(&self, edition: &str) -> Result<hyper::Request<hyper::Body>, Error> {
        let pb: std::path::PathBuf = ["/geoip/databases", edition, "update"].iter().collect();
        let Some(path) = pb.to_str() else {
            return Err(Error::new(ErrorKind::Other, "failed to build uri path"));
        };
        let uri = match hyper::Uri::builder()
            .scheme("https")
            .authority(self.config.host.to_owned())
            .path_and_query(path)
            .build()
        {
            Ok(u) => u,
            Err(e) => return Err(Error::new(ErrorKind::Other, e.to_string())),
        };
        let req = match hyper::Request::builder()
            .method("GET")
            .uri(&uri)
            .header("Authorization", &self.auth)
            .header("User-Agent", format!("geoipupdate/{VERSION}")) // pretending we are the official geoipupdate
            .body(hyper::Body::empty())
        {
            Ok(r) => r,
            Err(e) => return Err(Error::new(ErrorKind::Other, e.to_string())),
        };
        Ok(req)
    }
}

#[tokio::main(flavor = "current_thread")]
async fn main() {
    let args = Config::parse_from(std::env::args());
    let s3 = s3_client().await;
    let dl = Downloader::new(args.clone());

    if args.test {
        println!("{:?}", args);
        // let results = args.edition_ids.0.iter().map(async |id| )
        let results = args
            .edition_ids
            .iter()
            .map(|id| print_exists(&s3, &args.s3_bucket, id));
        futures::future::join_all(results).await;

        let now = chrono::offset::Local::now().date_naive();
        println!("{now}");

        match s3.list_objects_v2().bucket(&args.s3_bucket).send().await {
            Ok(res) => {
                // println!("{res:?}");
                println!("count: {}", res.key_count());
                let objects = res.contents().unwrap_or(&[]);
                for o in objects {
                    println!("{o:?}");
                }
            }
            Err(e) => println!("Error getting bucket location: {e}"),
        }
        return;
    }

    let editions = args.edition_ids.iter().filter(|id| !id.is_empty());
    let jobs = editions.map(|id| dl.download_database(id));
    let job_results = futures::future::join_all(jobs).await;
    // let (results, errors): (Vec<_>, Vec<_>) = job_results.into_iter().partition(Result::is_ok);
    // println!(
    //     "{:?}",
    //     results.into_iter().map(Result::unwrap).collect::<Vec<_>>()
    // );
    // println!(
    //     "{:?}",
    //     errors
    //         .into_iter()
    //         .map(Result::unwrap_err)
    //         .collect::<Vec<_>>()
    // );
    for res in job_results {
        match res {
            Ok(r) => println!("{r:?}"),
            Err(e) => println!("Error: {e}"),
        };
    }

    // for res in results {
    //     println!("{:?}", res.length);
    // }
    // let r: Result<(), Error> = Ok(());
    // for r in result
}
