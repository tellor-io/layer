# local devnet

## dependency

- docker
- docker-compose

## steps to run devnet

- `go build ./cmd/layerd`

- `cd local_devnet`
- `docker-compose build --no-cache core0 core1 core2 core3 grafana prometheus otel-collector`  
- `docker-compose up core0 core1 and so on`

Note:

- you don't have to build everything if you don't want, ie run only one node

## grafana

- `localhost:3000`  

after starting the chain you can interact with the chain:  
`../layerd query staking validators`

to reset:  
`docker-compose down`
