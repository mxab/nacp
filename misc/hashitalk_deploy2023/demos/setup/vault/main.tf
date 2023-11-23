provider "vault" {

  address = "http://${local.local_ip}:8200"

  token = "root"
}


provider "postgresql" {
  host            = local.local_ip
  port            = 5432
  database        = "postgres"
  username        = "postgres"
  password        = "postgres"
  sslmode         = "disable"
}

locals {
  #local_ip = chomp(data.http.myip.response_body)
  local_ip = chomp(file("${path.module}/../infra/ip_address.txt"))

  jobs  = ["my-app"]

}

resource "postgresql_database" "db" {
  for_each = toset(local.jobs)

  name              =  replace(each.key, "-", "_")
  owner             = "postgres"
  template =  "template0"

}

resource "vault_database_secrets_mount" "db" {
  for_each = postgresql_database.db
  path = "db/${each.value.name}"
  postgresql {
    name              = "postgres-db-${each.value.name}"
    username          = "postgres"
    password          = "postgres"
    connection_url    = "postgresql://{{username}}:{{password}}@${local.local_ip}:5432/${each.value.name}"
    verify_connection = true
      allowed_roles = ["*"]

  }
}

resource "vault_database_secret_backend_role" "admin" {
  for_each = vault_database_secrets_mount.db
  name    = "admin"
  backend = each.value.path
  db_name = each.value.postgresql[0].name
  creation_statements = [
    "CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}';",
    "GRANT USAGE ON SCHEMA public TO \"{{name}}\";",
    "GRANT CREATE ON SCHEMA public TO \"{{name}}\";",
    "GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO \"{{name}}\";",
    "GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO \"{{name}}\";",
    "GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO \"{{name}}\";",
    "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO \"{{name}}\";"

  ]
    default_ttl = 3600
    max_ttl     = 3600 * 24
}


resource "vault_policy" "app" {
  for_each = toset(local.jobs)

  name = "${each.value}-db-access"

  policy = <<EOT
path "db/${replace(each.key, "-", "_")}/creds/admin" {
  capabilities = ["read"]
}
EOT
}
