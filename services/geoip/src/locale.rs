use std::pin::Pin;
use std::{cmp::Ordering, io};

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

#[derive(Debug, Clone, Default)]
pub(crate) struct Language(pub String);

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

static DEFAULT_LOCALE_Q: f32 = 1.0;

fn parse_locale(raw: &str) -> Locale {
    let mut l = Locale::default();
    let name;
    match raw.split_once(";") {
        Some((n, q)) => {
            name = n.trim();
            l.q = q
                .trim()
                .strip_prefix("q=") // TODO handle spaces
                .and_then(|p| p.parse::<f32>().ok())
                .unwrap_or(DEFAULT_LOCALE_Q);
        }
        None => {
            name = raw;
            l.q = DEFAULT_LOCALE_Q;
        }
    };

    match name.split_once('-') {
        Some((name, region)) => {
            l.name = name.to_string();
            l.region = region.to_string();
        }
        None => {
            l.name = name.to_string();
        }
    };
    return l;
}

fn get_locales(req: &HttpRequest) -> Locales {
    if let Ok(q) = web::Query::<LanguageQuery>::from_query(req.query_string()) {
        Locales(vec![Locale {
            name: q.lang.clone(),
            region: "".to_string(),
            q: DEFAULT_LOCALE_Q,
        }])
    } else {
        let mut l: Vec<_> = get_accept_language(req)
            .unwrap_or(DEFAULT_LANGUAGE_CODE)
            .split(",")
            .map(|l| l.trim())
            .map(parse_locale)
            .collect();
        l.sort_by(|a, b| b.compare(a));
        Locales(l)
    }
}

#[derive(Debug)]
pub(crate) struct Locales(Vec<Locale>);

impl Locales {
    #[inline]
    pub fn iter(&self) -> std::slice::Iter<'_, Locale> {
        self.0.iter()
    }
}

impl FromRequest for Locales {
    type Error = io::Error;
    type Future = Pin<Box<dyn core::future::Future<Output = Result<Self, Self::Error>>>>;

    fn from_request(req: &HttpRequest, _: &mut actix_web::dev::Payload) -> Self::Future {
        let l = get_locales(req);
        Box::pin(async move { Ok(l) })
    }
}

#[derive(Debug, Default, Clone, PartialEq)]
pub(crate) struct Locale {
    pub name: String,
    pub region: String,
    pub q: f32,
}

impl Locale {
    pub fn full_name(&self) -> String {
        let mut key = String::new();
        key.push_str(&self.name);
        if self.has_region() {
            key.push('-');
            key.push_str(&self.region);
        }
        return key;
    }

    #[inline]
    pub fn has_region(&self) -> bool {
        self.region.len() > 0
    }

    #[inline]
    fn compare(&self, other: &Self) -> Ordering {
        self.partial_cmp(other).unwrap_or(Ordering::Equal)
    }
}

impl std::string::ToString for Locale {
    fn to_string(&self) -> String {
        let mut s = String::new();
        s.push_str(&self.name);
        if self.has_region() {
            s.push('-');
            s.push_str(&self.region);
        }
        s.push_str(";q=");
        s.push_str(&self.q.to_string());
        return s;
    }
}

impl From<&str> for Locale {
    #[inline]
    fn from(value: &str) -> Self {
        parse_locale(value)
    }
}

impl FromRequest for Locale {
    type Error = io::Error;
    type Future = Pin<Box<dyn core::future::Future<Output = Result<Self, Self::Error>>>>;

    fn from_request(req: &HttpRequest, _: &mut actix_web::dev::Payload) -> Self::Future {
        let l = get_locales(req).0;
        let locale = if l.len() < 1 {
            Locale {
                name: DEFAULT_LANGUAGE_CODE.to_string(),
                region: "".to_string(),
                q: DEFAULT_LOCALE_Q,
            }
        } else {
            l[0].clone()
        };
        Box::pin(async move { Ok(locale) })
    }
}

impl PartialOrd<Locale> for Locale {
    fn partial_cmp(&self, other: &Self) -> Option<Ordering> {
        let res = if self.q > other.q {
            Ordering::Greater
        } else if self.q < other.q {
            Ordering::Less
        } else {
            Ordering::Equal
        };
        Some(res)
    }
}

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    fn test_parse_locale() {
        let mut l = parse_locale("en-US");
        assert_eq!(l.name, "en");
        assert_eq!(l.region, "US");
        assert_eq!(l.q, 1.0);

        l = parse_locale("en-GB; q=0.5");
        assert_eq!(l.name, "en");
        assert_eq!(l.region, "GB");
        assert_eq!(l.q, 0.5);

        l = parse_locale("en-GB ;q=0.8");
        assert_eq!(l.name, "en");
        assert_eq!(l.region, "GB");
        assert_eq!(l.q, 0.8);

        l = parse_locale("en-GB-1998;q=0.7");
        assert_eq!(l.name, "en");
        assert_eq!(l.region, "GB-1998");
        assert_eq!(l.q, 0.7);

        l = parse_locale("ja");
        assert_eq!(l.name, "ja");
        assert_eq!(l.region, "");
        assert_eq!(l.q, 1.0);
    }

    #[actix_web::test]
    async fn test_locales() {
        use actix_web::{http, test, App, HttpResponse};
        let app = test::init_service(
            App::new()
                .service(web::resource("/").to(|locs: Locales| {
                    assert_eq!(locs.0.len(), 3);
                    let mut prev_q = 1.1;
                    for l in locs.iter() {
                        assert!(l.q < prev_q); // the list should be sorted in desc order
                        assert_eq!(&l.name, "en");
                        if l.has_region() {
                            assert!(l.region.len() > 0);
                            assert!(l.region == "US" || l.region == "GB");
                        }
                        prev_q = l.q;
                    }
                    HttpResponse::Ok()
                }))
                .service(web::resource("/test").to(|l: Locale| {
                    println!("{:?}", l);
                    assert_eq!(l.name, "en");
                    assert_eq!(l.region, "US");
                    assert_eq!(l.q, 0.9);
                    HttpResponse::Ok()
                })),
        )
        .await;
        let lang = ("Accept-Language", "en-GB; q=0.3, en-US ; q=0.9, en ;q=0.4");
        let mut req = test::TestRequest::default()
            .uri("/")
            .insert_header(lang)
            .to_request();
        let mut res = test::call_service(&app, req).await;
        assert_eq!(res.response().status(), http::StatusCode::OK);
        req = test::TestRequest::default()
            .uri("/")
            .insert_header(lang)
            .to_request();
        res = test::call_service(&app, req).await;
        assert_eq!(res.response().status(), http::StatusCode::OK);
    }
}
