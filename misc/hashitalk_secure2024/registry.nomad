job "registry" {

  type = "service"

  group "registry" {

    network {
      port "registry" {
        static = 5000
        to     = 5000
      }
    }

    task "registry" {

      driver = "docker"

      config {
        image = "registry:2"
        ports = ["registry"]
      }
      env {
        REGISTRY_STORAGE_DELETE_ENABLED = "true"
      }
    }
  }
}
