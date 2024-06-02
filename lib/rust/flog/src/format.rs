use std::io::{self, Write};

pub trait Fmt {
    fn write<W: Write>(&self, w: &mut W, record: &log::Record) -> io::Result<()>;
}

/// ButFmt is a joke format. All it does is write "butt".
pub struct ButtFmt;

impl Fmt for ButtFmt {
    fn write<W: io::Write>(&self, w: &mut W, _record: &log::Record) -> io::Result<()> {
        w.write("butt".as_bytes())?;
        Ok(())
    }
}

/// JsonFmt will format the log fields as json.
pub struct JsonFmt;

impl Fmt for JsonFmt {
    fn write<W: io::Write>(&self, w: &mut W, record: &log::Record) -> io::Result<()> {
        use chrono::Utc;
        use serde_json::Value;
        let now = Utc::now();
        let mut m = serde_json::Map::new();
        m.insert(
            "time".to_string(),
            Value::String(
                now.to_rfc3339_opts(chrono::SecondsFormat::Secs, true)
                    .to_string(),
            ),
        );
        m.insert(
            "level".to_string(),
            Value::String(record.level().to_string()),
        );
        m.insert(
            "target".to_string(),
            Value::String(record.target().to_string()),
        );
        m.insert(
            "msg".to_string(),
            Value::String(std::fmt::format(*record.args())),
        );
        let mut visitor = JsonFmtVisitor { m: &mut m };
        record
            .key_values()
            .visit(&mut visitor)
            .map_err(|e| io::Error::new(io::ErrorKind::Other, e))?;
        serde_json::to_writer(w, &m)?;
        Ok(())
    }
}

struct JsonFmtVisitor<'a> {
    m: &'a mut serde_json::Map<String, serde_json::Value>,
}

impl<'a, 'kvs> log::kv::VisitSource<'kvs> for JsonFmtVisitor<'a> {
    fn visit_pair(
        &mut self,
        key: log::kv::Key<'kvs>,
        value: log::kv::Value<'kvs>,
    ) -> Result<(), log::kv::Error> {
        let mut val = JsonValue(serde_json::Value::Null);
        value.visit(&mut val)?;
        self.m.insert(key.to_string(), val.0);
        Ok(())
    }
}

struct JsonValue(serde_json::Value);

impl<'kvs> log::kv::VisitValue<'kvs> for JsonValue {
    fn visit_str(&mut self, value: &str) -> Result<(), log::kv::Error> {
        self.0 = serde_json::Value::String(value.to_string());
        Ok(())
    }
    fn visit_null(&mut self) -> Result<(), log::kv::Error> {
        self.0 = serde_json::Value::Null;
        Ok(())
    }
    fn visit_bool(&mut self, value: bool) -> Result<(), log::kv::Error> {
        self.0 = serde_json::Value::Bool(value);
        Ok(())
    }
    fn visit_i64(&mut self, value: i64) -> Result<(), log::kv::Error> {
        self.0 = serde_json::Value::Number(serde_json::Number::from(value));
        Ok(())
    }
    fn visit_u64(&mut self, value: u64) -> Result<(), log::kv::Error> {
        self.0 = serde_json::Value::Number(serde_json::Number::from(value));
        Ok(())
    }
    fn visit_f64(&mut self, value: f64) -> Result<(), log::kv::Error> {
        if let Some(n) = serde_json::Number::from_f64(value) {
            self.0 = serde_json::Value::Number(n);
            Ok(())
        } else {
            Err(log::kv::Error::msg("invalid f64"))
        }
    }
    fn visit_any(&mut self, value: log::kv::Value) -> Result<(), log::kv::Error> {
        if let Some(v) = value.to_bool() {
            self.visit_bool(v)
        } else if let Some(v) = value.to_i64() {
            self.visit_i64(v)
        } else if let Some(v) = value.to_u64() {
            self.visit_u64(v)
        } else if let Some(v) = value.to_f64() {
            self.visit_f64(v)
        } else {
            self.0 = serde_json::Value::String(value.to_string());
            Ok(())
        }
    }
}

/// LogFmt will write out the log key/value pairs in logfmt format.
pub struct LogFmt;

impl Fmt for LogFmt {
    fn write<W: io::Write>(&self, w: &mut W, record: &log::Record) -> io::Result<()> {
        write_logfmt(w, record)
    }
}

fn write_logfmt<W>(w: &mut W, record: &log::Record) -> io::Result<()>
where
    W: io::Write,
{
    use chrono::Utc;
    let now = Utc::now();
    let level = record.level();
    let args = record.args();
    // time
    w.write("time=\"".as_bytes())?;
    w.write(
        now.to_rfc3339_opts(chrono::SecondsFormat::Secs, true)
            .as_bytes(),
    )?;
    w.write(&['"' as u8, ' ' as u8])?;
    // level
    w.write("level=".as_bytes())?;
    w.write(level.as_str().as_bytes())?;
    w.write(&[' ' as u8])?;
    // target
    w.write("target=".as_bytes())?;
    w.write(record.target().as_bytes())?;
    w.write(&[' ' as u8])?;
    // message
    w.write("msg=\"".as_bytes())?;
    w.write_fmt(*args)?;
    w.write(&['"' as u8])?;
    // other keys
    let kvs = record.key_values();
    let mut visitor = LogFmtVisitor { w };
    kvs.visit(&mut visitor)
        .map_err(|e| io::Error::new(io::ErrorKind::Other, e))?;
    Ok(())
}

struct LogFmtVisitor<'a, W: io::Write> {
    w: &'a mut W,
}

impl<'a, 'kvs, W: io::Write> log::kv::VisitSource<'kvs> for LogFmtVisitor<'a, W> {
    fn visit_pair(
        &mut self,
        key: log::kv::Key<'kvs>,
        value: log::kv::Value<'kvs>,
    ) -> Result<(), log::kv::Error> {
        self.w.write(&[' ' as u8])?;
        self.w.write(key.as_str().as_bytes())?;
        self.w.write(&['=' as u8])?;
        value.visit(self)?;
        Ok(())
    }
}

impl<'a, 'v, W: io::Write> log::kv::VisitValue<'v> for LogFmtVisitor<'a, W> {
    fn visit_any(&mut self, value: log::kv::Value) -> Result<(), log::kv::Error> {
        self.w.write(&['"' as u8])?;
        self.w.write(value.to_string().as_bytes())?;
        self.w.write(&['"' as u8])?;
        Ok(())
    }
    fn visit_str(&mut self, value: &str) -> Result<(), log::kv::Error> {
        if value.contains(' ') {
            self.w.write(&['"' as u8])?;
            self.w.write(value.as_bytes())?;
            self.w.write(&['"' as u8])?;
        } else {
            self.w.write(value.as_bytes())?;
        }
        Ok(())
    }
    fn visit_u64(&mut self, value: u64) -> Result<(), log::kv::Error> {
        self.w.write(value.to_string().as_bytes())?;
        Ok(())
    }
    fn visit_i64(&mut self, value: i64) -> Result<(), log::kv::Error> {
        self.w.write(value.to_string().as_bytes())?;
        Ok(())
    }
    fn visit_f64(&mut self, value: f64) -> Result<(), log::kv::Error> {
        self.w.write(value.to_string().as_bytes())?;
        Ok(())
    }
    fn visit_null(&mut self) -> Result<(), log::kv::Error> {
        self.w.write("nil".as_bytes())?;
        Ok(())
    }
    fn visit_bool(&mut self, value: bool) -> Result<(), log::kv::Error> {
        self.w.write(value.to_string().as_bytes())?;
        Ok(())
    }
}

#[derive(Default)]
struct KVCollector<'kvs> {
    kvs: Vec<(log::kv::Key<'kvs>, log::kv::Value<'kvs>)>,
}

impl<'kvs> log::kv::VisitSource<'kvs> for KVCollector<'kvs> {
    fn visit_pair(
        &mut self,
        key: log::kv::Key<'kvs>,
        value: log::kv::Value<'kvs>,
    ) -> Result<(), log::kv::Error> {
        self.kvs.push((key, value));
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use crate::{Fmt, LogFmt};

    #[test]
    fn logfmt() {
        let f = LogFmt;
        let mut b = Vec::new();
        // let mut b = std::io::stdout();
        let r = log::RecordBuilder::new().args(format_args!("test")).build();
        f.write(&mut b, &r).unwrap();
    }
}
