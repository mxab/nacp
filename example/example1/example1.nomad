job "app" {

  meta {
    #costcenter = "cccode-backoffice"
  }
  group "app" {

    task "app" {
      driver = "docker"

      config { # a very simple docker container
        image = "busybox:1.34.1"
        command = "sh"
        args = [
          "-c",
          "while true; do echo \"hello @ $(date)\"; sleep 5; done"
        ]
      }
    }
  }
}
