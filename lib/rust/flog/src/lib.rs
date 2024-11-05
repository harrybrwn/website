use std::env;
use std::io;
use std::io::Write;
use std::str::FromStr;
use std::sync::{Arc, Mutex};

mod format;
pub use crate::format::Fmt;
use crate::format::{ButtFmt, JsonFmt, LogFmt};

pub fn init() {
    Config::new().init().unwrap()
}

pub struct Config {
    pub level: Level,
    pub format: Format,
}

pub type Level = log::Level;

impl Config {
    pub fn new() -> Self {
        Self {
            level: Level::Info,
            format: Format::LogFmt,
        }
    }

    pub fn load_env(&mut self) -> &mut Self {
        if let Ok(v) = env::var("LOG_FORMAT") {
            self.format = Format::from(v.as_str());
        }
        if let Ok(v) = env::var("RUST_LOG") {
            if let Ok(l) = Level::from_str(&v) {
                self.level = l;
            }
        }
        self
    }

    pub fn level(&mut self, lvl: Level) -> &mut Self {
        self.level = lvl;
        self
    }

    pub fn format(&mut self, format: Format) -> &mut Self {
        self.format = format;
        self
    }

    pub fn init(&self) -> Result<(), log::SetLoggerError> {
        log::set_max_level(self.level.to_level_filter());
        match self.format {
            Format::Butt => {
                log::set_boxed_logger(Box::new(Logger::new(io::stdout(), ButtFmt, self.level)))
            }
            Format::LogFmt => {
                log::set_boxed_logger(Box::new(Logger::new(io::stdout(), LogFmt, self.level)))
            }
            Format::Json => {
                log::set_boxed_logger(Box::new(Logger::new(io::stdout(), JsonFmt, self.level)))
            }
        }?;
        Ok(())
    }
}

#[derive(Default, Clone, Copy, PartialEq, Eq, Debug, Hash)]
pub enum Format {
    #[default]
    LogFmt,
    Json,
    Butt,
}

impl std::fmt::Display for Format {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Json => f.write_str("json"),
            Self::LogFmt => f.write_str("logfmt"),
            Self::Butt => f.write_str("butt"),
        }
    }
}

impl From<&str> for Format {
    fn from(value: &str) -> Self {
        match value.to_lowercase().as_str() {
            "json" => Self::Json,
            "logfmt" | "text" => Self::LogFmt,
            "butt" => Self::Butt,
            _ => Self::LogFmt,
        }
    }
}

pub struct Logger<W, F>
where
    W: Write,
    F: Fmt,
{
    level: log::Level,
    w: Arc<Mutex<io::BufWriter<W>>>,
    formatter: F,
}

impl<W, F> Logger<W, F>
where
    W: io::Write,
    F: Fmt,
{
    pub fn new(w: W, f: F, level: Level) -> Self {
        Logger {
            level,
            w: Arc::new(Mutex::new(io::BufWriter::new(w))),
            formatter: f,
        }
    }
}

impl<W, F> log::Log for Logger<W, F>
where
    W: Write + Sync + Send,
    F: Fmt + Sync + Send,
{
    fn log(&self, record: &log::Record) {
        if self.enabled(record.metadata()) {
            let mut w = self.w.lock().unwrap();
            self.formatter.write(w.by_ref(), record).unwrap();
            w.write(&['\n' as u8]).unwrap();
            w.flush().unwrap();
        }
    }

    fn flush(&self) {
        self.w.lock().unwrap().flush().unwrap()
    }

    fn enabled(&self, metadata: &log::Metadata) -> bool {
        metadata.level() <= self.level
    }
}

#[cfg(test)]
mod tests {
    #[test]
    fn it_works() {}
}
