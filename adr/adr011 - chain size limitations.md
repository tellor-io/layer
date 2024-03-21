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

### Cap number of reporters (like validators)

You could cap the number of reporters at the total level.  This problem is that we would then be forcing anyone who wants to report to stake a large amount or tip a validator who potentially doesn't care about their small data point (the LINK problem of no one supporting your illiquid coin).  If you want to use it purely optimistically, you shouldn't need to worry about having too much stake.  

#### Cap number of reporters per report

You could take only the top 150 (for example) reports.  That way only the top guys would get rewards on the big queries, but the lesser people would still be able to compete on the lesser reported/illiquid/or optimistic queries.  


## Issues / Notes on Implementation

