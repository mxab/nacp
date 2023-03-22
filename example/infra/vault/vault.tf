terraform {
  required_providers {
    vault = {
      source = "hashicorp/vault"
      version = "3.14.0"
    }
  }
}

provider "vault" {
  address = "http://localhost:8200"
  token = "root"
}


resource "vault_mount" "db" {
  path = "postgres"
  type = "database"
}

resource "vault_database_secret_backend_connection" "postgres" {
  backend       = vault_mount.db.path
  name          = "postgres"
  allowed_roles = ["dev", "prod"]

  postgresql {
    connection_url = "postgres://postgres:secret@postgres.nomad.local:5432/postgres"
  }
}


resource "vault_database_secret_backend_role" "role" {
  backend             = vault_mount.db.path
  name                = "dev"
  db_name             = vault_database_secret_backend_connection.postgres.name
  creation_statements = ["CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}';"]
}
resource "vault_policy" "db_access" {
  name = "db-access"
  policy = <<-EOT
    path "postgres/creds/dev" {
        capabilities = ["read"]
    }

  EOT


}

# terraform {
#   required_providers {
#     postgresql = {
#       source = "cyrilgdn/postgresql"
#       version = "1.18.0"
#     }
#   }
# }
# provider "postgresql" {
#   host            = "localhost"
#   port            = 5432
#   database        = "postgres"
#   username        = "postgres"
#   password        = "secret"
#   sslmode         = "disable"
#   connect_timeout = 15
# }
