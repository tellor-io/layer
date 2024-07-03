# ADR 2007: initialization and time based rewards on bridge
## Authors

@themandalore 

## Changelog

- 2024-07-03: initial version

## Context

When starting layer, there will need to be an initial token supply in order for us to have an initial validator set.  Additionally, we need a way to start time base rewards (inflation) on layer without creating tokens out of thin air.  As per other ADR's, all tokens on layer will be tokens from mainnet Ethereum locked into the bridge contract, so there is a challenge in launching layer and the timing in switching inflationary rewards from the Ethereum mainnet contract to layer.  

The solution to this is to properly order and time the launch of the bridge contract/ layer as well as the switch of inflationary rewards.  
The order expected
    a) Launch bridge contract on Ethereum with initial validator set (public key of expected validator)
    b) Transfer tokens of initial supply on layer to the bridge contract
    * note, not a deposit, but just tokens transferred to the contract.  All tokens just transferred to the bridge contract will be considered inflationary rewards on layer, while tokens deposited through the "deposit" function will be eligible for claiming on layer. 
    c) Next, layer will launch with the initial validator set specified in (a) and token balance transferred in (b)
    d) Layer mainnet will commence with no inflationary rewards.  Parties can still bridge to layer, but the rewars will still be on Ethereum. 
    e) Once layer is deemed ready for users, "updateOracleAddress" will be run on Ethereum and inflationary rewards will be switched from the mainnet Ethereum oracle contract to the token bridge.  Now all newly minted tokens will be transferred to the bridge. 
    f)  Once this happens, layer governance will pass a vote to start inflationary rewards on layer. 
        - Minted rewards can be claimed on mainnet via a withdrawal process due to the TBR on mainnet going to the bridge contract
        - there will likely be a slight time mismatch (just different systems not synced up) between inflation inflation stopping on mainnet and initiating on layer.  This delay will ensure that the tokens minted on layer will always be less than are available in the bridge contract.  
    g) parties looking to exit layer can call the "mintToOracle" function in the token contract to send inflationary tokens to the bridge contract for withdrawal.  



## Alternative Approaches

### deposit and withdraw tbr on layer method

The original plan was to use a method of minting to the bridge contract, then having to bridge it over to layer and have a method for distributing it as rewards.  The problem here is that if we forget to bridge the rewards over, you actually don't have rewards on your chain.  This could cause validators or reporters to halt while the rewards are being bridged, something we want to avoid since the tokens are there. 

### extra tokens minted on creation that can't be withdrawn

We had another idea to handle the launch portion that would mint a certain number of tokens on layer to the team initially and those wouldn't be withdrawn.  This would be fine, but since we want the tokens to be disputable and eligible for earning rewards, this would create an incentive for the holder of these tokens to submit bad data and then dispute themselves to access the tokens. Additionaly, the idea of "new" tokens even if slightly different and locked has a bad image to it and we want to avoid it. 


### launch layer and mint from start

This idea would be to just launch layer and start minting.  We would then switch over the oracle contract on mainnet at somepoint, but there would be minting on both chains for that time being.  Assuming the mismatch amount never tries to withdraw more than in the bridge contract (~4k per month mismatched), you would be fine.  So we would have extra tokens, but as long as not everyone withdraws from layer, the system would work fine. 

This would simplify things, but image wise it's printing unbacked tokens on layer.  It also would be just handing extra tokens to the team since we would likely be the initial validator set (the team should run the chain without rewards to kick start it). It could also lead to race conditions on the bridge to get rewards faster than other reporters on layer.  By not going with this method, we allow for mainnet testing of layer in an opt-in fashion and allow time for reporters to switch over.  

## Issues / Notes on Implementation

