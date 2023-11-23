terraform {
  required_providers {
    vault = {
      source = "hashicorp/vault"
      version = "3.23.0"
    }
    http = {
      source = "hashicorp/http"
      version = "3.4.0"
    }
    postgresql = {
      source = "cyrilgdn/postgresql"
      version = "1.21.1-beta.1"
    }
  }
}
