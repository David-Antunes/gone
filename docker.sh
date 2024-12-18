#!/bin/bash

trap ctrl_c INT
function ctrl_c() {
  docker kill rtt-$(hostname)
  docker kill proxy-$(hostname)
  docker kill gone-$(hostname)
  exit 1
}

if [ "$#" -ne 1 ]; then
    echo "Illegal number of parameters"
    exit 1
fi

LOCAL_IP=$(ip -4 -o addr show $1 | awk '{print $4}' | cut -d "/" -f 1 )

if [[ ! $LOCAL_IP =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Invalid interface name"
  exit 1
fi


docker unpause $(docker ps -q --filter "network=gone_net")
docker kill $(docker ps -q --filter "network=gone_net")

docker kill proxy-$(hostname)
docker kill gone-$(hostname)
docker kill rtt-$(hostname)
docker kill agent-$(hostname)

docker rm rtt-$(hostname)
docker rm proxy-$(hostname)
docker rm gone-$(hostname)
docker rm agent-$(hostname)


docker run -d --name rtt-$(hostname) --network gone_net gone-rtt

NETWORK_ID=$(docker network list -f "name=gone_net" --format "{{.ID}}")

sleep 1

docker run -d --privileged --ulimit memlock=65535 --network none --name proxy-$(hostname) -v /var/run/docker/netns:/var/run/docker/netns -v /tmp:/tmp -e NETWORK=$NETWORK_ID -e NUM_TESTS=1000 gone-proxy

docker run -d --privileged --name gone-$(hostname) -p 3000:3000 -p 3001:3001 -v /tmp:/tmp -v /var/run/docker.sock:/var/run/docker.sock -v /proc:/proc -e GRAPHDB=$LOCAL_IP -e SERVER_IP=$LOCAL_IP -e PRIMARY_SERVER_IP=$LOCAL_IP -e PRIMARY=1 -e ID=primary -e NUM_TESTS=1000 gone

docker run -d --name agent-$(hostname) -p 3300:3300 -v /tmp:/tmp -v /var/run/docker.sock:/var/run/docker.sock -e GONE_ID=gone-$(hostname) -e GONE_PROXY_ID=proxy-$(hostname) -e GONE_RTT_ID=rtt-$(hostname) -e SERVER_IP=$LOCAL_IP -e PRIMARY_IP=$LOCAL_IP gone-agent

