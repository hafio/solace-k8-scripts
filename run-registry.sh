#!/bin/bash

podman run -d --name podman-registry -p 443:443 -p 80:443 -p 5000:5000 -e REGISTRY_HTTP_ADDR="0.0.0.0:443" -v ../registry-data:/var/lib/registry -v "./server.crt:/secret/server.crt" -v "./server.key:/secret/server.key" docker.io/library/registry:latest
