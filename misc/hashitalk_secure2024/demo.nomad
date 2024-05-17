job "demo" {
  datacenters = ["dc1"]
  group "demo" {
    task "demo" {
      driver = "docker"
      config {
        image = "localhost:5000/my-app:v1"
      }
    }
  }
}
