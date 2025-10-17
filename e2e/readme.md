# E2E Tests

These are end to end tests using the [interchaintest](https://github.com/strangelove-ventures/interchaintest) framework. These tests spin up a live chain with a given number of nodes/validators in docker that you can run transactions and queries against. To run all e2e tests:

Install heighliner:
```sh
make get-heighliner
```
Create image:
```sh
make local-image
```
Run all e2e tests:
```sh
make e2e
```
Run an individual test:
```sh
cd e2e
go test -v -run TestLayerFlow -timeout 10m
```

Note: 
- Image needs to be rebuilt everytime the chain logic is changed.  
- When writing a test, do not reuse Layer Spinup