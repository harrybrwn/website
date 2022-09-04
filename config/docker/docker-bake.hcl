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
    params = [name]
    result = [
        "${REGISTRY}/harrybrwn/${name}:latest",
        "${REGISTRY}/harrybrwn/${name}:${VERSION}",
        notequal("", GIT_COMMIT) ? "${REGISTRY}/harrybrwn/${name}:${GIT_COMMIT}" : "",
        notequal("", GIT_BRANCH) ? "${REGISTRY}/harrybrwn/${name}:${GIT_BRANCH}" : "",
    ]
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
    tags = tags("nginx")
    inherits = ["base-service"]
    platforms = ["linux/amd64"]
}

target "api" {
    target = "api"
    tags = tags("api")
    inherits = ["base-service"]
}

target "hooks" {
    target = "hooks"
    tags = tags("hooks")
    inherits = ["base-service"]
}

target "backups" {
    target = "backups"
    tags = tags("backups")
    inherits = ["base-service"]
}

target "geoip" {
    target = "geoip"
    tags = tags("geoip")
    inherits = ["base-service"]
}

target "legacy-site" {
    target = "legacy-site"
    tags = tags("legacy-site")
    inherits = ["base-service"]
}

target "vanity-imports" {
    target = "vanity-imports"
    tags = tags("vanity-imports")
    inherits = ["base-service"]
}

target "postgres" {
    context = "config/docker/postgres"
    args = {
        BASE_IMAGE_VERSION = "13.6-alpine"
    }
    tags = concat(
        tags("postgres"),
        formatlist("${REGISTRY}/harrybrwn/postgres:13.6-alpine"),
    )
    inherits = ["base-service"]
}

target "fluentbit" {
    dockerfile = "config/docker/Dockerfile.fluentbit"
    args = {
        FLUENTBIT_VERSION = "1.9.3"
    }
    tags = tags("fluent-bit")
    inherits = ["base-service"]
}

target "grafana" {
    dockerfile = "config/grafana/Dockerfile"
    args = {
        GRAFANA_VERSION = "latest"
    }
    tags = tags("grafana")
    inherits = ["base-service"]
}

target "loki" {
    dockerfile = "config/docker/Dockerfile.loki"
    args = {
        LOKI_VERSION = "2.5.0"
    }
    tags = tags("loki")
    inherits = ["base-service"]
}

target "redis" {
    context = "config/redis"
    dockerfile = "Dockerfile"
    args = {
        REDIS_VERSION = "6.2.6-alpine"
    }
    tags = concat(
        tags("redis"),
        formatlist("${REGISTRY}/harrybrwn/redis:6.2.6-alpine")
    )
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
    tags = tags("s3")
    platforms = [
        "linux/amd64",
    ]
}

#
# Tools
#

target "provision" {
    target = "provision"
    tags = tags("provision")
    inherits = ["base-service"]
}

target "ansible" {
    dockerfile = "config/ansible/Dockerfile"
    labels = labels()
    tags = tags("ansible")
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