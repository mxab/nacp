terraform {
  required_providers {
    keycloak = {
      source  = "mrparkers/keycloak"
      version = "4.2.0"
    }
  }
}

provider "keycloak" {

  url   = "http://keycloak.nomad.local"
  username  = "admin"
  password  = "admin"
  realm     = "master"
  client_id = "admin-cli"
}

resource "keycloak_realm" "realm" {
  realm = "demo"
}
resource "keycloak_openid_client" "openid_client" {
  realm_id  = keycloak_realm.realm.id
  client_id = "webapp"

  name    = "webapp"
  enabled = true

  access_type = "CONFIDENTIAL"
  implicit_flow_enabled = true
  standard_flow_enabled = true
  root_url = "http://webapp.nomad.local"
  base_url = "/"
  valid_redirect_uris = [
    "http://webapp.nomad.local/*",
  ]

  client_secret = "secret"


}

resource "keycloak_user" "alice" {
  realm_id = keycloak_realm.realm.id
  username = "alice"
  email    = "alice@example.com"
    enabled  = true

  first_name = "Alice"
  last_name  = "Armstrong"

  initial_password {
    value     = "alice"

  }
  email_verified = true
}
