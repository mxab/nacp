job "my-app" {
  meta {
    owner = "hr"
  }
  group "app" {
    network {
      port "app" {
        to = 8000
      }
      port "agent" {
        to = 12345
      }
    }
    task "main" {
      leader = true
      driver = "docker"

      meta {
        postgres = "native"
        logging = true
      }

      config {
        image = "my-app:v1"
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
    task "grafanaagent" {
      driver = "docker"
      lifecycle {
        hook = "prestart"
        sidecar = true
      }
      config {
        image = "grafana-agent-sidecar:v1"
        ports = ["agent"]
      }
      template {
        env = true
        data = <<-EOH
        {{ with nomadVar "grafana" }}
        LOKI_URL={{ .loki_endpoint }}
        LOKI_USERNAME={{ .loki_username }}
        LOKI_API_TOKEN={{ .loki_api_token }}
        {{ end }}
        TASK_TO_LOG=main
        EOH
        destination = "secrets/grafanaagent.env"
      }
    }
  }
}
