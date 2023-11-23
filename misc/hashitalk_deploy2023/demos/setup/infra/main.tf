# Register a job
provider "nomad" {

  address = "http://localhost:4646"
}
resource "nomad_job" "infastructure" {
  jobspec = file("${path.module}/infra.nomad")
}

variable "grafana_api_key" {
  type = string

}
resource "nomad_variable" "grafana_api_key" {

  #

  path = "grafana"
  items = {
    api_key = var.grafana_api_key
  }
}
