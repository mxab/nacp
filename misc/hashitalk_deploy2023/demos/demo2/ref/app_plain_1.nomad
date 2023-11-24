job "my-app" {
  group "app" {
    task "main" {

      config {
        image = "my-app:v1"
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
