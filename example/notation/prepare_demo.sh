#!/bin/bash
rm notation-demo.mp4

nomad stop -purge registry
nomad stop -purge demo
./delete_test_certs.sh
notation cert generate-test --default "wabbit-networks.io"
nomad run registry.nomad
docker rmi --no-prune localhost:5001/net-monitor:v1

vhs notation-demo.tape
