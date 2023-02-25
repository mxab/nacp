job "example" {
  datacenters = ["dc1"]

  meta {
    costcenter = "cccode-bar"

  }

  group "cache" {
    network {
      port "db" {
        to = 6379
      }
    }

    task "redis" {
      driver = "docker"


      config {
        image          = "redis:7"
        ports          = ["db"]
        auth_soft_fail = true
      }
      env {
        foo = "bar"
      }
      resources {
        cpu    = 500
        memory = 256
      }
    }
  }
}
