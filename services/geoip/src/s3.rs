use std::error::Error;

use anyhow::bail;
use aws_sdk_s3::config;
use aws_sdk_s3::Credentials;
use aws_sdk_s3::Region;

fn aws_env_credentials() -> Result<Credentials, std::env::VarError> {
    use std::env::var;
    Ok(Credentials::new(
        var("AWS_ACCESS_KEY_ID")?,
        var("AWS_SECRET_ACCESS_KEY")?,
        match var("AWS_SESSION_TOKEN") {
            Err(_) => None,
            Ok(v) => Some(v),
        },
        match var("AWS_SESSION_EXPIRATION") {
            Err(_) => None,
            Ok(_) => todo!("parse aws session expiration"),
        },
        "",
    ))
}

fn s3_client_config(u: &url::Url) -> config::Builder {
    let mut builder = config::Builder::default()
        .region(Region::new(match std::env::var("AWS_REGION") {
            Ok(v) => v,
            Err(_) => "us-east-1".to_string(), // MinIO default
        }))
        .force_path_style(true);

    if let Some(domain) = u.domain() {
        let endpoint_scheme = match std::env::var("S3_ALLOW_INSECURE") {
            Ok(v) => match v.as_str() {
                "true" => "http",
                _ => "https",
            },
            Err(_) => "https",
        };
        let port = match u.port() {
            None => String::new(),
            Some(p) => format!(":{}", p),
        };
        let endpoint = format!("{}://{}{}", endpoint_scheme, domain, port);
        log::info!("using s3 endpoint \"{}\"", endpoint);
        builder = builder.endpoint_url(endpoint);
    }

    if let Ok(credentials) = aws_env_credentials() {
        log::info!("using env credentials for s3");
        builder = builder.credentials_provider(credentials);
    }

    return builder;
}

pub(crate) async fn open_from_s3(url: &url::Url) -> anyhow::Result<Vec<u8>> {
    let conf = s3_client_config(url).build();
    let client = aws_sdk_s3::Client::from_conf(conf);
    let path = match url.path().strip_prefix("/") {
        Some(p) => p,
        None => url.path(),
    };
    match client.get_object().key(path).send().await {
        Err(e) => {
            bail!(
                "failed to send request to s3: {:?} {:?} {:?}",
                e.to_string(),
                e.source(),
                e
            )
        }
        Ok(res) => match res.body.collect().await {
            Ok(res) => Ok(res.to_vec()),
            Err(e) => {
                bail!("failed to collect response from s3: {:?}", e)
            }
        },
    }
}
