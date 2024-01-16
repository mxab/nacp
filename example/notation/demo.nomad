variable "image" {
  type = string
}
job "demo" {

  group "demo" {
    count = 1

    task "demo" {
      driver = "docker"

      config {
        image = var.image
      }
    }
  }
}
