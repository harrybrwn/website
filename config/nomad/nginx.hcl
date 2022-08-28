job "web" {
  datacenters = ["dc1"]
  type = "service"

  update {
    max_parallel = 1
    min_healthy_time = "10s"
    healthy_deadline = "3m"
    progress_deadline = "10m"
    auto_revert = false
    canary = 0
  }
  migrate {
    max_parallel = 1
    health_check = "checks"
    min_healthy_time = "10s"
    healthy_deadline = "5m"
  }

  group "nginx" {
    count = 1

    network {
      mode = "host"
      port "http" {
        static = 80
      }
      port "https" {
        static = 443
      }
    }

    service {
      name     = "nginx-server"
      tags     = ["global", "web", "gateway"]
      port     = "http"
      provider = "nomad"
    }

    volume "certs" {
      type = "host"
      source = "certs"
      read_only = true
    }

    restart {
      attempts = 2
      interval = "30m"
      delay = "15s"
      mode = "fail"
    }

    ephemeral_disk {
      # When sticky is true and the task group is updated, the scheduler
      # will prefer to place the updated allocation on the same node and
      # will migrate the data. This is useful for tasks that store data
      # that should persist across allocation updates.
      #sticky = true
      #
      # Setting migrate to true results in the allocation directory of a
      # sticky allocation directory to be migrated.
      #migrate = true
      #
      # The "size" parameter specifies the size in MB of shared ephemeral disk
      # between tasks in the group.
      size = 300
    }

    task "nginx" {
      driver = "docker"

      config {
        image = "10.0.0.11:5000/harrybrwn/nginx:v0.0.1"
        ports = ["http", "https"]
        # authenticate when pulling images
        auth_soft_fail = false
      }

      volume_mount {
        volume = "certs"
        destination = "/etc/harrybrwn/certs"
        read_only = true
      }

      resources {
        cpu    = 500 # 500 MHz
        memory = 256 # 256MB
      }

      # The "template" stanza instructs Nomad to manage a template, such as
      # a configuration file or script. This template can optionally pull data
      # from Consul or Vault to populate runtime configuration data.
      #
      #     https://www.nomadproject.io/docs/job-specification/template
      #
      # template {
      #   data          = "---\nkey: {{ key \"service/my-key\" }}"
      #   destination   = "local/file.yml"
      #   change_mode   = "signal"
      #   change_signal = "SIGHUP"
      # }

      # The "template" stanza can also be used to create environment variables
      # for tasks that prefer those to config files. The task will be restarted
      # when data pulled from Consul or Vault changes.
      #
      # template {
      #   data        = "KEY={{ key \"service/my-key\" }}"
      #   destination = "local/file.env"
      #   env         = true
      # }
    }
  }

  group "geoip" {
    count = 1
    network {
      port "http" {
        to = 8084
      }
    }

    service {
      name = "geoip"
      port = "http"
      provider = "nomad"
    }

    restart {
      attempts = 2
      interval = "30m"
      delay = "15s"
      mode = "fail"
    }

    ephemeral_disk {
      size = 300
    }

    task "geoip" {
      driver = "docker"
      config {
        image = "10.0.0.11:5000/harrybrwn/geoip:v0.0.1"
        ports = ["http"]
        args = [
          "--port",
          "${NOMAD_PORT_http}",
        ]
        auth_soft_fail = true
      }
      resources {
        cpu    = 500 # 500 MHz
        memory = 256 # 256MB
      }
    }
  }
}
