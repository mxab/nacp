client {
    preferred_address_family = "ipv4"
    network_interface = "en0"
}
bind_addr = "0.0.0.0" # the default
plugin "docker" {
  config {
    gc {
      # Disable automatic image garbage collection
      image = false

      # Optional: Adjust the delay time before a container itself is cleaned up
      image_delay = "3m"
    }
  }
}
