#[derive(Clone, Debug)]
pub struct Mime {
    pub typ: Type,
    pub sub: SubType,
}

impl From<&str> for Mime {
    fn from(v: &str) -> Self {
        match v.split_once('/') {
            Some((t, s)) => Self {
                typ: Type::from(t),
                sub: SubType::from(s),
            },
            None => Self::default(),
        }
    }
}

impl Default for Mime {
    fn default() -> Self {
        Self {
            typ: Type::None,
            sub: SubType::None,
        }
    }
}

impl Mime {
    pub fn any() -> Self {
        Self {
            typ: Type::Any,
            sub: SubType::Any,
        }
    }

    pub const fn valid(&self) -> bool {
        match (self.typ, self.sub) {
            (Type::None, SubType::None) | (Type::None, _) | (_, SubType::None) => false,
            (typ, sub) => sub.valid_type(typ),
        }
    }

    #[inline]
    pub fn matches_str(&self, text: &str) -> bool {
        let other = Self::from(text);
        self.matches(&other)
    }

    #[inline]
    pub fn matches(&self, other: &Self) -> bool {
        self.matches_pair(other.typ, other.sub)
    }

    fn matches_pair(&self, typ: Type, sub: SubType) -> bool {
        match (typ, sub) {
            (Type::None, SubType::None) | (Type::None, _) | (_, SubType::None) => false,
            (Type::Any, SubType::Any) => true,
            (typ, SubType::Any) => typ == self.typ || self.typ == Type::Any,
            (Type::Any, sub) => sub == self.sub || self.sub == SubType::Any,
            (typ, sub) => {
                (typ == self.typ || self.typ == Type::Any)
                    && (sub == self.sub || self.sub == SubType::Any)
            }
        }
    }
}

// see https://mimetype.io/all-types
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Type {
    None,
    Any,
    Application,
    Audio,
    Font,
    Image,
    Message,
    Multipart,
    Text,
    Video,
}

impl From<&str> for Type {
    fn from(v: &str) -> Self {
        match v {
            "*" => Self::Any,
            "application" => Self::Application,
            "audio" => Self::Audio,
            "font" => Self::Font,
            "image" => Self::Image,
            "multipart" => Self::Multipart,
            "text" => Self::Text,
            "video" => Self::Video,
            _ => Self::None,
        }
    }
}

impl Type {
    #[inline]
    pub const fn is_none(self) -> bool {
        (self as u16) == (Self::None as u16)
    }
}

// see https://mimetype.io/all-types
// and https://www.iana.org/assignments/media-types/media-types.xhtml
/// SubType is a mimetype subtype.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SubType {
    None,
    Ac3,
    Acc,
    Aiff,
    Any,
    Atom,
    Avif,
    Basic,
    Bmp,
    BytesRange,
    Bzip,
    Bzip2,
    Calendar,
    Css,
    Csv,
    Digest,
    Dns,
    DnsJson,
    DnsMessage,
    Encrypted,
    Epub,
    Example,
    Flac,
    FormData,
    Gif,
    Global,
    Gzip,
    Html,
    Http,
    Icon,
    Javascript,
    Jpeg,
    Json,
    Jsonld,
    Markdown,
    Mathml,
    Midi,
    Mp4,
    Mpa,
    Mpeg,
    Msword,
    OctetStream,
    Ogg,
    Opus,
    Otf,
    Pdf,
    Plain,
    Png,
    Raw,
    RichText,
    Rss,
    Rtf,
    Rtx,
    Sgml,
    Signed,
    Sql,
    Svg,
    Tiff,
    Troff,
    Tsv,
    Ttf,
    UriList,
    UrlEncoded,
    Wasm,
    Webm,
    Webp,
    Woff,
    Woff2,
    Xhtml,
    Xml,
    Yaml,
    Zip,
}

impl From<&str> for SubType {
    #[inline]
    fn from(v: &str) -> Self {
        Self::parse(v)
    }
}

macro_rules! is_one_of {
    ($item:expr; [$one:expr]) => {
        (($item as u16) == ($one as u16))
    };
    ($item:expr; [$one:expr, $($args:expr),+]) => {
        (($item as u16) == ($one as u16)) || is_one_of!($item; [$($args),+])
    };
}

impl SubType {
    #[inline]
    pub fn can_audio(&self) -> bool {
        self.valid_type(Type::Audio)
    }

    #[inline]
    pub fn can_video(&self) -> bool {
        self.valid_type(Type::Video)
    }

    #[inline]
    pub fn can_image(&self) -> bool {
        self.valid_type(Type::Image)
    }

    #[inline]
    pub fn can_font(&self) -> bool {
        self.valid_type(Type::Font)
    }

    #[inline]
    pub fn can_text(&self) -> bool {
        self.valid_type(Type::Text)
    }

    const fn valid_type(&self, typ: Type) -> bool {
        if (typ as u16) == (Type::Any as u16) {
            return true;
        }
        match self {
            Self::None => false,
            Self::Any => !typ.is_none(),
            Self::Ogg => is_one_of!(typ; [Type::Application, Type::Audio, Type::Video]),
            Self::Mp4 => is_one_of!(typ; [Type::Application, Type::Audio, Type::Video]),
            Self::Mpeg | Self::Webm => is_one_of!(typ; [Type::Audio, Type::Video]),
            Self::Jpeg | Self::Raw => is_one_of!(typ; [Type::Image, Type::Video]),
            Self::Rtf => is_one_of!(typ; [Type::Application, Type::Text]),
            Self::Rtx => is_one_of!(typ; [Type::Application, Type::Audio, Type::Video, Type::Text]),
            Self::Atom
            | Self::Bzip
            | Self::Bzip2
            | Self::Dns
            | Self::DnsJson
            | Self::DnsMessage
            | Self::Epub
            | Self::Gzip
            | Self::Json
            | Self::Jsonld
            | Self::Msword
            | Self::OctetStream
            | Self::Pdf
            | Self::Rss
            | Self::Sql
            | Self::UrlEncoded
            | Self::Wasm
            | Self::Xhtml
            | Self::Xml
            | Self::Yaml
            | Self::Zip => is_one_of!(typ; [Type::Application]),
            Self::Avif
            | Self::Bmp
            | Self::Gif
            | Self::Icon
            | Self::Png
            | Self::Svg
            | Self::Tiff
            | Self::Webp => is_one_of!(typ; [Type::Image]),
            Self::Acc
            | Self::Ac3
            | Self::Aiff
            | Self::Basic
            | Self::Flac
            | Self::Midi
            | Self::Mpa
            | Self::Opus => is_one_of!(typ; [Type::Audio]),
            Self::Calendar
            | Self::Css
            | Self::Csv
            | Self::Html
            | Self::Javascript
            | Self::Markdown
            | Self::Mathml
            | Self::Plain
            | Self::RichText
            | Self::Sgml
            | Self::Troff
            | Self::Tsv
            | Self::UriList => is_one_of!(typ; [Type::Text]),
            Self::Otf | Self::Ttf | Self::Woff | Self::Woff2 => is_one_of!(typ; [Type::Font]),
            Self::Example => is_one_of!(typ; [
                Type::Application,
                Type::Audio,
                Type::Image,
                Type::Message,
                Type::Multipart
            ]),
            Self::Http | Self::Global => is_one_of!(typ; [Type::Message]),
            Self::BytesRange | Self::FormData | Self::Encrypted | Self::Digest | Self::Signed => {
                is_one_of!(typ; [Type::Multipart])
            }
        }
    }

    pub fn compressed(&self) -> bool {
        match self {
            Self::Gzip | Self::Zip | Self::Bzip | Self::Bzip2 => true,
            _ => false,
        }
    }

    fn parse(v: &str) -> Self {
        match v {
            "*" => Self::Any,
            "ac3" => Self::Ac3,
            "acc" => Self::Acc,
            "aiff" | "x-aiff" => Self::Aiff,
            "atom+xml" => Self::Atom,
            "avif" => Self::Avif,
            "basic" => Self::Basic,
            "bmp" => Self::Bmp,
            "bytesrange" => Self::BytesRange,
            "calendar" => Self::Calendar,
            "css" => Self::Css,
            "csv" => Self::Csv,
            "digest" => Self::Digest,
            "dns" => Self::Dns,
            "dns+json" => Self::DnsJson,
            "dns+message" => Self::DnsMessage,
            "encrypted" => Self::Encrypted,
            "epub+zip" => Self::Epub,
            "example" => Self::Example,
            "flac" => Self::Flac,
            "form-data" => Self::FormData,
            "global" => Self::Global,
            "gzip" | "x-gzip" => Self::Gzip,
            "html" => Self::Html,
            "http" => Self::Http,
            "javascript" | "ecmascript" => Self::Javascript,
            "jpeg" => Self::Jpeg,
            "json" => Self::Json,
            "ld+json" => Self::Jsonld,
            "markdown" | "x-markdown" => Self::Markdown,
            "mathml" => Self::Mathml,
            "midi" => Self::Midi,
            "mp4" => Self::Mp4,
            "mpa" => Self::Mpa,
            "mpeg" => Self::Mpeg,
            "msword" => Self::Msword,
            "octet-stream" => Self::OctetStream,
            "ogg" => Self::Ogg,
            "opus" => Self::Opus,
            "otf" | "x-font-otf" => Self::Otf,
            "pdf" => Self::Pdf,
            "plain" => Self::Plain,
            "png" => Self::Png,
            "richtext" => Self::RichText,
            "rss+xml" => Self::Rss,
            "rtf" => Self::Rtf,
            "rtx" => Self::Rtx,
            "sgml" => Self::Sgml,
            "signed" => Self::Signed,
            "sql" => Self::Sql,
            "svg" | "svg+xml" => Self::Svg,
            "tab-separated-values" => Self::Tsv,
            "tiff" => Self::Tiff,
            "troff" => Self::Troff,
            "ttf" | "x-font-ttf" => Self::Ttf,
            "uri-list" => Self::UriList,
            "wasm" => Self::Wasm,
            "webm" => Self::Webm,
            "webp" => Self::Webp,
            "woff" => Self::Woff,
            "woff2" => Self::Woff2,
            "x-bzip" => Self::Bzip,
            "x-bzip2" => Self::Bzip2,
            "x-icon" => Self::Icon,
            "x-www-form-urlencoded" => Self::UrlEncoded,
            "xhtml+xml" => Self::Xhtml,
            "xml" => Self::Xml,
            "yaml" | "yml" => Self::Yaml,
            "zip" | "zip-compressed" | "x-zip-compressed" => Self::Zip,
            _ => Self::None,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn mime_from() {
        let m = Mime::from("application/xml");
        assert_eq!(m.typ, Type::Application);
        assert_eq!(m.sub, SubType::Xml);
        assert!(m.matches_str("*/*"));
        assert!(m.matches_str("application/*"));
        assert!(m.matches_str("*/xml"));
        let any = Mime::any();
        assert!(any.matches_str("application/yaml"));
        assert!(any.matches_str("application/*"));
        assert!(any.matches_str("*/json"));
        assert!(any.matches_str("*/*"));
    }

    #[test]
    fn type_validity() {
        let mut m = Mime::from("application/xml");
        assert!(m.valid());
        m = Mime::default();
        assert!(!m.valid());
        for s in [
            "audio/json",
            "image/zip",
            "text/ogg",
            "video/png",
            "audio/x-bzip",
        ] {
            assert!(!Mime::from(s).valid());
        }
        for s in [
            "*/*",
            "application/*",
            "audio/mp4",
            "application/ogg",
            "video/ogg",
            "audio/ogg",
            "*/x-bzip",
        ] {
            assert!(Mime::from(s).valid());
        }
    }

    #[test]
    fn can() {
        use super::SubType as S;
        for s in [S::Mpeg, S::Ogg, S::Mp4, S::Webm] {
            for t in [Type::Audio, Type::Video] {
                assert!(s.valid_type(t));
            }
        }
        for s in [S::Ogg, S::Mp4] {
            for t in [Type::Audio, Type::Video, Type::Application] {
                assert!(s.valid_type(t));
            }
        }
        assert!(S::Ogg.valid_type(Type::Audio));
    }
}
