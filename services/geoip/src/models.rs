use std::io;
use std::io::ErrorKind;
use std::net::IpAddr;
use std::sync::TryLockError;
use std::{collections::BTreeMap, fmt::Display};

use actix_web::http;
use maxminddb::geoip2;
use serde::{Deserialize, Serialize};

use crate::locale::Locales;

#[derive(Serialize, Deserialize, Clone)]
struct Loc {
    latitude: Option<f64>,
    longitude: Option<f64>,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
struct City {
    name: Option<String>,
    id: Option<u32>,
}

#[derive(Serialize, Deserialize, Clone)]
struct Country {
    iso_code: Option<String>,
    name: Option<String>,
}

#[derive(Serialize, Deserialize, Clone)]
struct Subdivision {
    iso_code: Option<String>,
    name: Option<String>,
}

#[derive(Serialize, Deserialize, Clone)]
pub(crate) struct LocationResponse {
    #[serde(skip_serializing)]
    pub(crate) locale: String,

    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) ip: Option<IpAddr>,

    location: Option<Loc>,
    city: Option<City>,
    country: Option<Country>,
    subdivisions: Option<Vec<Subdivision>>,
}

impl Default for LocationResponse {
    fn default() -> Self {
        Self {
            ip: None,
            location: None,
            city: None,
            country: None,
            subdivisions: None,
            locale: "".to_string(),
        }
    }
}

impl Into<actix_web::HttpResponse> for LocationResponse {
    fn into(self) -> actix_web::HttpResponse {
        actix_web::HttpResponse::Ok().json(self)
    }
}

#[derive(Serialize, Deserialize, Clone, Debug)]
pub(crate) struct ErrorResponse {
    pub status: String,
    pub message: String,
}

#[derive(Debug)]
pub(crate) enum LocationError {
    /// Address not found
    NotFound,
    /// Invalid input
    BadInput,
    /// Bad language code
    BadLang,
    /// Internal server error
    Internal,
}

impl std::error::Error for LocationError {}

impl Display for LocationError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::BadInput => write!(f, "invalid input"),
            Self::BadLang => write!(f, "invalid language code"),
            Self::NotFound => write!(f, "ip address location not found"),
            Self::Internal => write!(f, "internal error"),
        }
    }
}

impl Default for LocationError {
    fn default() -> Self {
        Self::Internal
    }
}

impl Into<http::StatusCode> for LocationError {
    fn into(self) -> http::StatusCode {
        use actix_web::http::StatusCode;
        match self {
            Self::BadInput => StatusCode::BAD_REQUEST,
            Self::NotFound => StatusCode::NOT_FOUND,
            Self::BadLang => StatusCode::BAD_REQUEST,
            Self::Internal => StatusCode::INTERNAL_SERVER_ERROR,
        }
    }
}

impl Into<ErrorResponse> for LocationError {
    fn into(self) -> ErrorResponse {
        let message = self.to_string();
        ErrorResponse {
            status: "error".to_string(),
            message,
        }
    }
}

impl Into<actix_web::HttpResponse> for LocationError {
    fn into(self) -> actix_web::HttpResponse {
        let message = self.to_string();
        actix_web::HttpResponseBuilder::new(self.into()).json(ErrorResponse {
            status: "error".to_string(),
            message,
        })
    }
}

impl From<io::Error> for LocationError {
    fn from(e: io::Error) -> Self {
        match e.kind() {
            ErrorKind::NotFound => Self::NotFound,
            ErrorKind::InvalidInput => Self::BadInput,
            _ => Self::Internal,
        }
    }
}

impl From<maxminddb::MaxMindDBError> for LocationError {
    fn from(e: maxminddb::MaxMindDBError) -> Self {
        use maxminddb::MaxMindDBError as MMErr;
        match e {
            MMErr::AddressNotFoundError(_) => Self::NotFound,
            MMErr::InvalidNetworkError(_) => Self::BadInput,
            _ => Self::Internal,
        }
    }
}

impl<T> From<TryLockError<T>> for LocationError {
    fn from(_: TryLockError<T>) -> Self {
        Self::Internal
    }
}

fn get_i18n_name<'a>(
    names: &Option<BTreeMap<&str, &'a str>>,
    locales: &Locales,
    locale: &mut String,
) -> Result<String, LocationError> {
    if let Some(map) = names {
        for l in locales.iter() {
            // Try the full name, then just the raw name.
            let key = l.full_name();
            if let Some(name) = map.get(key.as_str()) {
                if locale.len() == 0 {
                    *locale = key;
                }
                return Ok(name.to_string());
            }
            // Try only the name
            if let Some(name) = map.get(l.name.as_str()) {
                if locale.len() == 0 {
                    *locale = l.name.clone();
                }
                return Ok(name.to_string());
            }
        }
        log::warn!(
            "invalid language codes: {:?} not in {:?}",
            locales,
            map.keys()
        );
        return Err(LocationError::BadLang);
    }
    log::warn!("no locale names");
    return Err(LocationError::Internal);
}

pub(crate) fn city_to_response(
    city: geoip2::City,
    locales: &Locales,
) -> Result<LocationResponse, LocationError> {
    let mut res = LocationResponse::default();

    if let Some(loc) = city.location {
        let geoip2::city::Location {
            latitude,
            longitude,
            ..
        } = loc;
        res.location = Some(Loc {
            latitude,
            longitude,
        });
    }

    if let Some(country) = city.country {
        // If the country language code is not found, assume it's invalid and
        // bubble up the error
        let c = get_i18n_name(&country.names, &locales, &mut res.locale).map(|name| Country {
            iso_code: country.iso_code.map(|s| s.to_string()),
            name: Some(name),
        })?;
        res.country = Some(c);
    }

    if let Some(city) = city.city {
        res.city = Some(City {
            id: city.geoname_id,
            name: get_i18n_name(&city.names, &locales, &mut res.locale).ok(),
        })
    }

    if let Some(subs) = city.subdivisions {
        res.subdivisions = subs
            .iter()
            .map(|s| {
                // If it has no language code name, return None and skip.
                get_i18n_name(&s.names, &locales, &mut res.locale)
                    .ok()
                    .map(|name| Subdivision {
                        iso_code: s.iso_code.map(|c| c.to_string()),
                        name: Some(name),
                    })
            })
            .collect();
    }

    Ok(res)
}
