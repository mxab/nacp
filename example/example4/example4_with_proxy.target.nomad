job "webapp" {

  datacenters = ["dc1"]

  type = "service"

  group "webapp" {
    meta {
        secure = "webapp"
    }
    network {
      port "http" {
        to = 8000
      }
      port "auth" {
        to = 4180
      }
    }


    task "webapp" {


      driver = "docker"

      config {

        image = "node:18-alpine"
        args  = ["/local/webapp.js"]
        ports = ["http"]

      }
      template {
        data        = file("webapp.js")
        destination = "local/webapp.js"
      }



      resources {
        memory = 256
      }

    }
    task "auth" {
      driver = "docker"
      config {
        image = "bitnami/oauth2-proxy:7.4.0"

        args = [
          "--config", "/local/config.cfg"
        ]
        ports = ["auth"]
      }
      template {
        data        = <<-EOF

          provider = "keycloak-oidc"
          client_id = "webapp"
          client_secret = "secret"
          redirect_url = "http://webapp.nomad.local/oauth2/callback"
          oidc_issuer_url = "http://keycloak.nomad.local/realms/demo"
          email_domains = ["*"]
          cookie_secret="TEktZxl3wbO9cL3mkm-DyMRvjhhJqxf7Xk8fcZQFq-U="
          http_address="http://0.0.0.0:4180"
          oidc_extra_audiences=["account"]
          cookie_secure=false
          {{ range nomadService "internal-webapp" }}
          upstreams = [
            "http://{{ .Address }}:{{ .Port }}"
          ]
          {{ end }}


          EOF
        destination = "${NOMAD_TASK_DIR}/config.cfg"
      }

    }


      service {

        name = "internal-webapp"
        provider = "nomad"
        port     = "http"
      }
      service {

        name = "webapp"
        provider = "nomad"
        port     = "auth"

      }
  }
}
