module harrybrown.com

// See list of available go versions supported by heroku:
// https://github.com/heroku/heroku-buildpack-go/blob/main/data.json#L3

// +heroku goVersion go1.18
go 1.18

require (
	github.com/aws/aws-sdk-go v1.44.24
	github.com/go-chi/chi/v5 v5.0.7
	github.com/go-redis/redis/v8 v8.11.4
	github.com/golang-jwt/jwt/v4 v4.2.0
	github.com/golang-migrate/migrate/v4 v4.7.0
	github.com/google/go-github/v43 v43.0.0
	github.com/google/uuid v1.3.0
	github.com/grafana/loki v1.6.1
	github.com/imdario/mergo v0.3.13
	github.com/joho/godotenv v1.4.0
	github.com/labstack/echo/v4 v4.6.1
	github.com/labstack/gommon v0.3.0
	github.com/lib/pq v1.10.3
	github.com/minio/madmin-go v1.3.16
	github.com/minio/minio-go/v7 v7.0.27
	github.com/ory/hydra-client-go v1.11.8
	github.com/oschwald/geoip2-golang v1.7.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.12.2
	github.com/sendgrid/rest v2.6.8+incompatible
	github.com/sendgrid/sendgrid-go v3.11.0+incompatible
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/oauth2 v0.0.0-20220411215720-9780585627b5
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
	google.golang.org/grpc v1.47.0
	nhooyr.io/websocket v1.8.7
)

require (
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-github v17.0.0+incompatible // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.1.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.13.5 // indirect
	github.com/klauspost/cpuid v1.3.1 // indirect
	github.com/klauspost/cpuid/v2 v2.0.4 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/minio/md5-simd v1.1.0 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/oschwald/maxminddb-golang v1.9.0 // indirect
	github.com/philhofer/fwd v1.1.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/prometheus/prometheus v1.8.2-0.20210321183757-31a518faab18 // indirect
	github.com/rs/xid v1.2.1 // indirect
	github.com/secure-io/sio-go v0.3.1 // indirect
	github.com/shirou/gopsutil/v3 v3.21.6 // indirect
	github.com/tinylib/msgp v1.1.3 // indirect
	github.com/tklauser/go-sysconf v0.3.6 // indirect
	github.com/tklauser/numcpus v0.2.2 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.1 // indirect
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20220325203850-36772127a21f // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20201214200347-8c77b98c765d // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/ini.v1 v1.57.0 // indirect
)

// Testing
require (
	github.com/golang/mock v1.6.0
	github.com/matryer/is v1.4.0
)
