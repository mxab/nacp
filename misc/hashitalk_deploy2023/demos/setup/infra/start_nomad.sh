#!/bin/sh

IP_ADDRESS=$(ipconfig getifaddr en0)
VAULT_ADDRESS="http://${IP_ADDRESS}:8200"

echo $IP_ADDRESS > ./misc/hashitalk_deploy2023/demos/setup/infra/ip_address.txt
exec sudo nomad agent -dev \
  -config=./misc/hashitalk_deploy2023/demos/setup/infra/config \
  -network-interface='en0' \
  -vault-address="$VAULT_ADDRESS"
#  -network-interface='{{ GetDefaultInterfaces | attr "name" }}' \
