# ADR 2009: reporter and bridge daemon structure

## Authors

@themandalore

## Changelog

- 2025-04-09: initial version

## Context

This ADR is meant to go over the location and organization of the tellor layer software as it relates to the various duties of validating, reporting and bridging data.  

For background, the base cosmos-sdk software allows validators to stake and run a validating node or just allow anyone to run a node (pass around messages, read data, and send txns through). Tellor added the additional job of being a reporter.  This is not necessarily the same thing as a validator.  To be a reporter, you scrape API's (or enter data manually) and then submit a transaction to a node. For the bridge function, they listen to the bridge contract on mainnet Ethereum.  This was previously done by telliot in our old system.  

When we first built layer, we copied a structure similar to dydx where a tightly coupled daemon would also run and do this reporting.  The problem arose however that anytime we wanted to update the reporter software (e.g. add a new exchange source), we would need to upgrade the entire node binary.  

As a background, upgrades are fine, but limiting them is best practice.  too many updates on the node software can lead to unforseen breaks in the chain as well as less rigorous examination of changes by validators who get used to just pushing github updates live.  

## Decision - Node and Reporter seperate, bridge tasks split accordingly
 
We decided to have the reporter as a separate binary, allowing for faster updates with just an rpc passed as the connection to the node.  This structure is similar to telliot with the current.  The bridge daemon did handle reporting bridge deposits and attesting to aggregated values.  Now the aggregation piece is kept as part of the node (since its run by validators), and the reporting piece is part of the reporting software (as it's done by reporters). 

The potential downside is that latency could become an issue if a non-local rpc is used.  Additionaly, if operating a node is resource intensive, it could lead to incentizing reporters to use hosted rpc's which could reduce decentralization.  Both issues will need to be monitored. 


## Alternate approaches
### Everything in the layer binary

We tossed around keeping it as it was built originally with one binary, but the potential need for more frequent updates to simply adjust the reporter sources or structure was enough to force the change.  


## Issues / Notes on Implementation







