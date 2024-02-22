# ADR 011: Size Limitations of the Chain

## Authors

@themandalore

## Changelog

- 2024-02-21: initial version

## Context

Layer is limited in many ways:
    - blockSize - how many reports can fit into a block (queryIds with one reporter or multiple reporters on a given queryId)
    - validator Size - how many signatures can fit into ETH block? 
    - voteExtension size limit - How many signatures can we add to vote extensions (queryId's aggregated x validators needed to hit 2/3 consnesns)?
    - If we don't use voteExtensions, how many signatures can we fit into a block? 
    - How fast does our chain grow in size? 


## Alternative Approaches

### altApproach1


## Issues / Notes on Implementation

