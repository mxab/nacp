job "postgres" {

    type = "service"

    datacenters = ["dc1"]

    group "pg" {

        network {
            port "db" {
                to = 5432
                static = 5432
            }
        }
        task "postgres" {
            driver = "docker"

            config {
                image = "postgres:15"
                ports = ["db"]

            }
            env {
                POSTGRES_PASSWORD = "secret"
            }
            resources {
                cpu = 500
                memory = 1024
            }
        }
        service {
            name = "postgres"
            port = "db"
            provider = "nomad"
            check {
                type = "tcp"
                interval = "10s"
                timeout = "2s"
            }
        }
    }
}
