job "my-app" {
  meta {
    owner = "hr"
  }
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
        logging = true
      }

      config {
        image = "my-app:v1"
        ports = ["app"]
      }
    }
  }
}
