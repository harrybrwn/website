use actix_web::{FromRequest, HttpRequest};
use core::future::Future;
use std::io::{self, ErrorKind};
pub mod accept;
pub use self::accept::MimeItem;

pub enum Accept {
    None,
    Any,
    PlainText,
    Json,
    Html,
    Xml,
}

impl FromRequest for Accept {
    type Error = io::Error;
    type Future = std::pin::Pin<Box<dyn Future<Output = Result<Self, Self::Error>>>>;

    fn from_request(req: &HttpRequest, _payload: &mut actix_web::dev::Payload) -> Self::Future {
        let acc = match req.headers().get(actix_web::http::header::ACCEPT) {
            None => Ok(Self::None),
            Some(val) => match val.to_str() {
                Ok(v) => Ok(if v.starts_with("application/json") {
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
                }),
                Err(e) => Err(io::Error::new(ErrorKind::NotFound, e)),
            },
        };
        Box::pin(async move { acc })
    }
}
