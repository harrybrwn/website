use redis::aio;
use serde::de;
use url::Url;

use actix_web::error::{
    Error, ErrorBadRequest, ErrorConflict, ErrorInternalServerError, ErrorNotFound,
};
use serde_derive::{Deserialize, Serialize};

const ID_SIZE: usize = 16;

fn gen_id<T: rand::Rng>(rng: T) -> Vec<u8> {
    rng.sample_iter(&rand::distributions::Alphanumeric)
        .take(ID_SIZE)
        .collect()
}

#[derive(Deserialize, Serialize, Debug)]
pub(crate) struct CreateRequest {
    /// URL of the link being created.
    pub(crate) url: String,
    /// Number of seconds the link will live.
    #[serde(default, deserialize_with = "deserialize_option_ignore_error")]
    pub(crate) expires: Option<u64>,
    /// Number of times a link can be accessed before self destructing.
    pub(crate) access_limit: Option<u32>,
}

#[derive(Deserialize, Serialize, Debug)]
pub(crate) struct Link {
    /// URL of the link being created.
    pub(crate) url: String,
    /// Number of accesses left before a link is deleted.
    pub(crate) accesses: Option<u32>,
}

#[derive(Debug)]
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
    serde_json::to_string(item).map_err(ErrorInternalServerError)
}

fn new_key(prefix: &str, id: &str) -> String {
    let mut key = String::from(prefix);
    key.push(':');
    key.push_str(id);
    key
}

#[derive(Clone)]
pub(crate) struct Store {
    host: String,
    rd: redis::Client,
}

impl Store {
    pub(crate) fn new(host: String, rd: redis::Client) -> Self {
        Self { host, rd }
    }

    #[inline]
    async fn conn(&self) -> Result<aio::ConnectionManager, actix_web::Error> {
        self.rd
            .get_tokio_connection_manager()
            .await
            .map_err(ErrorInternalServerError)
    }

    pub(crate) async fn create(&self, req: &CreateRequest) -> Result<String, Error> {
        let u = Url::parse(&req.url).map_err(ErrorBadRequest)?;
        if let Some(url::Host::Domain(d)) = u.host() {
            if d == self.host {
                return Err(ErrorConflict("cannot self link"));
            }
        }
        let mut conn = self.conn().await?;
        let mut rng = rand::thread_rng();
        let id = gen_id(&mut rng);
        let link = Link {
            url: req.url.clone(),
            accesses: req.access_limit,
        };
        let str_id = std::str::from_utf8(&id).map_err(ErrorInternalServerError)?;
        let key = new_key("link", str_id);
        let cmd: redis::Cmd;
        if let Some(exp) = req.expires {
            cmd = redis::Cmd::set_ex(key, to_json(&link)?, exp as usize);
        } else {
            cmd = redis::Cmd::set(key, to_json(&link)?);
        }
        cmd.query_async::<_, Option<String>>(&mut conn)
            .await
            .map_err(ErrorInternalServerError)?;
        String::from_utf8(id).map_err(ErrorInternalServerError)
    }

    pub(crate) async fn get(&self, id: String) -> Result<Link, Error> {
        let mut conn = self.conn().await?;
        let key = new_key("link", &id);
        let raw = redis::Cmd::get(&key)
            .query_async::<_, Option<String>>(&mut conn)
            .await
            .map_err(ErrorInternalServerError)?;
        match raw {
            None => {
                log::info!("failed to find {key}");
                Err(ErrorNotFound("link not found for id"))
            }
            Some(raw) => serde_json::from_str(&raw).map_err(ErrorInternalServerError),
        }
    }

    pub(crate) async fn del(&self, id: String) -> Result<(), Error> {
        let mut conn = self.conn().await?;
        let key = new_key("link", &id);
        redis::Cmd::del(key)
            .query_async(&mut conn)
            .await
            .map_err(ErrorInternalServerError)?;
        Ok(())
    }

    pub(crate) async fn list(&self) -> Result<Vec<LinkInfo>, Error> {
        let mut keys = Vec::new();
        {
            let mut conn = self.conn().await?;
            let mut keys_iter: redis::AsyncIter<String> = redis::cmd("SCAN")
                .cursor_arg(0)
                .arg("MATCH")
                .arg("link:*")
                .clone() // not sure why I need this but I do
                .iter_async(&mut conn)
                .await
                .map_err(ErrorInternalServerError)?;
            while let Some(key) = keys_iter.next_item().await {
                keys.push(key);
            }
        }

        let mut results = Vec::new();
        {
            let mut conn = self.conn().await?;
            let mut res: redis::AsyncIter<String> = redis::Cmd::mget(&keys)
                .iter_async(&mut conn)
                .await
                .map_err(ErrorInternalServerError)?;
            let mut i = 0;
            while let Some(item) = res.next_item().await {
                let link: Link = serde_json::from_str(&item).map_err(ErrorInternalServerError)?;
                results.push(LinkInfo {
                    link,
                    key: keys[i].clone(),
                    expires: None,
                });
                i += 1;
            }
        }

        Ok(results)
    }
}

pub fn deserialize_option_ignore_error<'de, T, D>(d: D) -> Result<Option<T>, D::Error>
where
    T: de::Deserialize<'de>,
    D: de::Deserializer<'de>,
{
    Ok(T::deserialize(d).ok())
}
