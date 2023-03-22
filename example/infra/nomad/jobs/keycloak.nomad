job "keycloak" {

    datacenters = ["dc1"]
    type        = "service"

    group "keycloak" {
        count = 1

        network {
            port "http" {
                to = 8080
            }
        }
        task "keycloak" {
            driver = "docker"


            config {
                image = "quay.io/keycloak/keycloak:21.0.1"

                ports = ["http"]
                command = "start-dev"
            }
            env {
                KEYCLOAK_ADMIN = "admin"
                KEYCLOAK_ADMIN_PASSWORD = "admin"
            }
            resources {

                memory = 2048

            }
        }
        service {
            name = "keycloak"
            port = "http"
            provider = "nomad"
            tags = [
                "traefik.enable=true",

            ]
            check {
                type     = "http"
                path     = "/"
                interval = "10s"
                timeout  = "2s"
            }
        }
    }
}
