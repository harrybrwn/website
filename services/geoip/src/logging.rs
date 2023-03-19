use std::future::{ready, Ready};
use std::io;
use std::rc::Rc;
use std::{io::ErrorKind, pin::Pin};

use actix_web::dev::{forward_ready, Service, ServiceRequest, ServiceResponse, Transform};
use actix_web::http::header::USER_AGENT;
use slog::OwnedKV;

pub fn new_logger(service: &'static str) -> io::Result<slog::Logger> {
    let log = slog::Logger::root(
        slog::Fuse(std::sync::Mutex::new(slog_json::Json::default(
            std::io::stdout(),
        ))),
        slog::o!("service" => service),
    );
    let logger = std::boxed::Box::new(JsonLogger::new(log.clone()));
    if let Err(err) = log::set_boxed_logger(logger) {
        return Err(io::Error::new(
            ErrorKind::Other,
            format!("Error: failed to set logger: {err}"),
        ));
    }
    log::set_max_level(log::LevelFilter::Info);
    Ok(log)
}

struct JsonLogger {
    log: slog::Logger,
}

impl JsonLogger {
    fn new(log: slog::Logger) -> Self {
        Self { log }
    }
}

fn to_slog_level(l: log::Level) -> slog::Level {
    match l {
        log::Level::Error => slog::Level::Error,
        log::Level::Warn => slog::Level::Warning,
        log::Level::Info => slog::Level::Info,
        log::Level::Debug => slog::Level::Debug,
        log::Level::Trace => slog::Level::Trace,
    }
}

impl log::Log for JsonLogger {
    #[inline]
    fn flush(&self) {}

    #[inline]
    fn enabled(&self, _metadata: &log::Metadata) -> bool {
        true
    }

    fn log(&self, record: &log::Record) {
        if !self.enabled(record.metadata()) {
            return;
        }
        let values = OwnedKV(slog::kv!(
            "file" => record.file(),
            "line" => record.line(),
            "target" => record.target(),
            "module" => record.module_path(),
        ));

        let static_record = slog::RecordStatic {
            location: &slog::RecordLocation {
                file: record.file_static().unwrap_or(""),
                line: record.line().unwrap_or(0),
                column: 0,
                function: "",
                module: record.module_path_static().unwrap_or(""),
            },
            tag: record.target(),
            level: to_slog_level(record.level()),
        };
        let rec = slog::Record::new(&static_record, record.args(), slog::BorrowedKV(&values));
        self.log.log(&rec);
    }
}

pub(crate) struct AutoLog(Rc<slog::Logger>);

impl AutoLog {
    pub fn new(log: slog::Logger) -> Self {
        Self(Rc::new(log))
    }
}

impl Default for AutoLog {
    fn default() -> Self {
        use std::io::stdout;
        use std::sync::Mutex;
        let drain = Mutex::new(slog_json::Json::default(stdout()));
        let logger = slog::Logger::root(slog::Fuse(drain), slog::o!());
        Self(Rc::new(logger))
    }
}

impl<S, B> Transform<S, ServiceRequest> for AutoLog
where
    S: Service<ServiceRequest, Response = ServiceResponse<B>, Error = actix_web::Error>,
    S::Future: 'static,
    B: 'static,
{
    type Error = actix_web::Error;
    type InitError = ();
    type Response = ServiceResponse<B>;
    type Transform = LoggingMiddleware<S>;
    type Future = Ready<Result<Self::Transform, Self::InitError>>;

    fn new_transform(&self, service: S) -> Self::Future {
        ready(Ok(LoggingMiddleware {
            service,
            logger: self.0.clone(),
        }))
    }
}

pub(crate) struct LoggingMiddleware<S> {
    service: S,
    logger: Rc<slog::Logger>,
}

impl<S, B> Service<ServiceRequest> for LoggingMiddleware<S>
where
    S: Service<ServiceRequest, Response = ServiceResponse<B>, Error = actix_web::Error>,
    S::Future: 'static,
    B: 'static,
{
    type Response = ServiceResponse<B>;
    type Error = actix_web::Error;
    type Future = Pin<Box<dyn core::future::Future<Output = Result<Self::Response, Self::Error>>>>;

    forward_ready!(service);

    fn call(&self, req: ServiceRequest) -> Self::Future {
        let head = req.head();
        let ua = head
            .headers
            .get(USER_AGENT)
            .and_then(|h| h.to_str().ok())
            .unwrap_or("-");
        let kvs = slog::kv!(
            "user_agent" => ua.to_owned(),
            "uri" => head.uri.to_string(),
            "method" => head.method.as_str().to_owned(),
        );
        let logger = self.logger.clone();

        let fut = self.service.call(req);
        Box::pin(async move {
            let res = fut.await?;
            let status = res.status().as_u16();
            let kvs = slog::kv!(kvs, "status" => status);

            let static_record = slog::record_static!(slog::Level::Info, "request");
            let args = format_args!("request");
            let record = slog::Record::new(&static_record, &args, slog::BorrowedKV(&kvs));
            logger.log(&record);
            Ok(res)
        })
    }
}
