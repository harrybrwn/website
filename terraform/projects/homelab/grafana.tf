provider "grafana" {
  url  = "https://grafana.lab.hrry.dev"
  auth = var.grafana_api_key
}

resource "grafana_data_source" "prometheus" {
  type               = "prometheus"
  name               = "Prometheus"
  url                = "http://prometheus-stack-kube-prom-prometheus.prometheus.svc.cluster.local:9090"
  basic_auth_enabled = false
  json_data_encoded = jsonencode({
    httpMethod    = "POST"
    manageAlerts  = true
    defaultEditor = "builder" # (builder|code)
    #disableMetricsLookup = false
  })
}

locals {
  nodes = toset([
    { name = "hp-laptop" },
    { name = "lenovo" },
    { name = "rpi1" },
    { name = "rpi2" },
    { name = "rpi3" },
  ])
  dashboards_dir = "../../../config/grafana/dashboards"
}

resource "grafana_folder" "node_alerts_folder" {
  title = "Node Metric Alerts"
}

resource "grafana_dashboard" "node_resources" {
  config_json = jsonencode(merge(jsondecode(file("${local.dashboards_dir}/node-resources.json")), {
    editable = false
    time = {
      from = "now-1h"
      to   = "now"
    }
  }))
  folder    = grafana_folder.node_alerts_folder.id
  overwrite = true
}

locals {
  geoip_dashboard_config = jsondecode(file("${local.dashboards_dir}/geoip.json"))
}

resource "random_uuid" "geoip_dashboard" {}

resource "grafana_dashboard" "geoip" {
  config_json = jsonencode(merge(local.geoip_dashboard_config, {
    editable = false
    id       = 11
    title    = "GeoIP"
    time = {
      from = "now-6h"
      to   = "now"
    }
    tags = [
      "Geolocation",
      "HTTP Gateway",
    ]
    panels  = local.geoip_dashboard_config["panels"]
    version = 1
    uid     = random_uuid.geoip_dashboard.result
  }))
  overwrite = true
}

resource "random_uuid" "geoip_demo_dashboard" {}

resource "grafana_dashboard" "geoip_demo" {
  config_json = jsonencode(merge(jsondecode(file("${local.dashboards_dir}/geoip.json")), {
    editable = true
    title    = "GeoIP (Duplicate)"
    panels   = local.geoip_dashboard_config["panels"]
    time = {
      from = "now-6h"
      to   = "now"
    }
    id      = 12
    version = 1
    uid     = random_uuid.geoip_demo_dashboard.result
  }))
  overwrite = true
}

# resource "grafana_dashboard" "nginx" {
#   # config_json = jsonencode(merge(jsondecode(file("${local.dashboards_dir}/nginx.json")), {}))
#   config_json = file("${local.dashboards_dir}/nginx.json")
# }

resource "grafana_folder" "kubernetes" {
  title = "k8s"
}

resource "grafana_dashboard" "kubernetes" {
  config_json = jsonencode(merge(jsondecode(file("../../../config/grafana/dashboards/kubernetes.json")), {
    editable = true
    time = {
      from = "now-3h"
      to   = "now"
    }
  }))
  folder    = grafana_folder.kubernetes.id
  overwrite = true
}

module "k8s_alerts" {
  source           = "../../modules/grafana/alert"
  datasource_uid   = grafana_data_source.prometheus.uid
  folder_uid       = grafana_folder.kubernetes.uid
  name             = "Kubernetes Alerts"
  interval_seconds = 60 * 5
  rules = [
    {
      name        = "Pod Restarts"
      summary     = "Pod restart events"
      description = "Triggers an alert if a pod restarts."
      duration    = "0s"
      dashboard   = { uid = grafana_dashboard.kubernetes.uid, panel_id = 0 }
      query       = <<EOT
        sum by (nodename, pod, container, service) (
          changes(kube_pod_container_status_restarts_total{}[1m])
        )
      EOT
      condition = {
        op      = "gt"
        args    = [0]
        reducer = "last"
      }
    }
  ]
}

resource "grafana_dashboard" "traefik" {
  config_json = jsonencode(merge(jsondecode(file("traefik.json")), {
    editable = false
  }))
}
