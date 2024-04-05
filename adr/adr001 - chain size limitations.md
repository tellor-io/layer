# ADR 001: Size Limitations of the Chain

## Authors

@themandalore
@brendaloya

## Changelog

- 2024-02-21: initial version
- 2024-04-02: formatting
- 2024-04-01: clarity

## Context

Layer is limited in many ways. This ADR is meant to go over the limits relating to decisions that affect the size of the chain and the ability to bridge data efficiently.  

## Chain 

### General State growth

    How fast does the chain grow in size? 
    What measures / designs are in place for pruning state? 
    Should we consider a data availability layer? Should we assume no-one needs data or verification passed prunning timeframe?

###  Blocksize limits
    
    What is the current blockLimit?
    How many reports can fit into a block
        - How many different queryIds with one reporter?
        - How many reporters can we fit in with one given queryId?
    How many transfers can fit into a block?

## Bridge 

### Validator Size limits

    How many signatures can fit into ETH block? 
    Do we expect ETH to be the main chain data is bridged to?

### Vote extension limits

We have currently opted for implementing signing off on bridge data via vote extensions. However there are still a few unknowns? 
    - What is the size limit for VoteExtension? Currently estimated at about 4MB.
    - How many signatures can we add to vote extensions (queryId's aggregated x validators needed to hit 2/3 consensus)?

If we don't use voteExtensions, 
    - how many signatures can we fit into a block (i.e. store them in the next block)? 
    - Should lanes be implemented? What is a good balance between data reports, transfers, and bridge signatures?


## Alternate approaches to state growth

### Cap number of reporters (like validators)

You could cap the number of reporters at the total level.  This problem is that we would then be forcing anyone who wants to report to stake a large amount or tip a validator who potentially doesn't care about their small data point (the LINK problem of no one supporting your illiquid coin).  If you want to use it purely optimistically, you shouldn't need to worry about having too much stake.  

### Cap number of reporters per report

You could take only the top 150 (for example) reports.  That way only the top guys would get rewards on the big queries, but the lesser people would still be able to compete on the lesser reported/illiquid/or optimistic queries.  


## Issues / Notes on Implementation

