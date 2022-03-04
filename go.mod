module harrybrown.com

// +heroku goVersion go1.17.3
go 1.17

require (
	github.com/go-redis/redis/v8 v8.11.4
	github.com/golang-jwt/jwt/v4 v4.2.0
	github.com/google/uuid v1.3.0
	github.com/joho/godotenv v1.4.0
	github.com/labstack/echo/v4 v4.6.1
	github.com/labstack/gommon v0.3.0
	github.com/lib/pq v1.10.3
	github.com/pkg/errors v0.9.1
	github.com/sendgrid/rest v2.6.8+incompatible
	github.com/sendgrid/sendgrid-go v3.11.0+incompatible
	github.com/sirupsen/logrus v1.8.1
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324
	nhooyr.io/websocket v1.8.7
)

require (
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/klauspost/compress v1.10.3 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.1 // indirect
	golang.org/x/net v0.0.0-20210913180222-943fd674d43e // indirect
	golang.org/x/sys v0.0.0-20210910150752-751e447fb3d0 // indirect
	golang.org/x/text v0.3.7 // indirect
)

// Testing
require (
	github.com/golang/mock v1.6.0
	github.com/matryer/is v1.4.0
)
