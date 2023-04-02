provider "grafana" {
  url  = "https://grafana.lab.hrry.dev"
  auth = var.grafana_api_key
}

resource "grafana_data_source" "prometheus" {
  type               = "prometheus"
  name               = "Prometheus"
  url                = "http://prometheus.prometheus.svc.cluster.local:9090"
  basic_auth_enabled = false
  json_data_encoded = jsonencode({
    httpMethod   = "POST"
    manageAlerts = true
  })
}

locals {
  discord_role_id  = "989730245168988170"
  discord_template = "discord.message"
}

resource "grafana_message_template" "discord_template" {
  name = "Discord Message Template"
  # See https://github.com/grafana/alerting/blob/main/templates/default_template.go
  template = <<EOT
{{ define "_values_list" -}}
	{{- if len .Values }}{{ $first := true -}}
		{{- range $refID, $value := .Values -}}
		 	{{- if ne $refID "metric" -}}
				{{- if $first }}{{ $first = false }}{{ else }}, {{ end -}}
				{{- $refID }}={{ $value -}}
			{{- end -}}
		{{- end -}}
	{{- else }}[no value]{{ end -}}
{{- end }}

{{ define "discord.title" -}}
{{- if gt (.Alerts.Firing | len) 0 }}:red_circle: {{ end -}}
{{- if gt (.Alerts.Resolved | len) 0 }}:white_check_mark: {{ end -}}
[**{{ .Status | title }}**{{ if eq .Status "firing" }}:{{ .Alerts.Firing | len }}{{ if gt (.Alerts.Resolved | len) 0 }}, Resolved:{{ .Alerts.Resolved | len }}{{ end }}{{ end }}]{{" "}}{{ .GroupLabels.SortedPairs.Values | join " " }} {{ if gt (len .CommonLabels) (len .GroupLabels) }}({{ with .CommonLabels.Remove .GroupLabels.Names }}{{ .Values | join " " }}{{ end }}){{ end }}
{{- end }}

{{ define "_alert_list" -}}
{{- range . }}
Value: {{ template "_values_list" . }}
{{- end }}{{ end }}

{{ define "${local.discord_template}" }}
{{ template "discord.title" . }}
{{ template "default.message" . -}}
Got {{ len .Alerts }} alert{{ if gt 1 (len .Alerts) }}s{{ end }}.
<@&${local.discord_role_id}>
{{ end }}
EOT
}

resource "grafana_contact_point" "discord_alerts" {
  name = "Discord Alerts"
  discord {
    url                     = var.discord_webhook_url
    message                 = "{{ template \"${local.discord_template}\" . }}"
    use_discord_username    = true
    disable_resolve_message = false
  }
}

resource "grafana_notification_policy" "root" {
  # Policy for testing out one off notifications
  policy {
    matcher {
      label = "test"
      match = "="
      value = "true"
    }
    contact_point   = grafana_contact_point.discord_alerts.name
    group_by        = ["alertname"]
    group_wait      = "1s"
    group_interval  = "10s"
    repeat_interval = "10s"
  }
  # Default policy
  group_by        = ["alertname"]
  contact_point   = grafana_contact_point.discord_alerts.name
  group_interval  = "5m"  # time between notifications in a group
  repeat_interval = "25m" # time between resending a notification
  group_wait      = "30s" # time spent buffering fired alerts
}

locals {
  base_rule_model = {
    hide          = false
    intervalMs    = 1000
    maxDataPoints = 43200
  }
}

resource "grafana_folder" "alerting_rule_folder" {
  title = "Homelab Alerts"
}

module "homelab_alerts" {
  source           = "../../modules/grafana/alert"
  name             = "Homelab Alerts"
  folder_uid       = grafana_folder.alerting_rule_folder.uid
  datasource_uid   = grafana_data_source.prometheus.uid
  interval_seconds = 120
  dashboard_uid    = "MsjffzSZz"
  rules = [
    {
      name               = "NGINX Up"
      description        = "Triggers when nginx stops reporting the \"up\" metric."
      dashboard_panel_id = 15
      duration           = "2m"
      query              = "nginx_up{}"
      condition = {
        op   = "lt"
        args = [1]
      }
    },
  ]
}

locals {
  nodes = toset([
    { name = "hp-laptop" },
    { name = "lenovo" },
    { name = "rpi1" },
    { name = "rpi2" },
    { name = "rpi3" },
  ])
}

resource "grafana_folder" "node_alerts_folder" {
  title = "Node Metric Alerts"
}

resource "grafana_dashboard" "node_resources" {
  config_json = jsonencode(merge(jsondecode(file("../../../config/grafana/dashboards/node-resources.json")), {
    editable = false
    time = {
      from = "now-1h"
      to   = "now"
    }
  }))
  folder    = grafana_folder.node_alerts_folder.id
  overwrite = true
}

/*
module "node_alerts" {
  source = "../../modules/grafana/alert"

  name             = "Node Alerts"
  folder_uid       = grafana_folder.node_alerts_folder.uid
  interval_seconds = 60 * 3
  datasource_uid   = grafana_data_source.prometheus.uid
  dashboard_uid    = grafana_dashboard.node_resources.uid

  rules = concat(
    [
      for node in local.nodes :
      {
        name               = "Node Memory: \"${node.name}\""
        summary            = "Memory for node \"${node.name}\""
        description        = "Triggered when the Memory for node \"${node.name}\" does above 40%"
        dashboard_panel_id = 1
        duration           = "5m"
        query              = <<EOT
            (
              node_memory_MemTotal_bytes{node="${node.name}"} -
              node_memory_MemAvailable_bytes{node="${node.name}"}
            ) / node_memory_MemTotal_bytes
        EOT
        condition = {
          op      = "gt"
          args    = [10.0]
          reducer = "last"
        }
      }
    ],
    [
      for node in local.nodes :
      {
        name               = "Node CPU: \"${node.name}\""
        summary            = "CPU for node \"${node.name}\""
        description        = "Triggered when the CPU for node \"${node.name}\" does above the threshold"
        dashboard_panel_id = 0
        duration           = "5m"
        query              = <<EOT
            avg without (cpu) (
              rate(node_cpu_seconds_total{mode="user", node="${node.name}"}[5m]) * 100
            )
        EOT
        condition = {
          op      = "gt"
          args    = [4.5]
          reducer = "last"
        }
      }
    ],
  )
}
moved {
  from = grafana_rule_group.node_alerts
  to = module.node_alerts.grafana_rule_group.node_alerts
}
*/

resource "grafana_rule_group" "node_alerts" {
  org_id           = 1
  name             = "Node Alerts"
  folder_uid       = grafana_folder.node_alerts_folder.uid
  interval_seconds = 60 * 3

  dynamic "rule" {
    for_each = local.nodes
    iterator = item
    content {
      name = "Node Memory: \"${item.value.name}\""
      annotations = {
        summary          = "Memory for node \"${item.value.name}\""
        description      = "Triggered when the Memory for node \"${item.value.name}\" does above 40%"
        __dashboardUid__ = grafana_dashboard.node_resources.uid
        __panelId__      = "1"
      }
      for       = "5m"
      is_paused = false
      condition = "B"

      data {
        ref_id         = "A"
        datasource_uid = grafana_data_source.prometheus.uid
        relative_time_range {
          from = 600
          to   = 0
        }
        model = jsonencode(merge(local.base_rule_model, {
          expr  = <<EOT
            (
              node_memory_MemTotal_bytes{node="${item.value.name}"} -
              node_memory_MemAvailable_bytes{node="${item.value.name}"}
            ) / node_memory_MemTotal_bytes
          EOT
          range = true
          refId = "A"
        }))
      }

      data {
        ref_id         = "B"
        datasource_uid = "-100"
        relative_time_range {
          from = 0
          to   = 0
        }
        model = jsonencode(merge(local.base_rule_model, {
          conditions = [
            {
              type      = "query"
              evaluator = { type = "gt", params = [10.0] }
              operator  = { type = "and" }
              reducer   = { type = "last", params = [] }
              query     = { params = ["A"] }
            }
          ]
          refId      = "B"
          type       = "classic_conditions"
          datasource = { type = "__expr__", uid = "-100" }
        }))
      }
    }
  }

  dynamic "rule" {
    for_each = local.nodes
    iterator = item
    content {
      name = "Node CPU: \"${item.value.name}\""
      annotations = {
        summary          = "CPU for node \"${item.value.name}\""
        description      = "Triggered when the CPU for node \"${item.value.name}\" does above the threshold"
        __dashboardUid__ = grafana_dashboard.node_resources.uid
        __panelId__      = "0"
      }
      for       = "5m"
      is_paused = false
      condition = "B"

      data {
        ref_id         = "A"
        datasource_uid = grafana_data_source.prometheus.uid
        relative_time_range {
          from = 600
          to   = 0
        }
        model = jsonencode(merge(local.base_rule_model, {
          expr  = <<EOT
            avg without (cpu) (
              rate(node_cpu_seconds_total{mode="user", node="${item.value.name}"}[5m]) * 100
            )
          EOT
          range = true
          refId = "A"
        }))
      }

      data {
        ref_id         = "B"
        datasource_uid = "-100"
        relative_time_range {
          from = 0
          to   = 0
        }
        model = jsonencode(merge(local.base_rule_model, {
          conditions = [
            {
              type      = "query"
              evaluator = { type = "gt", params = [4.5] }
              operator  = { type = "and" }
              reducer   = { type = "last", params = [] }
              query     = { params = ["A"] }
            }
          ]
          refId      = "B"
          type       = "classic_conditions"
          datasource = { type = "__expr__", uid = "-100" }
        }))
      }
    }
  }
}

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
  dashboard_uid    = grafana_dashboard.kubernetes.uid
  rules = [
    {
      name               = "Pod Restarts"
      summary            = "Pod restart events"
      description        = "Triggers an alert if a pod restarts."
      dashboard_panel_id = 0
      duration           = "0s"
      query              = <<EOT
        sum by (node, pod, container, service) (
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
