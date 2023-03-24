job "app" {

  group "app" {

    task "app" {

      meta {
       # postgres = "native"
      }

      driver = "docker"

      config { # a very simple docker container
        image = "busybox:1.34.1"
        command = "sh"
        args = [
          "-c",
          "echo \"Environment:\"; env | sort; while true; do echo .; sleep 100; done"
        ]
      }
    }
  }
}
