# ADR 2011: Attestation and valset signature slashing

## Authors

@heavychain

## Changelog

- 2025-06-13: initial version

## Context

We need slashing for validators who sign malicious attestations or validator sets. This makes it more risky for validators to coordinate attacks on the bridge.

The main goal here is to make malicious attestation attacks harder to coordinate. Since the data bridge needs signatures from 2/3 of tellor validators in order to accept data relayed from Layer to Ethereum, a coordinated attack would require validators to share their malicious signatures with someone else to assemble and relay to EVM contracts. With slashing in place, any validator who receives these malicious attestations has an incentive to submit them as evidence and slash their competition.

### Attestation Slashing

If a validator signs an attestation that's different from what actually happened on-chain, anyone can submit evidence to slash them.

Here's how it works:
1. Submit `MsgSubmitAttestationEvidence` with the malicious attestation details
2. The system reconstructs the attestation snapshot from the evidence  
3. Check that this snapshot doesn't exist in consensus (meaning it's actually malicious)
4. Verify the signature matches a real validator
5. Check that the attestation timestamp is not within the 10 minute rate limit window of any previously submitted attestation evidence
6. Look up the block height from the validator checkpoint to determine historical voting power
7. Slash the validator by 1% and jail them

One important detail: if you submit attestation evidence with an invalid checkpoint, the transaction fails because we can't look up the block height. This actually suggests the validator might have signed a bad validator set, so you should submit valset signature evidence instead.

### Valset Signature Slashing

If a validator signs an invalid validator set checkpoint, anyone can submit evidence to slash them.

Process:
1. Submit `MsgSubmitValsetSignatureEvidence` with the bad signature details
2. Reconstruct the checkpoint from the evidence
3. Verify this checkpoint doesn't match any real validator set at that timestamp
4. Verify the signature matches a real validator
5. Check that the valset timestamp is not within the 10 minute rate limit window of any previously submitted valset signature evidence
6. Find the most recent valid checkpoint before this timestamp to get historical power
7. Slash the validator by 5% and jail them

### Rate Limiting

You can only slash a validator once per 10-minute window for each type of evidence, as determined by the attestation timestamp and validator set timestamp. This prevents excessive penalties for related mistaken signatures.

### Block Height Lookup

For slashing to work properly, we need to know the validator's voting power at the time they signed the malicious data. We use the validator checkpoint associated with the evidence to look up the correct block height. The staking module then uses this block height to calculate how much to slash from bonded and delegated tokens.

## Alternative Approaches

### Social Layer Slashing

Instead of building automatic slashing into the protocol, we could rely on the social layer to handle malicious validators. This would mean coordinating hard forks to slash bad actors manually.

We decided against this approach because:
- Hard forks are much harder to coordinate than on-chain slashing
- It requires significant social consensus for each incident
- The delay between detection and punishment makes the system less secure
- Automatic slashing provides immediate consequences

### No Rate Limiting

We could allow unlimited slashing for any bad behavior, but rate limiting prevents validators from losing everything due to multiple related mistakes.

##  Issues / Notes on Implementation

We implemented a higher penalty for validator set signature slashing (5%) than attestation slashing (1%) because signing a bad validator set risks compromising the entire data bridge contract, while signing a bad attestation only risks compromising a single data feed.

Attestations do not include a tellor chain id. This means that mainnet validators should not use the same keys for both mainnet and testnet.