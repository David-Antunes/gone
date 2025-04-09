# Go Network Emulator

Go Network Emulator (GONE) is a scalable network emulator that provides the user to create their own network topology between docker containers, appropriate for testing applications.

GONE leverages Docker to execute unmodified application code and resorts to using technologies such as eBPF and XDP to intercept the network traffic between the affected containers to emulate a network topology.

## Dependencies

To sucessfully run the network emulator, it is required some dependencies.

1. Minimium Linux Kernel version of 5.15
2. Docker engine installed

## Starting the application

### Setup

To launch the network emulator, you need to clone the repository.

```bash
git clone https://github.com/David-Antunes/gone
cd gone
git submodule update --init
```

Next, you need to start Docker Swarm.

```bash
docker swarm init
```

And create the default network used by GONE to configure the emulation:

```bash
docker network create --driver=overlay --attachable --subnet=10.1.0.0/24 gone_net
```

You can change the subnet to allow more containers in the network.

To build the necessary images you need to execute the following:

```bash
./setup.sh
```

This script starts the graph database and build the containers for GONE-RTT, GONE-Proxy, GONE, and GONE-Agent.

### Deployment

To deploy the network emulator the user can utilize the `start.sh` script to deploy the network emulator.

```bash
./start.sh
```
```
Usage: ./start.sh -i <ifname> -N <name> -n <network-name> (-P | -s <source_ip>) [-A] [-C]
  -i <ifname>       : Required network interface name
  -N <name>         : GONE instance ID
  -n <network-name> : Docker network created
  -P                : Start GONE as Leader (mutually exclusive with -s)
  -s <source_ip>    : Leader IP address (mutually exclusive with -P)
  -A                : Starts GONE-Agent (optional)
  -C                : Clears all containers (Ignores all other flags)
```

This script requires the user to provide a name of the network interface connected to the internet. Typically this interface is called eth0 or eno1.

To deploy locally the user can use the following flags:

```bash
./start.sh -i eth0 -N $(hostname) -P -n gone_net
```
This will start GONE-RTT, GONE-Proxy, and GONE as a Leader. This will create a local network emulator that you can use.

The "-i" is the network interface that is connected to the network.
The "-N" flag receives an identifier for the instance.
The "-P" flag starts GONE as a Leader.

To clear the network topology you can just re-execute the same command, as this will redeploy GONE. However, don't forget to run `docker system prune` to remove the created containers from the previous experiment (you can add the flag "--rm" to the docker command so that when the exit they get automatically removed by docker), as this will result in failing to add the container back to the emulation since the container already exists.

### Distributed GONE

To execute GONE in multiple machines, you have to join the participating machines into the Docker Swarm, and build the docker images.

For every machine that is going to participate as a Follower, you can use the `build.sh` script to only build the docker images.

In the machine participating as the leader, you have to create the relevant docker network, if you have not created previously.

```bash
docker network create --driver=overlay --attachable --subnet=10.1.0.0/24 gone_net
```

In the leader machine, run the `setup.sh` script.

```bash
./setup.sh
```
This builds the docker images in the leader machines and starts the graph database, which every follower instance will use.

Considering deploying in two machines, one leader and one follower, with the IP addresses of 192.168.1.2 and 192.168.1.3 respectively.

In the leader machine you execute `start.sh` with the following values:

```bash
./start.sh -i eth0 -N leader-1 -P -A -n gone_net
```
In this scenario, we identified the leader instance with the id of "leader-1". This identifier is important since it is used to create network components in the desired instance.

In this command, we use the "-A" flag to also deploy GONE-Agent, to manage the GONE instances in all participating machines, allowing the restarting or shutting down of the network emulator in all machines.


To add a follower to the network emulator, the follower machine must run the following command:

```bash
./start.sh -i eth0 -N follower-1 -s 192.168.1.2 -A -n gone_net
```

In this scenario, the user must use the "-s" flag, providing the IP address of the leader so the follower can connect to the leader and participate in the emulation.

To interact with the network emulator, you can use [GONE-CLI](https://github.com/David-Antunes/gone-cli)

### Constraints

To add an application to the emulation, the docker command provided to GONE must contain the following flags:

1. "-d" to run in detached mode. At this moment GONE does not allow to executing a docker container interactively.
2. "--network gone_net" needs to be present on the command so that the container is executed inside of the correct namespace.

We have not tested every flag available in Docker to check whether or not GONE can execute the docker command. In case of failing to deploy, add a new flag to the docker command to check whether or not GONE can execute, until finding the breaking flag.

### Warning

Be careful of the docker command you supply to GONE, since for example when mounting a directory inside of the docker container, the mounted directory will be in the host machine, so be careful regarding the mounting operation as this could lead to loss of data.

## Examples

We provide some examples of how to configure a network using GONE and GONE-CLI.

### 1 Bridge and 2 Nodes

To configure a simple network containing a bridge and 2 nodes, a client and a server the user can execute the following:

* To configure GONE:
```bash
./start.sh -i eth0 -N $(hostname) -P -n gone_net
```

* To create the network topology:
```bash
## Creates the iperf server
gone-cli node -- docker run --rm -d --network gone_net --name server nicolaka/netshoot iperf -s

## Creates the iperf client
gone-cli node -- docker run --rm -d --network gone_net --name client nicolaka/netshoot iperf -c server

# Create a bridge
gone-cli bridge bridge1 

## Connects server and client to the same bridge with a default bandwidth of 10Mbits
gone-cli connect -n server bridge1

gone-cli connect -n client bridge1

## Starts the server and then the client
gone-cli unpause server
gone-cli unpause client
```

To stop the entire system you can execute:
```bash
./start.sh -C -N $(hostname) -n gone_net
```

If you plan on deploying other networks you can restart by executing the previous command:
```bash
./start.sh -i eth0 -N $(hostname) -P -n gone_net
```


### 4 Nodes, 4 Bridges, and 2 Routers

To configure a network containing a network with 4 nodes, 2 clients and 2 servers, connected to their own routers and bridges, and with custom links, the user can execute the following commands:

```bash
## Create the iperf servers
gone-cli node -- docker run -d --network gone_net --name server-left nicolaka/netshoot iperf -s
gone-cli node -- docker run -d --network gone_net --name server-right nicolaka/netshoot iperf -s

## Creates the iperf clients
gone-cli node -- docker run -d --network gone_net --name client-left nicolaka/netshoot iperf -c server-right
gone-cli node -- docker run -d --network gone_net --name client-right nicolaka/netshoot iperf -c server-left

# Create a bridges
gone-cli bridge bridge-client-left
gone-cli bridge bridge-server-left

gone-cli bridge bridge-client-right
gone-cli bridge bridge-server-right

## Create routers
gone-cli router router-left
gone-cli router router-right

## Connect servers and clients with 100Mbits of bandwidth

gone-cli connect -w 100M -n client-left bridge-client-left
gone-cli connect -w 100M -n server-left bridge-server-left

gone-cli connect -w 100M -n client-right bridge-client-right
gone-cli connect -w 100M -n server-right bridge-server-right

## Connect the bridges to the respective routers

gone-cli connect -w 100M -b bridge-client-left router-left
gone-cli connect -w 100M -b bridge-server-left router-left

gone-cli connect -w 100M -b bridge-client-right router-right
gone-cli connect -w 100M -b bridge-server-right router-right

## Connect the two routers with 10Mbits and 10 millisecond latency (5 ms for each direction)

gone-cli connect -l 10 -w 10M -r router-left router-right

## Propagate routing rules
gone-cli propagate router-left

## Unpause servers
gone-cli unpause server-left
gone-cli unpause server-right

## Unpause all nodes
gone-cli unpause -a
```

## Interacting with GONE

GONE provides the following API endpoints:

```
"/addNode"
"/addBridge"
"/addRouter"

"/connectNodeToBridge"
"/connectBridgeToRouter"
"/connectRouterToRouter"

"/inspectNode"
"/inspectBridge"
"/inspectRouter"

"/removeNode"
"/removeBridge"
"/removeRouter"

"/disconnectNode"
"/disconnectBridge"
"/disconnectRouters"

"/forget"
"/propagate"

"/sniffNode"
"/sniffBridge"
"/sniffRouters"
"/stopSniff"
"/listSniffers"

"/interceptNode"
"/interceptBridge"
"/interceptRouter"
"/stopIntercept"
"/listIntercepts"

"/disruptNode"
"/stopDisruptNode"
"/disruptBridge"
"/stopDisruptBridge"
"/disruptRouters"
"/stopDisruptRouters"

"/startBridge"
"/stopBridge"

"/startRouter"
"/stopRouter"

"/pause"
"/unpause"
```

To use this endpoints, you can use the [GONE-CLI](https://github.com/David-Antunes/gone-cli). This repository contains a simple explanation of every operation available that you can use to manage the network emulator.

## Advanced

The user can change the behavior of the network emulator by altering the available environment variables. To do that, the user must change the `start.sh` script to change or include the environment variables it wishes to use.

### GONE

```
GRAPHDB=localhost
GRAPHDB_PASSWORD=
GRAPHDB_USER=
GRAPH_COST=10000000
ID=gone
NETWORK_NAMESPACE=gone_net
NUM_TESTS=1000
PRIMARY=1
PRIMARY_ROUTE_PORT=3001
PRIMARY_SERVER_IP=192.168.1.1
PRIMARY_SERVER_PORT=3000
PROXY_RTT_SOCKET=/tmp/proxy-rtt.sock
PROXY_RTT_UPDATE_MS=0
PROXY_SERVER=/tmp/proxy-server.sock
SERVER_IP=192.168.1.1
SERVER_PORT=3000
SERVER_ROUTE_PORT=3001
TIMEOUT_REMOTE_RTT_MS=0
```

### GONE-Proxy

```
NUM_TESTS=1000
PROXY_RTT_SOCKET=/tmp/proxy-rtt.sock
PROXY_SERVER=/tmp/proxy-server.sock
TIMEOUT=0
```

### GONE-Agent

```
GONE_ID=gone
GONE_IMAGE=gone
GONE_PROXY_ID=proxy-brain
GONE_PROXY_IMAGE=gone-proxy
GONE_RTT_ID=rtt-brain
GONE_RTT_IMAGE=gone-rtt
NETWORK_NAMESPACE=gone_net
PORT=3300
PRIMARY_IP=192.168.1.1
SERVER_IP=192.168.1.1
```

## Related Projects

* [GONE-Proxy](https://github.com/David-Antunes/gone-proxy) Program that intercepts network traffic and sends it to the emulator.
* [GONE-RTT](https://github.com/David-Antunes/gone-rtt) Program to measure RTT between an application and the emulator.
* [GONE-CLI](https://github.com/David-Antunes/gone-cli) CLI tool to interact with the network emulator.
* [GONE-Sniffer](https://github.com/David-Antunes/gone-sniffer) Example showing the usage of tcpdump in a particular link in the emulation.
* [GONE-Intercept](https://github.com/David-Antunes/gone-intercept) Example that intercepts network traffic and applies delay to a particular container.
* [GONE-Broadcast](https://github.com/David-Antunes/gone-broadcast) Example showing the usage of broadcast in the network emulation.
