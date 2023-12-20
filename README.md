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

## Starting the Chain

To start the chain locally without Ignite CLI:
`layerd start`

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