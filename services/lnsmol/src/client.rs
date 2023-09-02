use std::io::{self, Error, ErrorKind};
use url::Url;

pub(crate) struct Client {
    base: Url,
    client: hyper::Client<
        hyper_rustls::HttpsConnector<hyper::client::HttpConnector>,
        hyper::body::Body,
    >,
}

impl Client {
    pub fn new(base: Url) -> Self {
        let https = hyper_rustls::HttpsConnectorBuilder::new()
            .with_native_roots()
            .https_or_http()
            .enable_http1()
            .enable_http2()
            .build();
        Self {
            base,
            client: hyper::Client::builder()
                .http2_max_frame_size(Some((1 << 16) - 1))
                .http2_only(false)
                .build::<_, hyper::body::Body>(https),
        }
    }

    pub(crate) async fn put(&self, req: &crate::link::CreateRequest) -> Result<String, Error> {
        let uri = to_uri(&self.base)?;
        let req = hyper::Request::builder()
            .method("POST")
            .uri(&uri)
            .header("Accept", "application/json")
            .header("Content-Type", "application/json")
            .body(hyper::Body::from(serde_json::to_string(req)?))
            .map_err(convert_err)?;
        let res = self.client.request(req).await.map_err(convert_err)?;
        let (parts, body) = res.into_parts();
        if !parts.status.is_success() {
            return Err(Error::new(ErrorKind::InvalidData, "bad status code"));
        }
        let body_bytes = hyper::body::to_bytes(body)
            .await
            .map_err(|e| Error::new(ErrorKind::Other, e))?
            .to_vec();
        let l = serde_json::from_slice::<crate::LinkResponse>(&body_bytes).map_err(convert_err)?;
        Ok(l.id)
    }

    pub(crate) async fn get(&self, id: String) -> Result<(String, u16), io::Error> {
        let mut base = self.base.clone();
        if let Ok(mut path) = base.path_segments_mut() {
            path.push(&id);
        } else {
            return Err(Error::new(
                ErrorKind::InvalidData,
                "could failed to push url path",
            ));
        }
        let uri = to_uri(&base)?;
        let req = hyper::Request::builder()
            .method("GET")
            .uri(&uri)
            .header("Accept", "text/plain")
            .body(hyper::Body::empty())
            .map_err(convert_err)?;
        let res = self.client.request(req).await.map_err(convert_err)?;
        let (parts, _) = res.into_parts();
        if !parts.status.is_redirection() && !parts.status.is_success() {
            return Err(Error::new(ErrorKind::InvalidData, "bad status code"));
        }
        let loc = match parts.headers.get("Location") {
            Some(s) => Ok(String::from(s.to_str().map_err(convert_err)?)),
            None => Err(Error::new(
                ErrorKind::InvalidData,
                "no location header in response",
            )),
        }?;
        Ok((loc, parts.status.as_u16()))
    }

    pub(crate) async fn del(&self, id: String) -> Result<(), io::Error> {
        let mut base = self.base.clone();
        if let Ok(mut path) = base.path_segments_mut() {
            path.push(&id);
        } else {
            return Err(Error::new(
                ErrorKind::InvalidData,
                "could failed to push url path",
            ));
        }
        let uri = to_uri(&base)?;
        let req = hyper::Request::builder()
            .method("DELETE")
            .uri(&uri)
            .body(hyper::Body::empty())
            .map_err(convert_err)?;
        let res = self.client.request(req).await.map_err(convert_err)?;
        let (parts, _) = res.into_parts();
        if !parts.status.is_success() {
            return Err(Error::new(ErrorKind::InvalidData, "bad status code"));
        }
        Ok(())
    }
}

#[inline]
fn convert_err<E>(err: E) -> io::Error
where
    E: Into<Box<dyn std::error::Error + Send + Sync>>,
{
    io::Error::new(io::ErrorKind::Other, err)
}

#[inline]
fn to_uri(u: &url::Url) -> Result<hyper::Uri, io::Error> {
    hyper::Uri::builder()
        .scheme(u.scheme())
        .authority(u.authority())
        .path_and_query(u.path())
        .build()
        .map_err(convert_err)
}

#[cfg(test)]
mod client_tests {
    #[actix_web::test]
    async fn build_client() {}
}
