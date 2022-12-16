variable "REGISTRY" {
    # default = "registry.digitalocean.com/webreef"
    default = "10.0.0.11:5000"
}

variable "VERSION" {
    default = "latest"
}

variable "GIT_COMMIT" {
    default = "dev"
}

variable "GIT_BRANCH" {
    default = "dev"
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

        "provision",
        "ansible",
        "curl",
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

function "tags" {
    params = [name, extra_labels]
    result = concat(
        [
            "${REGISTRY}/harrybrwn/${name}:latest",
            "${REGISTRY}/harrybrwn/${name}:${VERSION}",
            notequal("", GIT_COMMIT) ? "${REGISTRY}/harrybrwn/${name}:${GIT_COMMIT}" : "",
            notequal("", GIT_BRANCH) ? "${REGISTRY}/harrybrwn/${name}:${GIT_BRANCH}" : "",
        ],
        [for t in extra_labels : "${REGISTRY}/harrybrwn/${name}:${t}"],
    )
}

function "labels" {
    params = []
    result = {
        "git.commit"      = "${GIT_COMMIT}"
        "git.branch"      = "${GIT_BRANCH}"
        "version"         = "${VERSION}"
        "docker.registry" = "${REGISTRY}"
    }
}

target "base-service" {
    labels = labels()
    platforms = platforms
}

target "nginx" {
    target = "nginx"
    args = {
        NGINX_VERSION = "1.23.3-alpine"
        REGISTRY_UI_ROOT = "/var/www/registry.hrry.dev"
    }
    tags = tags("nginx", ["1.23.3-alpine", "1.23.3"])
    inherits = ["base-service"]
    platforms = ["linux/amd64", "linux/arm/v7"]
}

target "api" {
    target = "api"
    tags = tags("api", [])
    inherits = ["base-service"]
}

target "hooks" {
    target = "hooks"
    tags = tags("hooks", [])
    inherits = ["base-service"]
}

target "backups" {
    target = "backups"
    tags = tags("backups", [])
    inherits = ["base-service"]
}

target "geoip" {
    target = "geoip"
    tags = tags("geoip", [])
    inherits = ["base-service"]
}

target "legacy-site" {
    target = "legacy-site"
    tags = tags("legacy-site", [])
    inherits = ["base-service"]
}

target "vanity-imports" {
    target = "vanity-imports"
    tags = tags("vanity-imports", [])
    inherits = ["base-service"]
}

target "postgres" {
    context = "config/docker/postgres"
    args = {
        BASE_IMAGE_VERSION = "13.6-alpine"
    }
    tags = tags("postgres", ["13.6-alpine", "13.6"])
    inherits = ["base-service"]
}

target "fluentbit" {
    dockerfile = "config/docker/Dockerfile.fluentbit"
    args = {
        #FLUENTBIT_VERSION = "1.9.3"
        FLUENTBIT_VERSION = "1.9.3-debug"
    }
    tags = tags("fluent-bit", ["1.9.3"])
    inherits = ["base-service"]
}

target "grafana" {
    dockerfile = "config/grafana/Dockerfile"
    args = {
        GRAFANA_VERSION = "9.1.4"
    }
    tags = tags("grafana", ["9.1.4"])
    inherits = ["base-service"]
}

target "loki" {
    dockerfile = "config/docker/Dockerfile.loki"
    args = {
        LOKI_VERSION = "2.5.0"
    }
    tags = tags("loki", ["2.5.0"])
    inherits = ["base-service"]
}

target "redis" {
    context = "config/redis"
    dockerfile = "Dockerfile"
    args = {
        REDIS_VERSION = "6.2.6-alpine"
    }
    tags = tags("redis", ["6.2.6", "6.2.6-alpine"])
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
    tags = tags("s3", ["RELEASE.2022-05-23T18-45-11Z.fips"])
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
    tags = tags("outline", ["0.66.0"])
    labels = labels()
    inherits = ["base-service"]
}

target "nomad" {
    context = "."
    dockerfile = "config/nomad/Dockerfile"
    target = "nomad"
    args = {
        ALPINE_VERSION = "3.14"
        NOMAD_VERSION = "1.3.5"
    }
    platforms = platforms
    tags = concat(
        tags("nomad", ["1.3.5", "1.3.5-alpine"]),
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
        ALPINE_VERSION = "3.16"
    }
    tags = tags("curl", ["3.16"])
    platforms = [
        "linux/amd64",
        "linux/arm/v7",
        "linux/arm/v6",
    ]
}

target "provision" {
    target = "provision"
    tags = tags("provision", [])
    inherits = ["base-service"]
}

target "ansible" {
    dockerfile = "config/ansible/Dockerfile"
    labels = labels()
    tags = tags("ansible", [])
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

#
# Special Builds
#

target "frontend" {
    target = "raw-frontend"
    output = ["./build"]
}
