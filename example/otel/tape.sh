#!/bin/bash

rm otel-demo.mp4
rm demo.nomad.hcl

export NOMAD_ADDR='http://localhost:6464'
exec vhs otel-demo.tape
