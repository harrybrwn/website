variable "bucket" {
  type        = string
  description = "Name of the S3 bucket to store all files in."
}

variable "base_dir" {
  type        = string
  description = "Base directory of all helm build objects."
}

variable "charts_dir" {
  type        = string
  description = "Directory where all the helm chart source code is."
  default     = "charts"
}

variable "build_dir" {
  type        = string
  description = "The build output folder for all the packaged helm charts."
  default     = "dist"
}
