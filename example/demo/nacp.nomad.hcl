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
                image = "nacp-local:v1"
                #image = "ghcr.io/mxab/nacp:v0.8.0"
                ports = ["nacp"]
                args  = ["-config", "/local/nacp.conf.hcl"]
            }

            service {
                name = "nacp"
                provider = "nomad"

                port = "nacp"

            }
            env {
                OTEL_EXPORTER_OTLP_ENDPOINT = "http://${attr.unique.network.ip-address}:4318"

                OTEL_EXPORTER_OTLP_PROTOCOL = "http/protobuf"

                OTEL_RESOURCE_ATTRIBUTES = "service.name=nacp,service.version=0.8.0,service.instance.id=${NOMAD_SHORT_ALLOC_ID}"
            }
            template {
                data = file("nacp.conf")
                destination = "local/nacp.conf.hcl"
            }
            template {
                data = file("../example1/validators/costcenter_meta.rego")
                destination = "local/validators/costcenter_meta.rego"
            }
            template {
                data = file("../example2/mutators/hello_world_meta.rego")
                destination = "local/mutators/hello_world_meta.rego"
            }
        }


    }
    group "monitoring" {
        count = 1

        network  {

            port "grafana" {
                static = 3000
            }
            port "otlp_http" {
                static = 4318
            }
        }

        task "lgtm" {
            driver = "docker"


            config {
                image = "grafana/otel-lgtm:0.11.4"
                ports = ["grafana", "otlp_http"]
            }

            service {

                provider = "nomad"
                name = "lgtm"

                port = "grafana"
            }

            resources {
                cpu    = 1024
                memory = 1024
            }
        }
    }
}
