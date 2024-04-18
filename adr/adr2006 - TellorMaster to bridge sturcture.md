# ADR 2006: TellorMaster to bridge structure

## Authors

@themandalore

## Changelog

- 2024-02-22: initial version

## Context

TellorMaster (oracle mints) => Bridge => OracleLightClient(proxy) => OracleLightClient(implemtation)
TellorMaster (team mints) => VotingContract => Bridge => OracleLightClient(proxy) => OracleLightClient(implemtation)


The tellor master contract is the tellor token contract on mainnet Ethereum.  It is non-upgradeable, however the minting of new tokens to both the team and the oracle contract can be changed via the oracle itself.  The current plan is to push both to the bridge contract and allow distribution to be handled by Layer. 

Voting Contract - we will deploy a separate contract that will be a pass through for team minting.  The reason we need this is to ensure that we can still vote in governance disputes on Ethereum mainnet.  All tokens however will be automatically routed to the bridge.  

Bridge Contract - the bridge contract will be non-upgradeable.  This is a one-time gotta get it right scenario (or we'll have to hard fork).  Tokens will be sent to the bridge and layer will read token balances and mint them on layer for reporting/validation purposes.  Parties can unlock tokens from the bridge, but that will be done through the standard tellor light client bridge, that the bridge will read.  The bridge contract will also pass through reports to the current oracle contract on mainnet (for all current users (Liquity)).  The only difference is that reports to the "oracle contract" address will be disabled, thus locking the minting from ever changing again.  

OracleLightClient - all light clients on each chain will be proxies. The reason for this is that they are upgradeable through layer itself (governance votes can update in the case of a breaking hard fork).  Additionally, we have mechanisms in place to fork the implementation contract in the case of a hard fork (e.g. push it to a chain multisig, social layer, or worst case voting contract to determine validity of the chain).  This is so that even if the layer breaks, we can still fork users to the better version (a huge benefit of being an L1). 

OracleLightClient (implementaton) - This will actually house the logic for parsing the validator set and oracle data from layer.  This will not be upgradeable.  New contracts can be deployed with different logic (e.g. drop an attacker from the validator set) and forks will be handled at the layer governance or hard fork fallback (on Eth/the user chain) levels.  

## Alternative Approaches

### No proxy contract

One option was to not have a proxy for the oracle.  This would work and was initially the plan, but maintaining the ability for Tellor to fork is important for security and it's important that users can still work with tellor in this case even if the network they are on does not have a social layer to choose forks (ie Eth).

### upgradeability in bridge contract

We had thought to have upgradeability in the bridge contract in case we'd want to change this for some reason.  We do plan on keeping the bridge contract very minimal and passing most logic over to just reading the oracle contract (which is upgradeable).  Additionally for sake of decentralization, the upgradeability of the bridge contract (whcih will likely hold a large amount of TRB) is a huge security risk.  

### Keep ability to change minting destination

For the minting of tellor tokens, we are passing all new mints over to the bridge so that layer can handle distribution.  Like the upgradeability of the bridge, if we'd want to upgrade where the distributions went, we could keep it from reports either from Layer or from the current mainnet oracle.  Both options pose different security risks though.  For the current oracle, it is likely to be deprecated should users move off of it in time.  Additionally the layer model is still new and switching right at the beginning could lead to security risks in the token contract itself (the one piece that needs a major hardfork to change). 

## Issues / Notes on Implementation

