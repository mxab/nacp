job "demo" {

    group "demo" {
        count = 1

        task "demo" {
            driver = "docker"

            config {
                image = "localhost:5001/net-monitor:v1"
            }
        }
    }
}
