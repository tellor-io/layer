# ADR 003: Nonces for Bridging

## Authors

@themandalore

## Changelog

- 2024-02-21: initial version

## Context

We currently have several nonces - validator set nonce and a timestamp (nonce).  This is separate than the original blobstream which also had a univesal nonce.  For us, using the timestamp as a nonce, where it just must be greater than the previous one, is advantageous for skipping blocks.  


## Alternative Approaches

### require a universal nonce

This requires pushing of each request. 


## Issues / Notes on Implementation

## Links

https://github.com/celestiaorg/blobstream-contracts 