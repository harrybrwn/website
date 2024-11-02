use rand::SeedableRng;
use rand_hc::Hc128Rng;
use redis::aio;
use serde::de;
use std::cell::RefCell;
use url::Url;

use crate::nanoid::{calc_id_size, gen_str_id};

use actix_web::error::{
    Error, ErrorBadRequest, ErrorConflict, ErrorInternalServerError, ErrorNotFound,
};
use serde_derive::{Deserialize, Serialize};

#[derive(Deserialize, Serialize, Debug, Clone)]
pub(crate) struct CreateRequest {
    /// URL of the link being created.
    pub(crate) url: String,
    /// Number of seconds the link will live.
    #[serde(default, deserialize_with = "deserialize_option_ignore_error")]
    pub(crate) expires: Option<u64>,
    /// Number of times a link can be accessed before self destructing.
    pub(crate) access_limit: Option<u32>,
}

#[derive(Deserialize, Serialize, Debug, Clone)]
pub(crate) struct Link {
    /// URL of the link being created.
    pub(crate) url: String,
    /// Number of accesses left before a link is deleted.
    pub(crate) accesses: Option<u32>,
}

#[derive(Clone, Debug)]
pub(crate) struct LinkInfo {
    pub(crate) link: Link,
    pub(crate) key: String,
    pub(crate) expires: Option<u64>,
}

#[inline]
fn to_json<T>(item: &T) -> Result<String, Error>
where
    T: Sized + serde::ser::Serialize,
{
    serde_json::to_string(item).map_err(internal_server_error)
}

fn new_key(prefix: &str, id: &str) -> String {
    let mut key = String::from(prefix);
    key.push(':');
    key.push_str(id);
    key
}

#[derive(Clone, Debug)]
pub(crate) struct Store {
    host: String,
    rd: redis::Client,
    rng: RefCell<Hc128Rng>,
}

const DEFAULT_EXPIRATION: u64 = 60 * 60 * 24 * 7; // 1 week

impl Store {
    pub(crate) fn new(host: String, rd: redis::Client) -> Self {
        Self {
            host,
            rd,
            rng: RefCell::new(Hc128Rng::from_entropy()),
        }
    }

    #[inline]
    async fn conn(&self) -> Result<aio::ConnectionManager, actix_web::Error> {
        self.rd
            .get_tokio_connection_manager()
            .await
            .map_err(internal_server_error)
    }

    pub(crate) async fn create(&self, req: &CreateRequest) -> Result<String, Error> {
        let u = Url::parse(&req.url).map_err(ErrorBadRequest)?;
        if let Some(url::Host::Domain(d)) = u.host() {
            if d == self.host {
                return Err(ErrorConflict("cannot self link"));
            }
        }
        let mut conn = self.conn().await?;
        let count = count_incr(&mut conn).await?;
        let id_size = calc_id_size(count, 1e-9);

        let link = Link {
            url: req.url.clone(),
            accesses: req.access_limit,
        };
        let link_json = to_json(&link)?;
        let exp = req.expires.unwrap_or(DEFAULT_EXPIRATION) as usize;

        let mut rng = self.rng.borrow_mut();
        let mut i = 0;
        loop {
            let id = gen_str_id(id_size, &mut *rng);
            match redis::cmd("SET")
                .arg(new_key("link", &id))
                .arg(&link_json)
                .arg("NX")
                .arg("EX")
                .arg(exp)
                .query_async::<_, Option<String>>(&mut conn)
                .await
                .map_err(internal_server_error)?
            {
                Some(res) => {
                    log::info!("{res:?} created new link {:?}", id);
                    break Ok(id);
                }
                None => {
                    if i > 32 {
                        log::error!("failed to find new id");
                        break Err(ErrorInternalServerError("failed to create id"));
                    }
                    log::warn!(id = id; "found key collision with {id:?}");
                    i += 1;
                    continue;
                }
            }
        }
    }

    pub(crate) async fn get(&self, id: &str) -> Result<Link, Error> {
        let mut conn = self.conn().await?;
        let key = new_key("link", &id);
        let raw: Option<String> = redis::Cmd::get(&key)
            .query_async(&mut conn)
            .await
            .map_err(ErrorInternalServerError)?;
        match raw {
            None => {
                log::info!("failed to find '{key}'");
                Err(ErrorNotFound("link not found for id"))
            }
            Some(raw) => serde_json::from_str(&raw).map_err(internal_server_error),
        }
    }

    pub(crate) async fn del(&self, id: String) -> Result<(), Error> {
        let mut conn = self.conn().await?;
        let key = new_key("link", &id);
        match redis::Cmd::del(&key)
            .query_async(&mut conn)
            .await
            .map_err(ErrorInternalServerError)?
        {
            1 => {
                count_decr(&mut conn).await?;
                Ok(())
            }
            0 => Err(ErrorNotFound("key not found")),
            code => {
                log::error!(id, key, code; "could not delete: unknown redis response code");
                Err(ErrorInternalServerError("could not delete"))
            }
        }
    }

    pub(crate) async fn list(&self) -> Result<Vec<LinkInfo>, Error> {
        let mut conn = self.conn().await?;
        let mut keys = Vec::new();
        {
            let mut keys_iter: redis::AsyncIter<String> = redis::cmd("SCAN")
                .cursor_arg(0)
                .arg("MATCH")
                .arg("link:*")
                .clone() // not sure why I need this but I do
                .iter_async(&mut conn)
                .await
                .map_err(internal_server_error)?;
            while let Some(key) = keys_iter.next_item().await {
                keys.push(key);
            }
        }

        let mut results = Vec::new();
        let mut res: redis::AsyncIter<String> = redis::Cmd::mget(&keys)
            .iter_async(&mut conn)
            .await
            .map_err(internal_server_error)?;
        let mut i = 0;
        while let Some(item) = res.next_item().await {
            let link: Link = serde_json::from_str(&item).map_err(internal_server_error)?;
            results.push(LinkInfo {
                link,
                key: keys[i].clone(),
                expires: None,
            });
            i += 1;
        }
        Ok(results)
    }
}

const COUNT_KEY: &'static str = "meta:count";

async fn count_incr<C>(c: &mut C) -> Result<usize, Error>
where
    C: redis::aio::ConnectionLike,
{
    redis::Cmd::incr(COUNT_KEY, 1)
        .query_async(c)
        .await
        .map_err(ErrorInternalServerError)
}

async fn count_decr<C>(c: &mut C) -> Result<usize, Error>
where
    C: redis::aio::ConnectionLike,
{
    redis::Cmd::decr(COUNT_KEY, 1)
        .query_async(c)
        .await
        .map_err(ErrorInternalServerError)
}

pub fn deserialize_option_ignore_error<'de, T, D>(d: D) -> Result<Option<T>, D::Error>
where
    T: de::Deserialize<'de>,
    D: de::Deserializer<'de>,
{
    Ok(T::deserialize(d).ok())
}

fn internal_server_error<E>(e: E) -> actix_web::Error
where
    E: std::fmt::Debug + std::fmt::Display + 'static,
{
    log::error!("{}", e);
    ErrorInternalServerError("internal server error")
}

#[cfg(test)]
mod tests {
    #[test]
    fn test() {}
}

#[cfg(feature = "functional")]
#[cfg(test)]
mod functional_tests {}
