# ADR 2007: initialization and time based rewards on bridge
## Authors

@themandalore 
@brendaloya

## Changelog

- 2024-07-03: initial version
- 2024-07-03: expanded on some scenarios
- 2024-08-12: clean up

## Context

When starting Layer, there will need to be an initial token supply in order for us to have an initial validator set.  Additionally, we need a way to start time base rewards (inflation) on Layer without creating tokens out of thin air. As per other ADR's, all tokens on Layer will be tokens from mainnet Ethereum locked into the bridge contract, so there is a challenge in launching Layer and the timing in switching inflationary rewards from the Ethereum mainnet contract to Layer.  

The solution to this is to properly order and time the launch of the bridge contract/Layer as well as the switch of inflationary rewards. 

The order expected

    a) Launch bridge contract on Ethereum

    b) Transfer tokens of initial supply on Layer to the bridge contract

    * note: not a deposit, but just tokens transferred to the contract.  All tokens just transferred to the bridge contract will be considered inflationary rewards on Layer, while tokens deposited through the "deposit" function will be eligible for claiming on Layer. 

    c) Next, Layer will launch with an initial validator and token balance transferred in (b)

    d) The bridge smart contract is initialized with the initial Layer validator set and block height

    e) Layer mainnet will commence with no inflationary rewards.  Parties can still bridge to Layer, but the rewards will still be on Ethereum. 

    f) Once Layer is deemed ready for users, "updateOracleAddress" will be run on Ethereum and inflationary rewards will be switched from the mainnet Ethereum oracle contract to the token bridge.  Now all newly minted tokens will be transferred to the bridge. 

    g)  Once this happens, Layer governance will pass a vote to start inflationary rewards on Layer. 
        - minted rewards can be claimed on mainnet via a withdrawal process due to the TBR on mainnet going to the bridge contract
        - there will likely be a slight time mismatch (just different systems not synced up) between inflation stopping on mainnet and initiating on Layer.  This delay will ensure that the tokens minted on Layer will always be less than are available in the bridge contract.  
        
    h) parties looking to exit Layer will use the token supply in the bridge, and it will automatically call "mintToOracle" if the contract needs to refresh its inflationary rewards.   


## Alternative Approaches

### Deposit and withdraw tbr on Layer method

The original plan was to use a method of minting to the bridge contract, then having to bridge it over to Layer and have a method for distributing it as rewards. The problem here is that if we forget to bridge the rewards over, there is a possibility of not having any rewards to distribute on the chain. This could cause validators and/or reporters to halt while the rewards are being bridged because it takes 12 hours to bridge tokens. We want to avoid the chain coming to a halt for possibly 12 hours as this could have an adverse effect on our users.

### Extra tokens minted on creation that can't be withdrawn

We had another idea to handle the launch portion that would mint a certain number of tokens on Layer to the team initially and those would be in a locked state and not be eligible for withdrawing. However, the security of the system relies on disputes and not being able to 'lose' tokens can lead to bad incentives. If the tokens were to be used to incentivize other validators (outside the team), it would create a scenario where they are incentivized to provide bad data so that they can 'unlock' the tokens via the dispute mechanism. Additionally, the idea of "new" tokens even if slightly different and locked has a bad image. We didn't go with this option because all of these scenarios seemed like threat to our security.

### Launch Layer and mint from start

This idea would be to just launch Layer and start minting. We would then switch over the oracle contract on mainnet at some point, but there would be minting on both chains for that time being. Assuming the mismatch amount and that withdrawal are never more than what is in the bridge contract (~4k per month mismatched), it could work, but that is a huge assumption. There would be extra tokens, but only as long as not everyone withdraws from Layer.

This was not a good option since it creates a non-liquid position and makes it seem like it's printing unbacked tokens on Layer, which would not be acceptable. It also would be handing extra tokens to the team since we would likely be the initial validator set (the team will run the chain without rewards to kick start it). It could also lead to race conditions on the bridge to get rewards faster than other reporters on Layer. By not going with this method, we allow for mainnet testing of Layer in an opt-in fashion and allow time for reporters to switch over.  

## Issues / Notes on Implementation


