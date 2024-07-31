# How to create multisig key through cli using layerd binray stored in test backend

- Purpose: to create keys locally and share pubkey in order to generate a multikey that will create a tellor prefixed address that is controlled by generating keys.

## Create single keys

```sh
layerd keys add key1  --keyring-backend test --keyring-dir ~/.layer
```

```sh
layerd keys add key2  --keyring-backend test --keyring-dir ~/.layer
```

```sh
layerd keys add key3  --keyring-backend test --keyring-dir ~/.layer
```

## Show created keys pubkey to share w/others

```sh
layerd keys show key1  --keyring-backend test --keyring-dir ~/.layer -p
```

example output:
`{"@type":"/cosmos.crypto.secp256k1.PubKey","key":"A1Td9cx4t8nKF42C9ag+Hron4Ny32r2365E0eoVlHhpZ"}`

## Create a multi key in different device given pub keys

- first add keys locally using pubkey

```sh
layerd keys add key1 --pubkey '{"@type":"/cosmos.crypto.secp256k1.PubKey","key":"A00994obHdjRcP+C1NTwMcRbC5b+C8uQjqsYg9Xcf5ZO"}'  --keyring-backend test --keyring-dir ~/.layer
```

```sh
layerd keys add key2 --pubkey '{"@type":"/cosmos.crypto.secp256k1.PubKey","key":"AsetxjcT/JPB9t5ddPHoou5zEW32Kst2mMWoxzjnxAsC"}'  --keyring-backend test --keyring-dir ~/.layer
```

```sh
layerd keys add key3 --pubkey '{"@type":"/cosmos.crypto.secp256k1.PubKey","key":"A6dkBv59j1xL1zHGahPq0s6gO2aSUX2wAZyUwey57l+Q"}'  --keyring-backend test --keyring-dir ~/.layer
```

## Generate or create the multi using the public keys

```sh
layerd keys add teammulti --multisig key1,key2,key3 --multisig-threshold 2  --keyring-backend test --keyring-dir ~/.layer
```

## Show mulitsig address

```sh
layerd keys show teammulti --keyring-backend test --keyring-dir ~/.layer -a
```

example output:
`tellor1025y9ux4uprvu6fchjxja6n555teeqvqadg97q`
