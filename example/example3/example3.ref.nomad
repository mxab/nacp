# reference on how the job should look like
job "app" {

  group "app" {

    task "app" {
      driver = "docker"


      config {
        image = "busybox:latest"
        command = "sh"
        args = ["-c", "while true; do echo \"hello @ $(date)\"; sleep 5; done"]
      }
      vault {
        policies = ["db-access"]
      }
      template {
        data = <<-EOH
          {{ range nomadService "postgres" }}
            PGHOSTADDR={{ .Address }}
            PGPORT={{ .Port }}
          {{ end }}
          PGDATABASE=postgres
          {{ with secret "postgres/creds/dev" }}
            PGUSER={{ .Data.username }}
            PGPASSWORD={{ .Data.password }}
          {{ end }}

        EOH
        env = true
        destination = "${NOMAD_SECRETS_DIR}/postgres.env"
      }
    }
  }
}
