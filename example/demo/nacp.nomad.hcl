job "nacp" {


    group "nacp" {
        count = 1

        network  {

            port "nacp" {
                static = 6464

            }
        }

        task "nacp" {
            driver = "docker"

            config {
                image = "nacp:latest"
                ports = ["http"]
            }

            service {
                name = "nacp"
                tags = ["urlprefix-/"]
                port = "http"
            }
        }
    }
    group "monitoring" {
        count = 1

        network  {

            port "grafana" {
                static = 3000
            }
        }

        task "lgtm" {
            driver = "docker"


            config {
                image = "grafana/otel-lgtm"
                ports = ["http"]
            }

            service {
                name = "lgtm"
                tags = ["urlprefix-/"]
                port = "http"
            }
        }
    }
}
