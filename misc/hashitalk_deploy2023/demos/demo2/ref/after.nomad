job "my-app" {
  group "app" {
    network {
      port "app" {
        to = 8000
      }
    }
    task "main" {
      driver = "docker"

      meta {
        postgres = "native"
      }

      config {
        image = "my-fastapi-app:v1"
        ports = ["app"]
      }
      vault {
        policies = ["my-app-db-access"]
      }
      template {
        data        = <<-EOH
        PGDATABASE=my_app
        {{ with secret "db/my_app/creds/admin" }}
        PGPASSWORD={{ .Data.password }}
        PGUSER={{ .Data.username }}
        {{ end }}
        {{ range nomadService "postgres" }}
        PGHOST={{ .Address }}
        PGPORT={{ .Port }}
        {{ end}}
        EOH
        destination = "secrets/postgres.env"
        env         = true
      }
    }
  }
}
