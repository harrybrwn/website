use std::{
    boxed::Box,
    future::{ready, Future, Ready},
    net::SocketAddr,
    pin::Pin,
};

use actix_web::dev::{forward_ready, Service, ServiceRequest, ServiceResponse, Transform};

pub struct RequestLogger;

impl<S, B> Transform<S, ServiceRequest> for RequestLogger
where
    S: Service<ServiceRequest, Response = ServiceResponse<B>, Error = actix_web::Error>,
    S::Future: 'static,
    B: 'static,
{
    type Error = S::Error;
    type InitError = ();
    type Response = S::Response;
    type Transform = LoggingMiddleware<S>;
    type Future = Ready<Result<Self::Transform, Self::InitError>>;
    // Runs once per thread.
    fn new_transform(&self, service: S) -> Self::Future {
        ready(Ok(LoggingMiddleware { service }))
    }
}

pub struct LoggingMiddleware<S> {
    service: S,
}

impl<S, B> Service<ServiceRequest> for LoggingMiddleware<S>
where
    S: Service<ServiceRequest, Response = ServiceResponse<B>, Error = actix_web::Error>,
    S::Future: 'static,
    B: 'static,
{
    type Response = S::Response;
    type Error = S::Error;
    type Future = Pin<Box<dyn Future<Output = Result<Self::Response, Self::Error>>>>;

    forward_ready!(service);

    fn call(&self, req: ServiceRequest) -> Self::Future {
        let head = req.head();
        let mut info = RequestInfo::from(head);
        let fut = self.service.call(req);
        Box::pin(async move {
            if log::Level::Info > log::max_level() {
                return fut.await;
            }
            let mut record = log::RecordBuilder::new();
            let res = fut.await?;
            info.read_response(&res);
            log::logger().log(
                &record
                    .module_path_static(Some(module_path!()))
                    .file(Some(file!()))
                    .line(Some(line!()))
                    .key_values(&info)
                    .level(log::Level::Info)
                    .target("web")
                    .args(format_args!("request finished"))
                    .build(),
            );
            Ok(res)
        })
    }
}

struct RequestInfo {
    method: String,
    uri: String,
    user_agent: Option<String>,
    status: u16,
    ip: Option<SocketAddr>,
}

impl RequestInfo {
    fn read_response<B>(&mut self, res: &ServiceResponse<B>) {
        self.status = res.status().as_u16();
    }
}

impl From<&actix_http::RequestHead> for RequestInfo {
    fn from(h: &actix_http::RequestHead) -> Self {
        Self {
            method: h.method.to_string(),
            uri: h.uri.to_string(),
            user_agent: if let Some(ua) = h.headers().get("User-Agent") {
                ua.to_str().ok().map(String::from)
            } else {
                None
            },
            status: 0,
            ip: None,
        }
    }
}

impl From<actix_http::RequestHead> for RequestInfo {
    fn from(value: actix_http::RequestHead) -> Self {
        Self::from(&value)
    }
}

impl log::kv::Source for RequestInfo {
    fn count(&self) -> usize {
        let mut c = 2;
        if self.status != 0 {
            c += 1;
        }
        if self.user_agent.is_some() {
            c += 1;
        }
        if self.ip.is_some() {
            c += 1;
        }
        c
    }

    fn get(&self, key: log::kv::Key) -> Option<log::kv::Value<'_>> {
        use log::kv::Value;
        match key.as_str() {
            "method" => Some(Value::from(&self.method)),
            "uri" => Some(Value::from(&self.uri)),
            "status" if self.status != 0 => Some(Value::from(self.status)),
            "user_agent" | "user-agent" => {
                if let Some(ref ua) = self.user_agent {
                    Some(Value::from(ua))
                } else {
                    None
                }
            }
            _ => None,
        }
    }

    fn visit<'kvs>(&'kvs self, v: &mut dyn log::kv::Visitor<'kvs>) -> Result<(), log::kv::Error> {
        use log::kv::{Key, Value};
        v.visit_pair(Key::from_str("uri"), Value::from(&self.uri))?;
        v.visit_pair(Key::from_str("method"), Value::from(&self.method))?;
        if self.status != 0 {
            v.visit_pair(Key::from_str("status"), Value::from(self.status))?;
        }
        if let Some(ref ua) = self.user_agent {
            v.visit_pair(Key::from_str("user-agent"), Value::from(ua))?;
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    #[test]
    fn it_works() {}
}
