[![Tests](https://github.com/tellor-io/layer/actions/workflows/go.yml/badge.svg)](https://github.com/tellor-io/layer/actions/workflows/go.yml)

<p align="center">
  <a href='https://twitter.com/WeAreTellor'>
    <img src= 'https://img.shields.io/twitter/url/http/shields.io.svg?style=social' alt='Twitter WeAreTellor' />
  </a>
</p>

## Overview <a name="overview"> </a>  

<b>Tellor Layer</b> is a stand alone L1 built using the cosmos sdk for the purpose of coming to consensus on any subjective data.  It works by using tendermint to agree upon requested data and its values, and in cases where consensus is not reached, falls back to relying on an optimistic approach given reported values can be disputed.

For more in-depth information about Layer, checkout the [TellorLayer - tech paper](https://github.com/tellor-io/layer/blob/main/TellorLayer%20-%20tech.pdf).

## Tests
To run all tests:
`go test -v ./...`

## Starting the Chain (Without Ignite):

1) Remove old test chains (if present):
`rm -rf ~/.layer`
2) Go build layerd:
`go build ./cmd/layerd`
3) Initialize the chain:
`./layerd init layer  --chain-id layer-test-1`
4) Add a validator account:
`./layerd keys add alice`
5) Create a tx to Give the alice loyas to stake:
`./layerd add-genesis-account tellor15sck900lsktpq9enm4v7kspmykg0e0fu7jcr9n 10000000000000loya`
6) Create a tx to Stake some loyas for alice:
`./layerd gentx alice 1000000000000loya  --chain-id layer-test-1`
7) Add the transactions to the genesis block:
`./layerd collect-gentxs`
8) Start the chain:
`layerd start`

## Starting the Chain With Ignite CLI:

To start the chain locally with Ignite CLI:
`ignite chain serve`

To create a transaction, in another terminal:
`layerd tx [command]`

To see all available commands:
`layerd`

## Maintainers <a name="maintainers"> </a>

This repository is maintained by the [Tellor team](https://github.com/orgs/tellor-io/people)

## How to Contribute<a name="how2contribute"> </a>  

Check out our issues log here on Github or feel free to reach out anytime [info@tellor.io](mailto:info@tellor.io)

## Copyright

Tellor Inc. 2022 