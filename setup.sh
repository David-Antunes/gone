#!/bin/bash

trap ctrl_c INT
function ctrl_c() {
  kill -9 $emu_pid
  kill -9 $pid
  exit 1
}

docker unpause $(docker ps -q)
docker kill $(docker ps -q --filter "network=gone_net")
docker kill gone-$(hostname)
docker kill proxy-$(hostname)
docker kill neo

docker system prune -f


docker run -d --name neo -p 7474:7474 -p 7687:7687 -e NEO4J_apoc_export_file_enabled=true -e NEO4J_apoc_import_file_enabled=true -e NEO4J_apoc_import_file_use__neo4j__config=true -e  NEO4J_PLUGINS=\[\"apoc\"\] -e NEO4J_AUTH=none neo4j

docker network create --driver=overlay --attachable --subnet=10.1.0.0/24 gone_net

cd gone-rtt

docker build -t gone-rtt .
cd ..

cd gone-proxy

docker build -t gone-proxy .

cd ..

docker build -t gone .

