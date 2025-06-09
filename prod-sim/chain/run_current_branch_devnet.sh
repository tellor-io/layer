#!/bin/bash

echo "copy default docker genesis to genesis/genesis.json"
cp ./exported_data/default_docker_genesis.json ./genesis/genesis.json

echo "build layerd binary for docker environment"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./bin/layerd ../../cmd/layerd

echo "set IS_FORK=false in docker-compose.yml"
sed -i '' 's/IS_FORK=true/IS_FORK=false/g' docker-compose.yml

echo "start docker containers"
docker compose up -d

echo "wait for docker containers to start"
sleep 10


echo "You can follow the logs of validator-0 using this command: docker logs -f validator-node-0"




