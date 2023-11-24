job "my-app" {
  group "app" {
    network {
      port "app" {
        to = 8000
      }
    }
    task "main" {
      leader = true
      driver = "docker"
      config {
        image = "my-app:v1"
        ports = ["app"]
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
