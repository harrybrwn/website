use actix_web::{FromRequest, HttpRequest};
use core::future::Future;
use std::cmp::Ordering;
use std::io::{self, ErrorKind};

use mime::Mime;

pub struct Accept(Vec<MimeItem>);

impl FromRequest for Accept {
    type Error = io::Error;
    type Future = std::pin::Pin<Box<dyn Future<Output = Result<Self, Self::Error>>>>;

    fn from_request(req: &HttpRequest, _payload: &mut actix_web::dev::Payload) -> Self::Future {
        let acc = match req.headers().get(actix_web::http::header::ACCEPT) {
            None => Ok(Self(vec![MimeItem::default()])),
            Some(val) => match val.to_str() {
                Ok(v) => Ok(Self::from(v)),
                Err(e) => Err(Self::Error::new(ErrorKind::NotFound, e)),
            },
        };
        Box::pin(async move { acc })
    }
}

impl From<&str> for Accept {
    fn from(v: &str) -> Self {
        Self::from_iter(v.split(',').map(|v| v.trim()))
    }
}

impl From<MimeItem> for Accept {
    fn from(v: MimeItem) -> Self {
        Self(vec![v])
    }
}

impl<T> From<Vec<T>> for Accept
where
    T: Into<MimeItem>,
{
    fn from(v: Vec<T>) -> Self {
        Self::from_iter(v)
    }
}

impl<T> From<&[T]> for Accept
where
    T: Into<MimeItem> + Clone,
{
    fn from(v: &[T]) -> Self {
        Self::from_iter(v.into_iter().map(|v| v.clone()))
    }
}

impl<M> FromIterator<M> for Accept
where
    M: Into<MimeItem>,
{
    fn from_iter<T: IntoIterator<Item = M>>(iter: T) -> Self {
        use mime::{SubType, Type};
        let mut v: Vec<_> = iter
            .into_iter()
            .map(|v| v.into())
            .filter(|m| m.mime.typ != Type::None && m.mime.sub != SubType::None)
            .collect();
        v.sort_by(|a, b| a.partial_cmp(b).unwrap_or(Ordering::Equal));
        Self(v)
    }
}

impl Accept {
    pub fn empty(&self) -> bool {
        self.0.len() > 0
    }

    pub fn has(&self, mime: Mime) -> bool {
        for item in self.0.iter() {
            if item.mime.matches(&mime) {
                return true;
            }
        }
        return false;
    }

    #[inline]
    pub fn iter(&self) -> std::slice::Iter<'_, MimeItem> {
        self.0.iter()
    }
}

impl IntoIterator for Accept {
    type Item = MimeItem;
    type IntoIter = std::vec::IntoIter<Self::Item>;
    #[inline]
    fn into_iter(self) -> Self::IntoIter {
        self.0.into_iter()
    }
}

#[derive(Debug, Clone)]
pub struct MimeItem {
    pub mime: Mime,
    pub q: f32,
}

impl Default for MimeItem {
    #[inline]
    fn default() -> Self {
        Self::from(Mime::default())
    }
}

static DEFAULT_Q: f32 = 1.0;

impl From<&str> for MimeItem {
    fn from(value: &str) -> Self {
        match value.split_once(';') {
            Some((name, q)) => Self {
                mime: Mime::from(name.trim()),
                q: q.trim()
                    .strip_prefix("q=")
                    .and_then(|p| p.parse::<f32>().ok())
                    .unwrap_or(DEFAULT_Q),
            },
            None => Self::from(Mime::from(value)),
        }
    }
}

impl From<String> for MimeItem {
    fn from(value: String) -> Self {
        Self::from(value.as_str())
    }
}

impl From<&&str> for MimeItem {
    fn from(value: &&str) -> Self {
        Self::from(*value)
    }
}

impl From<Mime> for MimeItem {
    fn from(mime: Mime) -> Self {
        Self { mime, q: DEFAULT_Q }
    }
}

impl PartialEq for MimeItem {
    fn eq(&self, other: &Self) -> bool {
        self.q == other.q
    }
}

impl PartialOrd for MimeItem {
    fn partial_cmp(&self, other: &Self) -> Option<std::cmp::Ordering> {
        Some(if self.q < other.q {
            std::cmp::Ordering::Less
        } else {
            std::cmp::Ordering::Greater
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn it_works() {
        assert!(true);
    }

    #[test]
    fn multi_value() {
        let value =
            "text/html, application/xhtml+xml, application/xml;q=0.9, image/webp, */*;q=0.8";
        let values = Accept::from(value).0;
        assert_eq!(values[0].mime.typ, mime::Type::Text);
        assert_eq!(values[0].mime.sub, mime::SubType::Html);
        assert_eq!(values[0].q, 1.0);
        assert_eq!(values[1].mime.typ, mime::Type::Application);
        assert_eq!(values[1].mime.sub, mime::SubType::Xhtml);
        assert_eq!(values[1].q, 1.0);
        assert_eq!(values[2].mime.typ, mime::Type::Application);
        assert_eq!(values[2].mime.sub, mime::SubType::Xml);
        assert_eq!(values[2].q, 0.9);
        assert_eq!(values[3].mime.typ, mime::Type::Image);
        assert_eq!(values[3].mime.sub, mime::SubType::Webp);
        assert_eq!(values[3].q, 1.);
        assert_eq!(values[4].mime.typ, mime::Type::Any);
        assert_eq!(values[4].mime.sub, mime::SubType::Any);
        assert_eq!(values[4].q, 0.8);
        println!("{:?}", values);
    }
}
