# ADR 015: trb bridge structure
## Authors

@themandalore @brendaloya

## Changelog

- 2024-03-28: initial version

## Context

Tellor Tributes (TRB) is the tellor token.  It exists on Ethereum and cannot be changed.  It mints ~4k to the team each month and ~4k to the oracle contract for time based rewards.  When starting Layer we will launch a bridging contract where parties can deposit TRB to Layer.  Layer will utilize reporters then to report deposit events to itself.  When the deposit is made it will be assigned a deposit ID and an event will be kicked off. All reporters will report for that event for a 1 hour window and then we will optimistically use the report in our system, ensuring that the report is at least 12 hours old before the tokens are minted on Layer. Once the value is 12 hours old anyone can mint the tokens on Layer for the specifed deposit ID.  

As an additional security meausure, the bridge contract will not allow more than 20% of the total supply on Layer to be bridged within a 12 hour period (the function will be locked).  This will be to ensure that someone does not bridge over a very large amount to stake/grief the network, manipulate votes, or grief the system via disputes without proper time to analyze the situation.  For the reverse direction, parties will burn TRB on Layer, the reporters will then report that it happened and then the bridge contract on Ethereum can use the tellor data as any other user, but this time reading burn events.  There will be similar limits in this direction and the bridge contract will also use the data optimistically (12 hours old) to further reduce attack vectors.  

The cycle list helps keep the network alive by providing a list of data requests that reporters can support to receive time-based rewards(inflationary rewards) when there are no tips available to claim. Each data request on the cycle list rotates over time so that each request gets pushed on chain on a regular basis. The bridge deposit data request will not appear in the "next request" of the cycle list, however reporters will be allowed to report for it and claim time based rewards for it. Time based rewards will be split between the data request on the cycle list and the bridged deposit request.  Parties can also use the tip functionality to incentivize faster updates for deposits. 

## Alternative Approaches

### validators run the bridge

Rather than reporters, we could simply have validators natively run the bridge.  This option would work fine and projects like Celer/gravity bridge already have implementations written.  The reason we're going against it is twofold: a) finality issues with Ethereum make this inherently risky for any bridge and b) we should be dogfooding our own product.  There's no reason you can't use tellor as a bridge, so we should do so.  Having validators the run bridge and submit would also require us to write a different bridge structure for the return trip. 

### don't add as part of the cycle list

Forcing parties to report for the bridge might not be feasible when there are no deposit events, finality is not always constant, and could force unecessary updates. For this reason we went with a longer time frame for submission (1 hour per event).  This can still work with the cycle, but represents using the cycle list in a unique way.  

### don't use optimistically, just use if consensus hit

The reason we want to use it optimistically here is for chain rollbacks.  Although it's unlikely to happen and validator set changes are limited by percent, you could still pretend to deposit a bunch, roll back the chain, and then double spend or dispute on our chain with unlocked tokens.  Since there is no immediacy need for trb to be bridged, we will simply make it take 12 hours and then even limit the amount bridged over. 

### allow a tip to be included along with the deposit on ethereum

It was considered to allow depositers to include a tip to incentivize reporters to bridge over the data faster to Layer. However, the process for verifying the data was reported to Layer and reporting back the reporters to Ethereum to claim the tip would be inneficient, make the process more complex, and require more storage to track the tips than tipping and distributing tips on Layer. 


## Issues / Notes on Implementation

Note that governance can change the queryId time frame and it should be monitored to make sure that commit times are enough notice on a given tip
