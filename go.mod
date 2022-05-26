module harrybrown.com

// See list of available go versions supported by heroku:
// https://github.com/heroku/heroku-buildpack-go/blob/main/data.json#L3

// +heroku goVersion go1.18
go 1.18

require (
	github.com/go-chi/chi/v5 v5.0.7
	github.com/go-redis/redis/v8 v8.11.4
	github.com/golang-jwt/jwt/v4 v4.2.0
	github.com/google/go-github/v43 v43.0.0
	github.com/google/uuid v1.3.0
	github.com/joho/godotenv v1.4.0
	github.com/labstack/echo/v4 v4.6.1
	github.com/labstack/gommon v0.3.0
	github.com/lib/pq v1.10.3
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.12.2
	github.com/sendgrid/rest v2.6.8+incompatible
	github.com/sendgrid/sendgrid-go v3.11.0+incompatible
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/oauth2 v0.0.0-20220411215720-9780585627b5
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
	nhooyr.io/websocket v1.8.7
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/klauspost/compress v1.10.3 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.1 // indirect
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd // indirect
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.26.0 // indirect
)

// Testing
require (
	github.com/golang/mock v1.6.0
	github.com/matryer/is v1.4.0
)
