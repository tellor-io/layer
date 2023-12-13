// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.22;

import "./ECDSA.sol";
import "./Constants.sol";
import "./DataRootTuple.sol";
import "./lib/tree/binary/BinaryMerkleProof.sol";
import "./lib/tree/binary/BinaryMerkleTree.sol";

struct Validator {
    address addr;
    uint256 power;
}

struct Signature {
    uint8 v;
    bytes32 r;
    bytes32 s;
}

struct ReportData {
    bytes value;
    uint256 timestamp;
    uint256 consensusThreshold;
}

struct OracleAttestationData {
    bytes32 queryId;
    ReportData report;
    uint256 validatorNonce;
    uint256 powerThreshold;
    bytes32 validatorSetHash;
    uint256 blockTimestamp;
}

/// @title BlobstreamO: Tellor Layer -> EVM, Oracle relay.
/// @dev The relay relies on a set of signers to attest to some event on
/// Tellor Layer. These signers are the validator set, who sign over every
/// block. At least 2/3 of the voting power of the current
/// view of the validator set must sign off on new relayed events.
contract BlobstreamO is ECDSA {
    /*Storage*/
    bytes32 public lastValidatorSetCheckpoint;///Domain-separated commitment to the latest validator set.
    uint256 public powerThreshold;/// Voting power required to submit a new update.
    uint256 public validatorNonce;/// Nonce for bridge events. Must be incremented sequentially.
    uint256 public currentRelayedBlockHeight;
    mapping(uint256 => bytes32) public oracleRoots;/// Mapping of data root tuple root nonces to data root tuple roots.
    mapping(bytes32 => bool) public isOracleRoot;
    /*Events*/
    event NewOracleRoot(bytes32 _oracleRoot);
    event ValidatorSetUpdated(uint256 indexed _nonce, uint256 _powerThreshold, bytes32 _validatorSetHash);

    /*Errors*/
    error MalformedCurrentValidatorSet();
    error InvalidSignature();
    error InsufficientVotingPower();
    error SuppliedValidatorSetInvalid();
    error InvalidValidatorSetNonce();

    /*Functions*/
    /// @param _nonce Initial event nonce.
    /// @param _powerThreshold Initial voting power that is needed to approve operations
    /// @param _validatorSetCheckpoint Initial checkpoint of the validator set. 
    constructor(uint256 _nonce, uint256 _powerThreshold, bytes32 _validatorSetCheckpoint){
        validatorNonce = _nonce;
        lastValidatorSetCheckpoint = _validatorSetCheckpoint;
        powerThreshold = _powerThreshold;
    }

    /// @notice Utility function to verify EIP-191 signatures.
    function verifySig(address _signer, bytes32 _digest, Signature calldata _sig) private pure returns (bool) {
        bytes32 digest_eip191 = ECDSA.toEthSignedMessageHash(_digest);
        return _signer == ECDSA.recover(digest_eip191, _sig.v, _sig.r, _sig.s);
    }

    /// @dev Computes the hash of a validator set.
    /// @param _validators The validator set to hash.
    function computeValidatorSetHash(Validator[] calldata _validators) private pure returns (bytes32) {
        return keccak256(abi.encode(_validators));
    }

    /// @dev A hash of all relevant information about the validator set.
    /// @param _nonce Nonce.
    /// @param _powerThreshold The voting power threshold.
    /// @param _validatorSetHash Validator set hash.
    function domainSeparateValidatorSetHash(uint256 _nonce, uint256 _powerThreshold, bytes32 _validatorSetHash)
        private
        pure
        returns (bytes32)
    {
        return keccak256(abi.encode(VALIDATOR_SET_HASH_DOMAIN_SEPARATOR, _nonce, _powerThreshold, _validatorSetHash));
    }

    /// @dev Checks that enough voting power signed over a digest.
    /// It expects the signatures to be in the same order as the _currentValidators.
    /// @param _currentValidators The current validators.
    /// @param _sigs The current validators' signatures.
    /// @param _digest This is what we are checking they have signed.
    /// @param _powerThreshold At least this much power must have signed.
    function checkValidatorSignatures(
        // The current validator set and their powers
        Validator[] calldata _currentValidators,
        Signature[] calldata _sigs,
        bytes32 _digest,
        uint256 _powerThreshold
    ) private pure {
        uint256 cumulativePower = 0;
        for (uint256 i = 0; i < _currentValidators.length; i++) {
            // If the signature is nil, then it's not present so continue.
            if (_sigs[i].r == 0 && _sigs[i].s == 0 && _sigs[i].v == 0) {
                continue;
            }
            // Check that the current validator has signed off on the hash.
            if (!verifySig(_currentValidators[i].addr, _digest, _sigs[i])) {
                revert InvalidSignature();
            }
            cumulativePower += _currentValidators[i].power;
            // Break early to avoid wasting gas.
            if (cumulativePower >= _powerThreshold) {
                break;
            }
        }
        if (cumulativePower < _powerThreshold) {
            revert InsufficientVotingPower();
        }
    }

    /// @notice This updates the validator set by checking that the validators
    /// in the current validator set have signed off on the new validator set.
    /// @param _newPowerThreshold At least this much power must have signed.
    /// @param _newValidatorSetHash The hash of the new validator set.
    /// @param _currentValidatorSet The current validator set.
    /// @param _sigs Signatures.
    function updateValidatorSet(
        bytes32 _newValidatorSetHash,
        uint64 _newPowerThreshold,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs
    ) external {
        if (_currentValidatorSet.length != _sigs.length) {
            revert MalformedCurrentValidatorSet();
        }
        // Check that the supplied current validator set matches the saved checkpoint.
        bytes32 currentValidatorSetHash = computeValidatorSetHash(_currentValidatorSet);
        if (
            domainSeparateValidatorSetHash(validatorNonce, powerThreshold, currentValidatorSetHash)
                != lastValidatorSetCheckpoint
        ) {
            revert SuppliedValidatorSetInvalid();
        }

        bytes32 newCheckpoint = domainSeparateValidatorSetHash(validatorNonce, _newPowerThreshold, _newValidatorSetHash);
        checkValidatorSignatures(_currentValidatorSet, _sigs, newCheckpoint, powerThreshold);
        lastValidatorSetCheckpoint = newCheckpoint;
        powerThreshold = _newPowerThreshold;
        validatorNonce++;
        emit ValidatorSetUpdated(validatorNonce, _newPowerThreshold, _newValidatorSetHash);
    }

    function verifyOracleData(
        OracleAttestationData calldata _attest, 
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs
    ) public view returns (bool) {
        if (_currentValidatorSet.length != _sigs.length) {
            revert MalformedCurrentValidatorSet();
        }
        bytes32 _currentValidatorSetHash = computeValidatorSetHash(_currentValidatorSet);
        if (
            domainSeparateValidatorSetHash(validatorNonce, powerThreshold, _currentValidatorSetHash)
                != lastValidatorSetCheckpoint
        ) {
            revert SuppliedValidatorSetInvalid();
        }

        if (_attest.validatorNonce != validatorNonce) {
            revert InvalidValidatorSetNonce();
        }
        
        bytes32 _dataDigest = _domainSeparateOracleAttestationData(_attest);
        checkValidatorSignatures(_currentValidatorSet, _sigs, _dataDigest, powerThreshold);
        return true;
    }

    function _domainSeparateOracleAttestationData(OracleAttestationData calldata _attest) internal pure returns (bytes32) {
        return keccak256(
            abi.encode(
                NEW_REPORT_ATTESTATION_DOMAIN_SEPARATOR, 
                _attest.queryId, 
                _attest.report.value, 
                _attest.report.timestamp, 
                _attest.report.consensusThreshold, 
                _attest.validatorNonce, 
                _attest.powerThreshold, 
                _attest.validatorSetHash, 
                _attest.blockTimestamp
            )
        );
    }

}
