use actix_web::{FromRequest, HttpRequest};
use std::io::{self, ErrorKind};
pub mod accept;
pub use self::accept::MimeItem;

#[derive(Debug)]
pub enum Accept {
    None,
    Any,
    PlainText,
    Json,
    Html,
    Xml,
}

impl Accept {
    fn from_header(v: &str) -> Self {
        if v.starts_with("application/json") {
            Self::Json
        } else if v.starts_with("text/plain") {
            Self::PlainText
        } else if v.starts_with("text/html") {
            Self::Html
        } else if v.starts_with("application/xml") {
            Self::Xml
        } else if v.starts_with("*/*") {
            Self::Any
        } else {
            Self::None
        }
    }
}

impl FromRequest for Accept {
    type Error = io::Error;
    type Future = std::future::Ready<Result<Accept, Self::Error>>;

    fn from_request(req: &HttpRequest, _payload: &mut actix_web::dev::Payload) -> Self::Future {
        let acc = match req.headers().get(actix_web::http::header::ACCEPT) {
            None => Ok(Self::None),
            Some(val) => match val.to_str() {
                Ok(v) => Ok(Self::from_header(v)),
                Err(e) => Err(io::Error::new(ErrorKind::NotFound, e)),
            },
        };
        std::future::ready(acc)
    }
}
