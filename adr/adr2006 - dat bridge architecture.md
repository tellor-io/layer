# ADR 2006: Layer Data Bridge Architecture

## Authors

@heavychain

## Changelog

- 2024-04-25: initial version

## Context


The Layer data bridge is used for relaying data from Tellor Layer to user chains. Layer validators attest to all oracle data, and these attestations are relayed to a bridge contract on any user chain. The bridge contract verifies that at least 2/3 of the current validator set's power has signed a given report. Note that tellor users are free to access tellor data in any way they choose, and this is just one of many possible implementations. This document communicates the way in which layer facilitates these oracle attestations, as well as signing updates to the validator set over time.
## Bridge Validator Set

The security of the bridge depends on the bridge contract being aware of the layer validator set at any given time. The contract is deployed with an initial validator set, and then updates to the validator set are signed by the validators and relayed to the bridge contract in a similar way to the bridge data itself. The way this is achieved is outlined below.

At the end of each layer block, the bridge module checks for changes to the validator set. If the layer validator set has changed by 5% since the last saved bridge validator set, the bridge validator set is updated. The hash of this new validator set, plus the new 2/3 power threshold and the current block time (also used in the bridge contract to ensure validator set updates always have a greater timestamp), are hashed together to created the latest bridge validator set checkpoint. This checkpoint is what all validators sign in order to update the validator set in the bridge contract. Then, in the vote extension where each validator votes on the latest proposed block, the validators check whether they have signed the latest bridge validator set checkpoint, and if not, they sign and include the signatures with their votes. Finally, the block proposer for the next block updates the layer state to save these checkpoint signatures. 

### Oracle Attestions
The bridge contract depends on validator signatures to verify what data has been aggregated on layer. All validators automatically sign all new aggregate reports, and new attestations of past data can be requested via a transaction. 

#### New Reports
At the end of each block, the bridge module also checks for new aggregate reports created in the current block. For any new aggregate reports, the bridge module begins preparing a new attestation request. This first depends on retrieving all parameters which will be relayed to the bridge contract, including the report queryId, aggregate value, report timestamp, aggregate power, previous report timestamp, next report timestamp, current bridge validator set checkpoint, and the attestation timestamp, which is the current block timestamp. All of these parameters are encoded together and hashed to create a snapshot. 

This snapshot is used for organizing each set of attestations, and it is also the message which will be signed to create the oracle attestation. While some of these parameters are fixed and already saved with the aggregate report, a few of them may change over time, and therefore they are saved separately as snapshot data, which includes the validator checkpoint, attestation timestamp, previous report timestamp, and next report timestamp. We also include the queryId and report timestamp in this snapshot data for purposes of maximizing the amount of space available in the vote extensions. The snapshot data will be needed by bridge relayers when relaying attestations from layer to user chains. The snapshot data is saved to layer state, the new snapshot is added to a list of snapshots for the particular aggregate report (mapped by queryId and timestamp), and then this snapshot is added to a list of attestation requests by the current block height. Then, in the vote extension, all validators check for attestation requests from the previous block. They sign any snapshots and include these signatures (and snapshots) with their block vote. Finally, the block proposer for the next block saves the attestations to state in a mapping from snapshot to attestations. 

#### Past Reports
In order to request new attestations for a past aggregate report, anyone can create a `requestAttestations`  transaction in the bridge module for a given queryId and timestamp. This works the same way as automated current attestation requests in that a snapshot is created, with snapshot data, and this snapshot is added to the current block's list of attestation requests.