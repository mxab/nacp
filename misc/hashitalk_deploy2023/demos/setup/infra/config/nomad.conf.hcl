bind_addr = "0.0.0.0"

vault {
    enabled = true
    token = "root"
}


plugin "docker" {
  config {
    gc {
      image = false
     // image_delay = "1h"
    }
  }
}
