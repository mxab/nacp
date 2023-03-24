job "webapp" {

  group "webapp" {

    meta {
     # secure = "webapp"
    }
    network {
      port "http" {
        to = 8000
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

    service {
        name = "webapp"
        provider = "nomad"
        port     = "http"

    }
  }
}
