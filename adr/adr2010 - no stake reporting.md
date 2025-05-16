# ADR 2010: no stake reporting

## Authors

@danflo27

## Changelog

- 2025-05-16: initial version

## Context

This ADR covers the idea of being able to submit data without any stake. Anyone can submit microdata, and validators can attest to each piece of data. 


## Design - free reporting
MsgNoStakeReport allows any address to submit a response (Value) to a question (QueryData). In the first iteration, the limitations for users are the following:
1. QueryData must be smaller than the oracle param QueryDataLimit (default 0.5 MB). 

2. Only one report per queryId can be submitted per block. This is to ensure each response per question has a unique identifier (Timestamp). 

Additionally, to save on storage, the QueryId and not QueryData is being stored with each report. The QueryData is being stored in a seperate map (QueryId:QueryData) so that each question is only being stored once. 

As the concept develops, incorporating query types will be needed. For example, we could have a special query type for creating questions, and no stake reports must correspond to those existing questions.

## Design - attestation requests 
Previously, attestations could be requested for any aggregate. Now, they can also be requested for no stake reports using the same exact function. If the function does not find an aggregate for a given QueryId and Timestamp, it now looks up a no stake report.

Because of the bridge and relayer design, it would be a pain to change the format for an attestation. The attestation for a no stake report has all of the same fields as an attestation for an aggregate, but the irrelevant fields (aggregatePower, previousTimestamp, nextTimestamp, lastConsensusTimestamp) will have zeros.

## Issue - storage/spam

### QueryData size 
No matter the limit on QueryData size, someone can submit multiple reports for different QueryData until a block is filled. Gas should punish this.

### Report Value size 
Similarly to the issue with QueryData, a query type with a bytes value type could be made obnoxiously large. Gas should punish this as well. 

Further testing is required to see how expensive it will be send malicious no stake reports. If needed, we could implement a custom gas cost relative to the size or number of reports already in the current block or previous blocks. 

## Issue - Signaling questions
How will the questions be sent/signalled to be answered ? A no stake report of a certain query type could potentially handle this. 