use std::error::Error;

use anyhow::bail;
use aws_sdk_s3::Credentials;
use aws_sdk_s3::Region;

fn credentials_from_url(u: &url::Url) -> Option<Credentials> {
    if let Some(pw) = u.password() {
        Some(Credentials::new(u.username(), pw, None, None, ""))
    } else {
        None
    }
}

fn get_endpoint(url: &url::Url) -> Option<impl Into<String>> {
    if let Some(domain) = url.domain() {
        let endpoint_scheme = match std::env::var("S3_ALLOW_INSECURE") {
            Ok(v) => match v.as_str() {
                "true" => "http",
                _ => "https",
            },
            Err(_) => "https",
        };
        let port = match url.port() {
            None => String::new(),
            Some(p) => format!(":{}", p),
        };
        let endpoint = format!("{}://{}{}", endpoint_scheme, domain, port);
        log::info!("using s3 endpoint \"{}\"", endpoint);
        return Some(endpoint);
    } else {
        return None;
    }
}

pub(crate) async fn open_from_s3(url: &url::Url) -> anyhow::Result<Vec<u8>> {
    let mut loader =
        aws_config::from_env().region(Region::new(match std::env::var("AWS_REGION") {
            Ok(v) => v,
            Err(_) => "us-east-1".to_string(), // MinIO default
        }));
    if let Some(endpoint) = get_endpoint(url) {
        loader = loader.endpoint_url(endpoint);
    }
    // Check for aws credentials in the url
    if let Some(credentials) = credentials_from_url(url) {
        loader = loader.credentials_provider(credentials);
    }
    let builder = aws_sdk_s3::config::Builder::from(&loader.load().await).force_path_style(true);
    let client = aws_sdk_s3::Client::from_conf(builder.build());
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
