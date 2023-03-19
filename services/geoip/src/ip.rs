use std::io::{Error, ErrorKind};
use std::net::IpAddr;
use std::pin::Pin;
use std::str::FromStr;

use actix_web::{FromRequest, HttpRequest};

#[derive(Debug, PartialEq, Eq, PartialOrd, Ord)]
pub(crate) struct ClientIP(IpAddr);

pub(crate) fn is_local(ip: &IpAddr) -> bool {
    match ip {
        IpAddr::V4(ip) => match ip.octets() {
            // RFC 1918
            [10, ..] => true,
            [192, 168, ..] => true,
            [172, b, ..] => b & 240 == 0x10, // true for 172.16.x.x - 172.31.x.x
            // loopback
            [127, ..] => true,
            // link local
            [169, 254, ..] => true,
            // broadcast
            [255, 255, 255, 255] => true,
            // unspecified
            [0, 0, 0, 0] => true,
            _ => false,
        },
        IpAddr::V6(ip) => {
            let segments = ip.segments();
            // check multicast
            if segments[0] & 0xfe00 == 0xfc00 {
                true
            } else if segments[0] & 0xff00 == 0xff00 {
                segments[0] & 0x000f != 14
            } else {
                match segments {
                    // loopback
                    [0, 0, 0, 0, 0, 0, 0, 1] => true,
                    // unspecified
                    [0, 0, 0, 0, 0, 0, 0, 0] => true,
                    _ => match segments[0] & 0xffc0 {
                        0xfe80 => true,
                        0xfec0 => true,
                        _ => {
                            if segments[0] & 0xfe00 == 0xfc00 {
                                true
                            } else {
                                (segments[0] == 0x2001) && (segments[1] == 0xdb8)
                            }
                        }
                    },
                }
            }
        }
    }
}

#[inline]
pub(crate) fn is_public_ip(ip: &IpAddr) -> bool {
    !is_local(ip)
}

impl ClientIP {
    #[inline]
    pub(crate) fn ip(&self) -> IpAddr {
        self.0
    }
}

impl FromRequest for ClientIP {
    type Error = Error;
    type Future = Pin<Box<dyn core::future::Future<Output = Result<Self, Self::Error>>>>;

    fn from_request(req: &HttpRequest, _: &mut actix_web::dev::Payload) -> Self::Future {
        let peer_ip = req.peer_addr().map(|s| s.ip());
        if let Some(ip) = peer_ip.filter(is_public_ip) {
            return Box::pin(async move { Ok(Self(ip)) });
        }
        let forwarded = req
            .headers()
            .get_all(actix_web::http::header::X_FORWARDED_FOR)
            .filter_map(|h| h.to_str().ok())
            .filter_map(|h| IpAddr::from_str(h).ok())
            .filter(is_public_ip)
            .next();
        Box::pin(async move {
            match forwarded {
                Some(ip) => Ok(Self(ip)),
                // Return the peer IP address if nothing is found
                None => match peer_ip {
                    Some(ip) => Ok(Self(ip)),
                    None => Err(Error::new(ErrorKind::InvalidInput, "no ip address")),
                },
            }
        })
    }
}
