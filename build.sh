#!/bin/bash

docker build -t gone-rtt gone-rtt/
docker build -t gone-proxy gone-proxy/
docker build -t gone-agent gone-agent/
docker build -t gone .
