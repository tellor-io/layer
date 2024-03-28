# ADR 015: trb bridge structure
## Authors

@themandalore

## Changelog

- 2024-03-28: initial version

## Context

Tellor Tributes (TRB) is the tellor token.  It exists on Ethereum and cannot be changed.  It mints ~4k to the team each month and ~4k to the oracle contract for time based rewards.  When starting layer we will launch a bridging contract where parties can deposit TRB to layer.  Layer will utilize reporters then to report deposit events to our chain.  Once an event is kicked off, all reporters will report for that event for 1 hour and then we will optimistically use the report in our system, ensuring that the report is at least 12 hours old before the tokens are minted on layer.  As an additional security meausure, the layer bridge contract will not allow more than 20% of the total supply on layer to be bridged within a 12 hour period (the function will be locked).  This will be to ensure that someone does not bridge over a very large amount to stake/grief the network, manipulate votes, or grief the system via disputes without proper time to analyze the situation.  For the reverse direction, parties will burn TRB on layer, the reporters will then report that it happened and then the bridge contract on Ethereum can use the tellor data as any other user, but this time reading deposit events.  There will be similar limits in this direction and the bridge contract will also use the data optimistically (12 hours old) to further reduce attack vectors.  

The bridge will also be part of the cycle list, however it will not appear in the "next request" as there is not necessarily a new mint event on a cycle. Instead, parties can use the tip functionality as they see fit normally however they will be eligible for time based rewards to support the reporting. 

## Alternative Approaches

### validators run the bridge

rather than reporters, we could simply have validators natively run the bridge.  This option would work fine and projects like Celer/gravity bridge already have implementations written.  The reason we're going against it is twofold: a) finality issues with Ethereum make this inherently risky for any bridge and b) we should be dogfooding our own product.  There's no reason you can't use tellor as a bridge, so we should do so.  Having validators the run bridge and submit would also require us to write a different bridge structure for the return trip. 

### don't add as part of the cycle list

Forcing parties to report for the bridge might not make sense.  We will likely have multiple mints per Ethereum block and finality is not always constant.  For this reason we went with a longer time frame for submission (1 hour per event).  This can still work with the cycle, but represents using the cycle list in a unique way.  

### don't use optimistically, just use if consensus hit

The reason we want to use it optimistically here is for chain rollbacks.  Although it's unlikely to happen and validator set changes are limited by percent, you could still pretend to deposit a bunch, roll back the chain, and then double spend or dispute on our chain with unlocked tokens.  Since there is no immediacy need for trb to be bridged, we will simply make it take 12 hours and then even limit the amount bridged over. 


## Issues / Notes on Implementation

Note that governance can change the queryId time frame and it should be monitored to make sure that commit times are enough notice on a given tip
