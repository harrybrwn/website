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
        "db",
        "s3",
    ]
}

function "tags" {
    params = [name]
    result = [
        "${REGISTRY}/harrybrwn/${name}:latest",
        "${REGISTRY}/harrybrwn/${name}:${VERSION}",
        "${REGISTRY}/harrybrwn/${name}:${GIT_COMMIT}",
        "${REGISTRY}/harrybrwn/${name}:${GIT_BRANCH}",
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

target "nginx" {
    platforms = platforms
    tags = tags("nginx")
}

target "api" {
    target = "api"
    context = "."
    dockerfile = "Dockerfile"
    args = {
        ALPINE_VERSION = "3.14"
    }
    labels = labels()
    tags = tags("api")
    platforms = platforms
}

target "hooks" {
    labels = labels()
    tags = tags("hooks")
    platforms = platforms
}

target "backups" {
    labels = labels()
    tags = tags("backups")
    platforms = platforms
}

target "geoip" {
    labels = labels()
    tags = tags("geoip")
    platforms = platforms
}

target "legacy-site" {
    labels = labels()
    tags = tags("legacy-site")
    platforms = platforms
}

target "vanity-imports" {
    labels = labels()
    tags = tags("vanity-imports")
    platforms = platforms
}

target "db" {
    args = {
        BASE_IMAGE_VERSION = "13.6-alpine"
    }
    labels = labels()
    tags = tags("postgres")
    platforms = platforms
}

target "fluentbit" {
    dockerfile = "config/docker/Dockerfile.fluentbit"
    args = {
        FLUENTBIT_VERSION = "1.9.3"
    }
    labels = labels()
    tags = tags("fluent-bit")
    platforms = platforms
}

target "grafana" {
    dockerfile = "config/docker/Dockerfile.grafana"
    args = {
        GRAFANA_VERSION = "latest"
    }
    labels = labels()
    tags = tags("grafana")
    platforms = platforms
}

target "loki" {
    dockerfile = "config/docker/Dockerfile.loki"
    args = {
        LOKI_VERSION = "2.5.0"
    }
    labels = labels()
    tags = tags("loki")
    platforms = platforms
}

target "redis" {
    dockerfile = "config/redis/Dockerfile"
    args = {
        REDIS_VERSION = "6.2.6-alpine"
    }
    labels = labels()
    tags = concat(
        tags("redis"),
        formatlist("${REGISTRY}/harrybrwn/redis:6.2.6-alpine")
    )
    platforms = platforms
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
    labels = labels()
    tags = tags("provision")
    platforms = platforms
}

target "ansible" {
    dockerfile = "config/ansible/Dockerfile"
    labels = labels()
    tags = tags("ansible")
    platforms = ["linux/amd64"]
}