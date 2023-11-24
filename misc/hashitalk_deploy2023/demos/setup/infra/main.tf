# Register a job
provider "nomad" {

  address = "http://localhost:4646"
}
resource "nomad_job" "infastructure" {
  jobspec = file("${path.module}/infra.nomad")
}

variable "grafana_loki_username" {
  type = string
}
variable "grafana_loki_api_token" {
  type = string
  sensitive = true
}
variable "grafana_loki_endpoint" {
  type = string
}
resource "nomad_variable" "grafana" {

  path = "grafana"
  items = {
    loki_username = var.grafana_loki_username
    loki_api_token = var.grafana_loki_api_token
    loki_endpoint = var.grafana_loki_endpoint
  }
}

