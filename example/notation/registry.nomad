job "registry" {

  type = "service"

  group "registry" {

    network {
      port "registry" {
        static = 5001
        to     = 5000

      }
    }

    task "registry" {

      driver = "docker"

      config {
        image = "registry"
        ports = ["registry"]
      }
      env {
        REGISTRY_STORAGE_DELETE_ENABLED = "true"
      }
    }
  }
}
