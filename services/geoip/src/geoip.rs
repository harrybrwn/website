use actix_web::http;
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;
use std::net::IpAddr;
use std::sync::TryLockError;

use maxminddb::{self, geoip2, MaxMindDBError};

use crate::locale::Locales;

#[derive(Serialize, Deserialize, Clone, Debug)]
pub(crate) struct ErrorResponse {
    pub status: String,
    pub message: String,
}

trait LocationDatabase {
    fn lookup(&self, ip: IpAddr, locales: &Locales) -> Result<GeoLoc, GeoError>;
    fn languages(&self, ip: IpAddr) -> Result<Vec<&str>, GeoError>;
}

pub(crate) struct GeoDB {
    pub city: maxminddb::Reader<Vec<u8>>,
    pub asn: maxminddb::Reader<Vec<u8>>,
}

impl GeoDB {
    pub fn new(city: maxminddb::Reader<Vec<u8>>, asn: maxminddb::Reader<Vec<u8>>) -> Self {
        let mut db = Self { city, asn };
        if db.city.metadata.database_type == "GeoLite2-ASN" {
            (db.city, db.asn) = (db.asn, db.city);
        }
        db
    }
}

impl GeoDB {
    pub fn lookup(&self, ip: IpAddr, locales: &Locales) -> Result<GeoLoc, GeoError> {
        let city = match self.city.lookup::<geoip2::City>(ip) {
            Ok(res) => res,
            Err(e) => return Err(GeoError::from(e)),
        };
        let mut result = GeoLoc {
            ip: None,
            as_org: match self.asn.lookup::<geoip2::Asn>(ip) {
                Ok(res) => res.autonomous_system_organization,
                Err(_) => None,
            },
            location: city.location.map(|l| l.into()),
            locale: String::new(),
            country: None,
            city: None,
            subdivisions: None,
        };
        if let Some(country) = city.country {
            let (name, lang) = get_i18n_name(&country.names, &result.locale, locales)?;
            result.country = Some(Country {
                iso_code: country.iso_code.map(|s| s.to_string()),
                name: Some(name),
            });
            result.locale = lang;
        }
        if let Some(city) = city.city {
            let (name, locale_res) = get_i18n_name(&city.names, &result.locale, locales)?;
            result.city = Some(City {
                id: city.geoname_id,
                name: Some(name),
            });
            if result.locale.is_empty() {
                result.locale = locale_res;
            }
        }
        if let Some(subs) = city.subdivisions {
            result.subdivisions = subs
                .iter()
                .map(|s| {
                    // If it has no language code name, return None and skip.
                    get_i18n_name(&s.names, &result.locale, locales)
                        .ok()
                        .map(|(name, _)| Subdivision {
                            iso_code: s.iso_code.map(|c| c.to_string()),
                            name: Some(name),
                        })
                })
                .collect();
        }
        Ok(result)
    }

    pub fn languages(&self, ip: IpAddr) -> Result<Vec<&str>, GeoError> {
        match self.city.lookup::<geoip2::City>(ip) {
            Ok(c) => {
                #[allow(clippy::map_clone)]
                match c
                    .country
                    .and_then(|c| c.names)
                    .map(|n| n.keys().map(|k| *k).collect::<Vec<_>>())
                {
                    Some(langs) => Ok(langs),
                    None => Err(GeoError::NotFound),
                }
            }
            Err(e) => Err(GeoError::from(e)),
        }
    }
}

fn get_i18n_name<'a>(
    names: &Option<BTreeMap<&str, &'a str>>,
    cache: &'a str,
    locales: &Locales,
) -> Result<(String, String), GeoError> {
    let map = match names {
        Some(m) => m,
        None => {
            log::warn!("no locale names");
            return Err(GeoError::Internal);
        }
    };
    if cache.is_empty() {
        if let Some(name) = map.get(cache) {
            return Ok((name.to_string(), cache.to_owned()));
        }
    }
    for l in locales.iter() {
        // Try the full name, then just the raw name.
        let key = l.full_name();
        if let Some(name) = map.get(&key.as_str()) {
            return Ok((name.to_string(), key));
        }
        // Try only the name
        if let Some(name) = map.get(l.name.as_str()) {
            return Ok((name.to_string(), l.name.clone()));
        }
    }
    log::warn!(
        "invalid language codes: {:?} not in {:?}",
        locales,
        map.keys()
    );
    Err(GeoError::BadLang)
}

#[derive(Serialize, Deserialize, Clone, Default, Debug)]
pub(crate) struct Loc {
    latitude: f64,
    longitude: f64,
}

impl<'a> From<geoip2::city::Location<'a>> for Loc {
    fn from(l: geoip2::city::Location) -> Self {
        let latitude = l.latitude.unwrap_or_default();
        let longitude = l.longitude.unwrap_or_default();
        Loc {
            latitude,
            longitude,
        }
    }
}

#[derive(Serialize, Deserialize, Clone, Debug)]
struct City {
    name: Option<String>,
    id: Option<u32>,
}

#[derive(Serialize, Deserialize, Clone, Debug)]
struct Country {
    iso_code: Option<String>,
    name: Option<String>,
}

#[derive(Serialize, Deserialize, Clone, Debug)]
struct Subdivision {
    iso_code: Option<String>,
    name: Option<String>,
}

#[derive(Serialize, Deserialize, Debug, Default)]
pub struct GeoLoc<'a> {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) ip: Option<IpAddr>,
    #[serde(skip_serializing)]
    pub(crate) locale: String,
    as_org: Option<&'a str>,
    location: Option<Loc>,
    city: Option<City>,
    country: Option<Country>,
    subdivisions: Option<Vec<Subdivision>>,
}

#[derive(Debug)]
pub(crate) enum GeoError {
    /// IP address location not found.
    NotFound,
    /// Bad language code.
    BadLang,
    /// Internal server error.
    Internal,
}

impl std::error::Error for GeoError {}

impl std::fmt::Display for GeoError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::NotFound => write!(f, "ip address location not found"),
            Self::BadLang => write!(f, "invalid language code"),
            Self::Internal => write!(f, "internal database error"),
        }
    }
}

impl From<GeoError> for ErrorResponse {
    fn from(e: GeoError) -> Self {
        let message = e.to_string();
        ErrorResponse {
            status: "error".to_string(),
            message,
        }
    }
}

impl From<GeoError> for http::StatusCode {
    fn from(e: GeoError) -> Self {
        use actix_web::http::StatusCode;
        use GeoError::*;
        match e {
            NotFound => StatusCode::NOT_FOUND,
            BadLang => StatusCode::BAD_REQUEST,
            Internal => StatusCode::INTERNAL_SERVER_ERROR,
        }
    }
}

impl From<GeoError> for actix_web::HttpResponse {
    fn from(e: GeoError) -> Self {
        let message = e.to_string();
        actix_web::HttpResponseBuilder::new(e.into()).json(ErrorResponse {
            status: "error".to_string(),
            message,
        })
    }
}

impl From<MaxMindDBError> for GeoError {
    fn from(err: maxminddb::MaxMindDBError) -> Self {
        use maxminddb::MaxMindDBError as MMErr;
        match err {
            MMErr::AddressNotFoundError(_) => Self::NotFound,
            MMErr::InvalidNetworkError(_) => Self::NotFound,
            _ => Self::Internal,
        }
    }
}

impl<T> From<TryLockError<T>> for GeoError {
    fn from(_: TryLockError<T>) -> Self {
        // TODO maybe log the error message since the data will be lost
        Self::Internal
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::locale::Locales;
    use std::net::IpAddr;
    use std::str::FromStr;

    #[test]
    fn test_get_i18n_name() {
        let names = BTreeMap::from([
            ("en", "Finland"),
            ("de", "Finnland"),
            ("es", "Finlandia"),
            ("fr", "Finlande"),
            ("ja", "フィンランド共和国"),
            ("pt-BR", "Finlândia"),
            ("ru", "Финляндия"),
            ("zh-CN", "芬兰"),
        ]);
        // let locales = Locales::from(["en;q=0.1", "zh-CN; q=0.8", "ja; q=0.2"]);
        let locales: &[&str] = &["en;q=0.1", "zh-CN; q=0.8", "ja; q=0.2"];
        // let mut l = String::new();
        let result = get_i18n_name(&Some(names.clone()), "", &locales.into()).unwrap();
        assert_eq!("芬兰", result.0);
        assert_eq!("zh-CN", result.1);
        let result = get_i18n_name(&Some(names.clone()), "", &Locales::from(vec!["en"])).unwrap();
        assert_eq!("Finland", result.0);
        assert_eq!("en", result.1);
        let result = get_i18n_name(&Some(names.clone()), "", &["en"].into()).unwrap();
        assert_eq!("Finland", result.0);
        let result = get_i18n_name(&Some(names.clone()), "ru", &["en"].into()).unwrap();
        assert_eq!("Финляндия", result.0);
        assert_eq!("ru", result.1);
    }

    #[test]
    fn test_open_db() {
        use maxminddb::Reader as MMDB;
        let ips = vec![
            "198.12.65.246",
            "134.195.121.38",
            "135.181.162.99",
            "99.83.231.61",
            "95.216.235.9",
            "20.244.22.67",
            "45.32.111.71",
        ];
        let db = GeoDB::new(
            MMDB::open_readfile("testdata/latest/GeoLite2-City.mmdb").unwrap(),
            MMDB::open_readfile("testdata/latest/GeoLite2-ASN.mmdb").unwrap(),
        );
        for ip in ips {
            let ip = IpAddr::from_str(ip).unwrap();
            let r = db.lookup(ip, &Locales::from_iter(["en"])).unwrap();
            let loc = r.location.unwrap();
            let country = r.country.unwrap();
            println!(
                "lookup {:>16} => {:?} {:<28} ({:>8.3}, {:>8.3}) {}:{:<16} {:<8}",
                ip,
                r.locale,
                r.as_org.unwrap(),
                loc.latitude,
                loc.longitude,
                country.iso_code.unwrap(),
                country.name.unwrap(),
                r.city.map_or(String::new(), |c| c.name.unwrap()),
            );
        }
    }
}
