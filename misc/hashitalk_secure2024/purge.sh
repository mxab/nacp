#!/bin/bash

nomad stop -purge registry

docker rmi -f localhost:5000/my-app:v1

./delete_test_data.sh
