job "infra" {

    group "vault" {

        network {

            port "vault" {
                static = 8200
            }
        }
        task "vault" {

            driver = "docker"

            config  {
                image = "hashicorp/vault:1.15"
                ports = ["vault"]

            }
            env {
                VAULT_DEV_ROOT_TOKEN_ID = "root"
            }
        }
    }
    group "postgres" {

        network {
            port "psql" {
                static = 5432
            }
        }
        task "postgres" {
            driver = "docker"
            config  {
                image = "postgres:15"
                ports = ["psql"]
            }
            env {
                POSTGRES_PASSWORD = "postgres"
            }
        }
        service {
            name = "postgres"
            port = "psql"
            provider = "nomad"
        }

    }

}
