use std::net::IpAddr;
use std::str::FromStr;

use actix_web::http::header::{self, map::HeaderMap, X_FORWARDED_FOR};

use crate::ip::is_public_ip;

#[test]
fn test_parse_ip() {
    let ipv4 = IpAddr::from_str("10.4.9.2").unwrap();
    let ipv6 = IpAddr::from_str("fe80::f26e:97f7:b38b:c890").unwrap();

    assert!(ipv4.is_ipv4());
    assert!(ipv6.is_ipv6());
}

#[test]
fn test_client_ip() {
    let mut headers = HeaderMap::new();
    let values = vec!["127.0.0.1", "invalid-ip", "10.2.8.0", "11.3.100.201"];
    for v in values {
        headers.append(X_FORWARDED_FOR, header::HeaderValue::from_static(v));
    }

    let result: Option<IpAddr> = headers
        .get_all(X_FORWARDED_FOR)
        .filter_map(|h| h.to_str().ok().and_then(|v| IpAddr::from_str(v).ok()))
        // .filter_map(|h| h.to_str().ok())
        // .filter_map(|h| IpAddr::from_str(h).ok())
        .filter(is_public_ip)
        .next();
    assert!(!result.is_none());
    assert_eq!(result.unwrap(), IpAddr::from_str("11.3.100.201").unwrap());
}
