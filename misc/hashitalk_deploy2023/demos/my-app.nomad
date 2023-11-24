job "my-app" {
  group "app" {
    network {
      port "app" {
        to = 8000
      }
    }
    task "main" {
      driver = "docker"
      
      config {
        image = "my-app:v1"
        ports = ["app"]
      }
    }
  }
}
