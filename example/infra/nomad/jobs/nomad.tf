terraform {
  required_providers {
    nomad = {
      source = "hashicorp/nomad"
      version = "1.4.19"
    }

  }
}
provider "nomad" {
  address = "http://localhost:4646"
}
resource "nomad_job" "pg" {
  jobspec = file("${path.module}/pg.nomad")
}
