variable "org_id" {
  type    = number
  default = 1
}

variable "datasource_uid" {
  type = string
}

variable "folder_uid" {
  type    = string
  default = null
}

variable "name" {
  type = string
}

variable "interval_seconds" {
  type = number
}

variable "rules" {
  type = list(object({
    name        = string
    summary     = optional(string)
    description = optional(string)
    duration    = optional(string, "0s")
    dashboard = optional(object({
      uid      = string
      panel_id = number
    }))
    # Prometheus query
    query = string
    # Trigger condition
    condition = object({
      op      = string
      args    = list(any)
      reducer = optional(string, "last")
    })
    labels    = optional(map(string), {})
    is_paused = optional(bool, false)
  }))

  validation {
    condition = alltrue([
      for rule in var.rules :
      contains([
        "gt",            # greater than - one param
        "lt",            # less than - one param
        "outside_range", # two params
        "within_range",  # two params
        "no_value"       # no params
      ], rule.condition.op)
    ])
    error_message = "Invalid operation code"
  }

  validation {
    error_message = "Invalid reducer in rule operation."
    condition = alltrue([
      for rule in var.rules :
      contains([
        "avg",
        "min",
        "max",
        "sum",
        "count",
        "last",
        "median",
        "diff",
        "diff_abs",
        "percent_diff",
        "percent_diff_abs",
        "count_non_null",
      ], rule.condition.reducer)
    ])
  }
}

variable "extra_annotations" {
  type    = map(string)
  default = {}
}
