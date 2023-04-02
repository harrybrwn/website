locals {
  base_rule_model = {
    hide          = false
    intervalMs    = 1000
    maxDataPoints = 43200
  }
}

resource "grafana_rule_group" "alerts" {
  org_id           = var.org_id
  name             = var.name
  folder_uid       = var.folder_uid
  interval_seconds = var.interval_seconds

  dynamic "rule" {
    for_each = var.rules
    iterator = i
    content {
      name      = i.value.name
      for       = i.value.duration
      is_paused = i.value.is_paused
      annotations = merge({
        summary          = i.value.summary == null ? i.value.name : i.value.summary
        description      = i.value.description == null ? "" : i.value.description
        __dashboardUid__ = var.dashboard_uid
        __panelId__      = "${i.value.dashboard_panel_id}"
      }, var.extra_annotations)
      labels    = i.value.labels
      condition = "B"

      data {
        ref_id         = "A"
        datasource_uid = var.datasource_uid
        relative_time_range {
          from = 600
          to   = 0
        }
        model = jsonencode(merge(local.base_rule_model, {
          expr  = i.value.query
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
              evaluator = { type = i.value.condition.op, params = i.value.condition.args }
              operator  = { type = "and" }
              reducer   = { type = i.value.condition.reducer, params = [] }
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
