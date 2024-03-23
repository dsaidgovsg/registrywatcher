#!/bin/sh

docker compose -f env/test/docker-compose.yml up -d
docker compose -f env/test/docker-compose.yml exec testenv sh
docker compose -f env/test/docker-compose.yml down 