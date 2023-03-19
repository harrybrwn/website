use std::io;
use std::pin::Pin;

use actix_web::{http, web, FromRequest, HttpRequest};
use serde::Deserialize;

static DEFAULT_LANGUAGE_CODE: &str = "en";

fn get_accept_language<'a>(req: &'a HttpRequest) -> Option<&'a str> {
    req.headers()
        .get(http::header::ACCEPT_LANGUAGE)?
        .to_str()
        .ok()
}

#[derive(Deserialize, Debug)]
struct LanguageQuery {
    lang: String,
}

#[derive(Debug, Clone)]
pub(crate) struct Language(pub String);

impl Language {
    #[inline]
    pub(crate) fn as_str(&self) -> &str {
        self.0.as_str()
    }
}

impl FromRequest for Language {
    type Error = io::Error;
    type Future = Pin<Box<dyn core::future::Future<Output = Result<Self, Self::Error>>>>;

    fn from_request(req: &HttpRequest, _: &mut actix_web::dev::Payload) -> Self::Future {
        if let Ok(q) = web::Query::<LanguageQuery>::from_query(req.query_string()) {
            Box::pin(async move { Ok(Language(q.lang.clone())) })
        } else {
            let l = get_accept_language(req)
                .unwrap_or(DEFAULT_LANGUAGE_CODE)
                .to_string();
            Box::pin(async move { Ok(Language(l)) })
        }
    }
}
