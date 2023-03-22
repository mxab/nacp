job "traefik" {
    datacenters = ["dc1"]
    type = "service"


    group "traefik" {
        network {
                port "http" {

                    static = 80
                }
                 port  "admin"{
                    static = 8080
      }
            }

        task "traefik" {
            driver = "docker"



            config {
                image = "traefik:v2.9.9"

                ports = ["admin", "http"]
                args = [
                    "--log.level=DEBUG",
                "--api.dashboard=true",
                "--api.insecure=true", ### For Test only, please do not use that in production
                "--entrypoints.web.address=:${NOMAD_PORT_http}",
                "--entrypoints.traefik.address=:${NOMAD_PORT_admin}",
                "--providers.nomad=true",
                "--providers.nomad.endpoint.address=http://host.docker.internal:4646", ### IP to your nomad server
                "--providers.nomad.defaultRule=Host(`{{ .Name }}.nomad.local`)"

                ]
            }


            resources {

                memory = 256

            }
        }
        service {
            name = "traefik-http"
            provider = "nomad"
            port = "http"
        }

    }
}
