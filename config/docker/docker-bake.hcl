variable "REGISTRY" {
    # default = "registry.digitalocean.com/webreef"
    default = "10.0.0.11:5000"
}

variable "VERSION" {
    default = "latest"
}

variable "GIT_COMMIT" {
    default = ""
}

variable "GIT_BRANCH" {
    default = "dev"
}

variable "ALPINE_VERSION"  { default = "3.17.0" }
variable "GO_VERSION"      { default = "1.18-alpine" }
variable "GRAFANA_VERSION" { default = "9.4.7" }
variable "RUST_BASE"       { default = "alpine3.16"}

variable "FLUENTBIT_VERSION" {
    default = "1.9.10"
    #default = "2.0.6"
}

variable "platforms" {
    default = [
        "linux/amd64",
        "linux/arm/v7",
    ]
}

group "default" {
    targets = [
        "nginx",
        "databases",
        "services",
        "logging",
        "tools",
    ]
}

group "services" {
    targets = [
        "api",
        "hooks",
        "backups",
        "geoip",
        "legacy-site",
        "vanity-imports",
        "outline",
    ]
}

group "logging" {
    targets = [
        "fluentbit",
        "grafana",
        "loki",
    ]
}

group "databases" {
    targets = [
        "redis",
        "postgres",
        "s3",
    ]
}

group "tools" {
    targets = [
        "provision",
        "ansible",
        "curl",
        "rust",
    ]
}

variable "_IS_LOCAL" { default = false }

function "tags" {
    params = [registry, name, extra_labels]
    result = concat(
        [
            join("/", compact([registry, "harrybrwn", "${name}:latest"])),
        ],
        _IS_LOCAL ? [] : concat(
            [
                join("/", compact([registry, "harrybrwn", "${name}:${VERSION}"])),
                notequal("", GIT_COMMIT) ?
                    join("/", compact([registry, "harrybrwn", "${name}:${GIT_COMMIT}"])) :
                    "",
                notequal("", GIT_BRANCH) ?
                    join("/", compact([registry, "harrybrwn", "${name}:${GIT_BRANCH}"])) :
                    "",
            ],
            [
                for t in compact(extra_labels) :
                    join("/", compact([registry, "harrybrwn", "${name}:${t}"]))
            ],
        ),
    )
}

function "labels" {
    params = []
    result = {
        "git.commit"      = "${GIT_COMMIT}"
        "git.branch"      = "${GIT_BRANCH}"
        "version"         = "${VERSION}"
        "docker.registry" = "${REGISTRY}"
        "author"          = "Harry Brown"
    }
}

target "base-service" {
    labels = labels()
    platforms = platforms
    args = {
        ALPINE_VERSION = ALPINE_VERSION
        GO_VERSION     = GO_VERSION
    }
}

target "nginx" {
    target = "nginx"
    args = {
        NGINX_VERSION = "1.23.3-alpine"
        REGISTRY_UI_ROOT = "/var/www/registry.hrry.dev"
    }
    tags = tags(REGISTRY, "nginx", ["1.23.3-alpine", "1.23.3"])
    inherits = ["base-service"]
    platforms = [
        "linux/amd64",
        "linux/arm/v7",
    ]
}

target "api" {
    target = "api"
    tags = tags(REGISTRY, "api", [])
    inherits = ["base-service"]
}

target "hooks" {
    target = "hooks"
    tags = tags(REGISTRY, "hooks", [])
    inherits = ["base-service"]
}

target "backups" {
    target = "backups"
    tags = tags(REGISTRY, "backups", [])
    inherits = ["base-service"]
}

group "geoip" { targets = ["geoip-rs"] }

target "go-geoip" {
    target = "go-geoip"
    tags = tags(REGISTRY, "go-geoip", [])
    inherits = ["base-service"]
}

target "geoip-rs" {
    context  = "services/geoip"
    tags     = tags("", "geoip", []) // publish to dockerhub
    inherits = ["base-service"]
}

target "legacy-site" {
    target = "legacy-site"
    tags = tags(REGISTRY, "legacy-site", [])
    inherits = ["base-service"]
}

target "vanity-imports" {
    target = "vanity-imports"
    tags = tags(REGISTRY, "vanity-imports", [])
    inherits = ["base-service"]
}

target "postgres" {
    context = "config/docker/postgres"
    args = {
        BASE_IMAGE_VERSION = "13.6-alpine"
    }
    tags = tags(REGISTRY, "postgres", ["13.6-alpine", "13.6"])
    inherits = ["base-service"]
}

target "fluentbit" {
    dockerfile = "config/docker/Dockerfile.fluentbit"
    args = {
        FLUENTBIT_VERSION = FLUENTBIT_VERSION
        #FLUENTBIT_VERSION = "${FLUENTBIT_VERSION}-debug"
    }
    tags = tags(REGISTRY, "fluent-bit", [FLUENTBIT_VERSION])
    inherits = ["base-service"]
}

target "grafana" {
    dockerfile = "config/grafana/Dockerfile"
    args = {
        GRAFANA_VERSION = "9.4.7"
    }
    tags = tags(REGISTRY, "grafana", [GRAFANA_VERSION])
    inherits = ["base-service"]
}

target "loki" {
    dockerfile = "config/docker/Dockerfile.loki"
    args = {
        LOKI_VERSION = "2.5.0"
    }
    tags = tags(REGISTRY, "loki", ["2.5.0"])
    inherits = ["base-service"]
}

target "redis" {
    context = "config/redis"
    dockerfile = "Dockerfile"
    args = {
        REDIS_VERSION = "6.2.6-alpine"
    }
    tags = tags(REGISTRY, "redis", ["6.2.6", "6.2.6-alpine"])
    inherits = ["base-service"]
}

target "s3" {
    context = "./config"
    dockerfile = "docker/minio/Dockerfile"
    args = {
        MINIO_VERSION = "RELEASE.2022-05-23T18-45-11Z.fips"
        MC_VERSION = "RELEASE.2022-05-09T04-08-26Z.fips"
    }
    labels = labels()
    tags = tags(REGISTRY, "s3", ["RELEASE.2022-05-23T18-45-11Z.fips"])
    platforms = [
        "linux/amd64",
    ]
}

target "outline" {
    context = "."
    dockerfile = "config/docker/Dockerfile.outline"
    args = {
        OUTLINE_VERSION = "0.66.0"
    }
    tags = tags(REGISTRY, "outline", ["0.66.0"])
    labels = labels()
    inherits = ["base-service"]
}

target "nomad" {
    context = "."
    dockerfile = "config/nomad/Dockerfile"
    target = "nomad"
    args = {
        ALPINE_VERSION = ALPINE_VERSION
        NOMAD_VERSION = "1.3.5"
    }
    platforms = platforms
    tags = concat(
        tags(REGISTRY, "nomad", ["1.3.5", "1.3.5-alpine"]),
        [
            # There is no private information in this docker image so I'm
            # pushing to dockerhub.
            "harrybrwn/nomad:latest",
            "harrybrwn/nomad:1.3.5",
            "harrybrwn/nomad:1.3.5-alpine",
        ],
    )
}

#
# Tools
#

target "curl" {
    dockerfile = "config/docker/Dockerfile.curl"
    labels = labels()
    args = {
        ALPINE_VERSION = ALPINE_VERSION
    }
    tags = tags(REGISTRY, "curl", [ALPINE_VERSION])
    platforms = [
        "linux/amd64",
        "linux/arm/v7",
        "linux/arm/v6",
    ]
}

target "provision" {
    target = "provision"
    tags = tags(REGISTRY, "provision", [])
    inherits = ["base-service"]
}

target "ansible" {
    dockerfile = "config/ansible/Dockerfile"
    labels = labels()
    tags = tags(REGISTRY, "ansible", [])
    platforms = ["linux/amd64"]
}

target "service" {
    target = "service"
    output = ["type=cacheonly"]
}

target "wait" {
    target = "wait"
    output = ["./.tmp/wait"]
}

group "rust" {
    targets = [
        for i in [
            68,
            69,
        ]:
        "rust_1-${i}-0"
    ]
}

target "rust-base" {
    context    = "config/docker/rust"
    dockerfile = "Dockerfile"
    labels     = labels()
    platforms  = ["linux/amd64"]
    args       = {RUST_BASE = RUST_BASE}
}

function "rust_tags" {
    params = [version, latest]
    result = [
        for t in compact(concat([
            version,
            RUST_BASE == "" ? "" : "${version}-${RUST_BASE}",
        ], latest ? ["latest", GIT_COMMIT] : [])) :
        "harrybrwn/rust:${t}"
    ]
}

target "rust_1-69-0" {
    inherits = ["rust-base"]
    tags = rust_tags("1.69.0", true)
    args = {RUST_VERSION = "1.69.0"}
}

target "rust_1-68-0" {
    inherits = ["rust-base"]
    tags = rust_tags("1.68.0", false)
    args = {RUST_VERSION = "1.68.0"}
}

#
# Special Builds
#

target "frontend" {
    target = "raw-frontend"
    output = ["./build"]
}
