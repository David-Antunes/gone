#!/bin/bash

# Initialize variables
INTERFACE=""
SOURCE_IP=""
NAME=""
NETWORK=""
P_FLAG=false
A_FLAG=false  # Boolean flag
C_FLAG=false  # Boolean flag

# Function to display usage
usage() {
    echo "Usage: $0 -i <ifname> -N <name> -n <network-name> (-P | -s <source_ip>) [-A] [-C]"
    echo "  -i <ifname>       : Required network interface name"
    echo "  -N <name>         : GONE instance ID"
    echo "  -n <network-name> : Docker network created"
    echo "  -P                : Start GONE as Leader (mutually exclusive with -s)"
    echo "  -s <source_ip>    : Leader IP address (mutually exclusive with -P)"
    echo "  -A                : Starts GONE-Agent (optional)"
    echo "  -C                : Clears all containers (Ignores all other flags)"
    exit 1
}

# Function to validate IPv4 addresses
is_valid_ipv4() {
    local ip=$1
    local regex='^([0-9]{1,3}\.){3}[0-9]{1,3}$'

    if [[ $ip =~ $regex ]]; then
        IFS='.' read -r -a octets <<< "$ip"
        for octet in "${octets[@]}"; do
            if (( octet < 0 || octet > 255 )); then
                return 1
            fi
        done
        return 0
    else
        return 1
    fi
}

# Parse command-line arguments
while getopts "i:n:N:Ps:AC" opt; do
    case "$opt" in
        i) INTERFACE="$OPTARG" ;;
        N) NAME="$OPTARG" ;;
        n) NETWORK="$OPTARG" ;;
        P) P_FLAG=true ;;
        s) SOURCE_IP="$OPTARG" ;;
        A) A_FLAG=true ;;
        C) C_FLAG=true ;;
        *) usage ;;
    esac
done

if [[ -z "$NETWORK" ]]; then
    echo "Error: -n (network name) is required."
    usage
fi

if [[ -z "$NAME" ]]; then
    echo "Error: -N (name) is required."
    usage
fi

if $C_FLAG; then
  echo "Clearing all containers"
  docker unpause $(docker ps -q --filter "network=$NETWORK")
  docker kill $(docker ps -q --filter "network=$NETWORK")

  docker kill proxy-$NAME
  docker kill gone-$NAME
  docker kill rtt-$NAME
  docker kill agent-$NAME
  docker kill neo

  docker rm rtt-$NAME
  docker rm proxy-$NAME
  docker rm gone-$NAME
  docker rm agent-$NAME
  docker rm neo

  exit 0
fi

# Validate required parameters
if [[ -z "$INTERFACE" ]]; then
    echo "Error: -i (interface) is required."
    usage
fi

if [[ -z "$NETWORK" ]]; then
    echo "Error: -n (network name) is required."
    usage
fi

# Ensure -P and -s are mutually exclusive
if $P_FLAG && [[ -n "$SOURCE_IP" ]]; then
    echo "Error: -P and -s cannot be used together."
    usage
fi

# Ensure at least one of -P or -s is provided
if ! $P_FLAG && [[ -z "$SOURCE_IP" ]]; then
    echo "Error: Either -P or -s must be specified."
    usage
fi

# Validate IP address if -s is provided
if [[ -n "$SOURCE_IP" ]] && ! is_valid_ipv4 "$SOURCE_IP"; then
    echo "Error: Invalid source IP address: $SOURCE_IP"
    exit 1
fi

# Display parsed values
echo "Interface: $INTERFACE"
echo "Name: $NAME"
echo "P Flag: $P_FLAG"
echo "A Flag: $A_FLAG"
#echo "C Flag: $C_FLAG"
if [[ -n "$SOURCE_IP" ]]; then
    echo "Source IP: $SOURCE_IP"
fi

# Get local IP address from provided network interface name
LOCAL_IP=$(ip -4 -o addr show $INTERFACE | awk '{print $4}' | cut -d "/" -f 1 )

if [[ ! $LOCAL_IP =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]]; then
  echo "Invalid interface name"
  exit 1
fi

#docker system prune -f

#docker network create --driver=overlay --attachable --subnet=10.1.0.0/24 gone_net

# Clear previous execution
docker unpause $(docker ps -q --filter "network=$NETWORK")
docker kill $(docker ps -q --filter "network=$NETWORK")

docker kill proxy-$NAME
docker kill gone-$NAME
docker kill rtt-$NAME
docker kill agent-$NAME
docker rm rtt-$NAME
docker rm proxy-$NAME
docker rm gone-$NAME
docker rm agent-$NAME

# Start GONE-RTT and obtain network id
docker run -d --name rtt-$NAME --network $NETWORK gone-rtt

NETWORK_ID=$(docker network list -f "name=$NETWORK" --format "{{.ID}}")

sleep 1

# Start GONE-Proxy
docker run -d --privileged --ulimit memlock=65535 --network none --name proxy-$NAME -v /var/run/docker/netns:/var/run/docker/netns -v /tmp:/tmp       \
  -e NETWORK=$NETWORK_ID    \
  -e NUM_TESTS=1000         \
  gone-proxy


# Example actions (modify as needed)
if $P_FLAG; then
    echo "Starting GONE as Leader."
    
    docker run -d --privileged --name gone-$NAME -p 3000:3000 -p 3001:3001 -v /tmp:/tmp -v /var/run/docker.sock:/var/run/docker.sock -v /proc:/proc                 \
      -e GRAPHDB=$LOCAL_IP                \
      -e GRAPHDB_PASSWORD=""              \
      -e GRAPHDB_USER=""                  \
      -e NETWORK_NAMESPACE=$NETWORK       \
      -e NUM_TESTS=1000                   \
      -e PRIMARY=1                        \
      -e PRIMARY_SERVER_IP=$LOCAL_IP      \
      -e PRIMARY_SERVER_PORT=3000         \
      -e SERVER_PORT=3000                 \
      -e SERVER_ROUTE_PORT=3001           \
      -e TIMEOUT_REMOTE_RTT_MS=0          \
      -e PROXY_RTT_UPDATE_MS=0            \
      -e SERVER_IP=$LOCAL_IP              \
      -e ID=$NAME                         \
      gone

elif [[ -n "$SOURCE_IP" ]]; then
    echo "Starting GONE Follower using Leader IP: $SOURCE_IP"
    
    docker run -d --privileged --name gone-$NAME -p 3000:3000 -p 3001:3001 -v /tmp:/tmp -v /var/run/docker.sock:/var/run/docker.sock -v /proc:/proc                 \
      -e GRAPHDB=$SOURCE_IP               \
      -e GRAPHDB_PASSWORD=""              \
      -e GRAPHDB_USER=""                  \
      -e NETWORK_NAMESPACE=$NETWORK       \
      -e NUM_TESTS=1000                   \
      -e PRIMARY=0                        \
      -e PRIMARY_SERVER_IP=$SOURCE_IP     \
      -e PRIMARY_SERVER_PORT=3000         \
      -e SERVER_PORT=3000                 \
      -e SERVER_ROUTE_PORT=3001           \
      -e TIMEOUT_REMOTE_RTT_MS=0          \
      -e PROXY_RTT_UPDATE_MS=0            \
      -e SERVER_IP=$LOCAL_IP              \
      -e ID=$NAME                         \
      gone
fi

if $A_FLAG; then
    echo "Starting GONE-Agent."
    
    if $P_FLAG; then
      docker run -d --name agent-$NAME -p 3300:3300 -v /tmp:/tmp -v /var/run/docker.sock:/var/run/docker.sock \
        -e GONE_ID=gone-$NAME             \
        -e GONE_IMAGE=gone                \
        -e GONE_PROXY_ID=proxy-$NAME      \
        -e GONE_PROXY_IMAGE=gone-proxy    \
        -e GONE_RTT_ID=rtt-$NAME          \
        -e GONE_RTT_IMAGE=gone-rtt        \
        -e PORT=3300                      \
        -e PRIMARY_IP=$LOCAL_IP           \
        -e SERVER_IP=$LOCAL_IP            \
        gone-agent
  else
      echo "Starting GONE-Agent."
      
      docker run -d --name agent-$NAME -p 3300:3300 -v /tmp:/tmp -v /var/run/docker.sock:/var/run/docker.sock \
        -e GONE_ID=gone-$NAME             \
        -e GONE_IMAGE=gone                \
        -e GONE_PROXY_ID=proxy-$NAME      \
        -e GONE_PROXY_IMAGE=gone-proxy    \
        -e GONE_RTT_ID=rtt-$NAME          \
        -e GONE_RTT_IMAGE=gone-rtt        \
        -e PORT=3300                      \
        -e PRIMARY_IP=$SOURCE_IP          \
        -e SERVER_IP=$LOCAL_IP            \
        gone-agent

    fi
fi

exit 0
