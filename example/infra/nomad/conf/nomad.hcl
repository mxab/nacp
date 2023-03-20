

bind_addr = "0.0.0.0" # the default
server {
  enabled          = true
}

client {
  enabled       = true
}

vault {
  enabled = true
  address = "http://localhost:8200"
  token = "root"
}
