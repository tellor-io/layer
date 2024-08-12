# ADR 001: Size Limitations of the Chain

## Authors

@themandalore
@brendaloya

## Changelog

- 2024-02-21: initial version
- 2024-04-02: formatting
- 2024-04-01: clarity
- 2024-08-03: progress update

## Context

This ADR is meant to go over the limits relating to decisions that affect the size of the chain and the ability to bridge data efficiently. The decision records are split on decisions made that had a direct effect on the chain and bridge. Further testing is needed to finalize these decisions/parameters.

## Chain 

### General State growth

    How fast does the chain grow in size? The initial tests show overall disk storage increasing about 3GB a day. The difference between the size of the chain and total storage is about 6-8GB. For example, it has been observed that the total storage being used showed as 18GB but the size of the chain was 10GB. The difference seems to be largely attributed to the log files. As we move forward on setting up the test nodes, validators, and reporters the log files will be pruned periodically. To ensure we are able to view the logs if something goes wrong there will be two logs, one historical and one current. The historical will be deleted daily after the current log is renamed to become the historical file and a new current is created.  

    What measures / designs are in place for pruning state? The chain will also be pruned periodically. Because disputes have a window of 21 days that is the minimun history the chain needs to keep. The team will keep archive nodes (and anyone is welcome to keep one as well) with state sync. State sync is necessary to allow other nodes to join the network without having to fully sync back to genesis. This will make it efficient for nodes, validators, and reporters to join the network as well as keep the storage requirements lower to become part of the network.

    Should we consider a data availability layer? Should we assume no-one needs data or verification passed pruning timeframe?
    Using a data availability layer is still an open question.
    

###  Blocksize limits
    
    What is the current blockLimit?
    How many reports can fit into a block
        - How many different queryIds with one reporter?
        - How many reporters can we fit in with one given queryId?
    How many transfers can fit into a block?
    How many delegators can unstake/change?  

## Bridge 

### Validator Size limits

    How many signatures can fit into ETH block? 
    Do we expect ETH to be the main chain data is bridged to?

### Vote extension limits

We have currently opted for implementing signing off on bridge data via vote extensions. However, there are still a few unknowns? 
    - What is the size limit for VoteExtension? Currently estimated at about 4MB.
    - How many signatures can we add to vote extensions (queryId's aggregated x validators needed to hit 2/3 consensus)?

If we don't use voteExtensions, 
    - how many signatures can we fit into a block (i.e. store them in the next block)? 
    - Should lanes be implemented? What is a good balance between data reports, transfers, and bridge signatures?

## Alternate approaches to state growth

### Cap number of reporters (like validators)

You could cap the number of reporters at the total level.  This problem is that we would then be forcing anyone who wants to report to stake a large amount or tip a validator who potentially doesn't care about their small data point (the LINK problem of no one supporting your illiquid coin). If you want to use it purely optimistically, you shouldn't need to worry about having too much stake.  

### Cap number of reporters per report

You could take only the top 150 (for example) reports.  That way only the top guys would get rewards on the big queries, but the lesser people would still be able to compete on the lesser reported/illiquid/or optimistic queries.  

## Issues / Notes on Implementation







