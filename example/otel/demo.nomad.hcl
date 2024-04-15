job "example" {
 group "demo" {
  task "app" {
   driver = "docker"
   meta { otel = "true" }
   config {
    image = "busybox:1.36.1"
     args  = ["/bin/sh", "-c", "env && sleep 1h"]
   }
  }
 }
}
