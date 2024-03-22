# Tellor Layer Bridge
The Tellor Layer data bridge relies on attestations of the Layer validator set for proving data existence. The bridge contract is initialized with an initial validator set (addresses and powers). After each new report is aggregated on the Layer chain, all validators sign this data. These signatures are then used to as inputs to the bridge contract for proving a report's existence, and then the data can be consumed by oracle users. In order to minimize gas costs in the bridge contract, validator set updates are only needed after the validator set changes by some threshold such as 5%. Each time the validator set changes by this threshold, the validators attest to this new validator set, and these signatures are relayed to the bridge contract and verified. 

Several pieces of data are included for each aggregated report in order to prove various properties of the report, including the validator power used to generate the aggregate report, validator checkpoint, attestation timestamp, previous report timestamp, and the next report timestamp. 
## Tracking the validator set
```
Validator = {
	address,
	power
}
```

`validatorSet = Validator[]`
- an array of all validator addresses and their powers
  
`validatorSetHash = hash(validatorSet)`


validatorCheckpoint - a hash of the following:
- validator set hash - a digest of all current validator addresses and their powers
- validator nonce - prevents replay attacks when updating validator set
- validator timestamp - tracks when validator set last updated
- validator power threshold - ⅔ of total power, used so bridge contract doesn’t have to calculate this on-chain

## Report Attestations

A report attestation consists of:
- value
- report timestamp
- report aggregate power - how much validator power contributed to this report aggregate?
- validator checkpoint - ensures the correct validator set is being used in the bridge contract. 
- attestation timestamp - communicates how old an attestation is
- previous report timestamp - used for “getDataAfter” proofs
- next report timestamp - used for “getDataBefore” proofs
## Layer Side

On the Layer side, there are two distinct concepts of validator sets: the actual validator set and the bridge validator set. The bridge validator set is updated in response to significant changes (around 1% or more) in the real validator set. Whenever such changes occur, validators collectively endorse the new set.

In addition to validator set changes, validators also provide their attestations to reports. This happens not just for new reports but also for past ones, particularly beneficial for optimistic oracle applications.

It's important to note that once a value is contested, validators cease their attestations for these values. This means that post-dispute, users cannot request or execute attestation proofs for the contested value.

The official bridge validator set list should be sorted by validator power in descending order. This should limit gas costs of running attestation proofs in the bridge contract, as signatures only have to be checked up until 2/3 of total validator power is reached. This sorting should be enforced on the Layer side. 