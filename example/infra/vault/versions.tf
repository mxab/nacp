terraform {
  required_providers {
    vault = {
      source = "hashicorp/vault"
      version = "3.14.0"
    }
    postgresql = {
      source = "cyrilgdn/postgresql"
      version = "1.21.0"
    }
  }
}
