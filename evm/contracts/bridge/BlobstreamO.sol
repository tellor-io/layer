// SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.22;

import "./ECDSA.sol";
import "./Constants.sol";

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
    uint256 aggregatePower;
    uint256 previousTimestamp;
    uint256 nextTimestamp;
}

struct OracleAttestationData {
    bytes32 queryId;
    ReportData report;
    uint256 attestTimestamp;
}

/// @title BlobstreamO: Tellor Layer -> EVM, Oracle relay.
/// @dev The relay relies on a set of signers to attest to some event on
/// Tellor Layer. These signers are the validator set, who sign over every
/// block. At least 2/3 of the voting power of the current
/// view of the validator set must sign off on new relayed events.
contract BlobstreamO is ECDSA {
    /*Storage*/
    bytes32 public lastValidatorSetCheckpoint; ///Domain-separated commitment to the latest validator set.
    uint256 public powerThreshold; /// Voting power required to submit a new update.
    uint256 public validatorTimestamp; /// Timestamp of the block where validator set is updated.
    uint256 public unbondingPeriod; /// Time period after which a validator can withdraw their stake.
    address public guardian; /// Able to reset the validator set only if the validator set becomes stale.
    /*Events*/
    event ValidatorSetUpdated(
        uint256 _powerThreshold,
        uint256 _validatorTimestamp,
        bytes32 _validatorSetHash
    );

    /*Errors*/
    error InsufficientVotingPower();
    error InvalidSignature();
    error MalformedCurrentValidatorSet();
    error NotGuardian();
    error StaleValidatorSet();
    error SuppliedValidatorSetInvalid();
    error ValidatorSetNotStale();
    error NotConsensusValue();

    /*Functions*/
    /// @param _powerThreshold Initial voting power that is needed to approve operations
    /// @param _validatorTimestamp Timestamp of the block where validator set is updated.
    /// @param _unbondingPeriod Time period after which a validator can withdraw their stake.
    /// @param _validatorSetCheckpoint Initial checkpoint of the validator set.
    /// @param _guardian Guardian address.
    constructor(
        uint256 _powerThreshold,
        uint256 _validatorTimestamp,
        uint256 _unbondingPeriod,
        bytes32 _validatorSetCheckpoint,
        address _guardian
    ) {
        powerThreshold = _powerThreshold;
        validatorTimestamp = _validatorTimestamp;
        unbondingPeriod = _unbondingPeriod;
        lastValidatorSetCheckpoint = _validatorSetCheckpoint;
        guardian = _guardian;
    }

    /// @notice This function is called by the guardian to reset the validator set
    /// only if it becomes stale.
    /// @param _powerThreshold Amount of voting power needed to approve operations.
    /// @param _validatorTimestamp The timestamp of the block where validator set is updated.
    /// @param _validatorSetCheckpoint The hash of the validator set.
    function guardianResetValidatorSet(
        uint256 _powerThreshold,
        uint256 _validatorTimestamp,
        bytes32 _validatorSetCheckpoint
    ) external {
        if (msg.sender != guardian) {
            revert NotGuardian();
        }
        if (block.timestamp - validatorTimestamp < unbondingPeriod) {
            revert ValidatorSetNotStale();
        }
        powerThreshold = _powerThreshold;
        validatorTimestamp = _validatorTimestamp;
        lastValidatorSetCheckpoint = _validatorSetCheckpoint;
    }

    /// @notice This updates the validator set by checking that the validators
    /// in the current validator set have signed off on the new validator set.
    /// @param _newValidatorSetHash The hash of the new validator set.
    /// @param _newPowerThreshold At least this much power must have signed.
    /// @param _newValidatorTimestamp The timestamp of the block where validator set is updated.
    /// @param _currentValidatorSet The current validator set.
    /// @param _sigs Signatures.
    function updateValidatorSet(
        bytes32 _newValidatorSetHash,
        uint64 _newPowerThreshold,
        uint256 _newValidatorTimestamp,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs
    ) external {
        if (_currentValidatorSet.length != _sigs.length) {
            revert MalformedCurrentValidatorSet();
        }
        // Check that the supplied current validator set matches the saved checkpoint.
        bytes32 _currentValidatorSetHash = _computeValidatorSetHash(
            _currentValidatorSet
        );
        if (
            _domainSeparateValidatorSetHash(
                powerThreshold,
                validatorTimestamp,
                _currentValidatorSetHash
            ) != lastValidatorSetCheckpoint
        ) {
            revert SuppliedValidatorSetInvalid();
        }

        bytes32 _newCheckpoint = _domainSeparateValidatorSetHash(
            _newPowerThreshold,
            _newValidatorTimestamp,
            _newValidatorSetHash
        );
        _checkValidatorSignatures(
            _currentValidatorSet,
            _sigs,
            _newCheckpoint,
            powerThreshold
        );
        lastValidatorSetCheckpoint = _newCheckpoint;
        powerThreshold = _newPowerThreshold;
        validatorTimestamp = _newValidatorTimestamp;
        emit ValidatorSetUpdated(
            _newPowerThreshold,
            _newValidatorTimestamp,
            _newValidatorSetHash
        );
    }

    /*Getter functions*/
    function verifyOracleData(
        OracleAttestationData calldata _attest,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs
    ) external view returns (bool) {
        return _verifyOracleData(_attest, _currentValidatorSet, _sigs);
    }

    function verifyConsensusOracleData(
        OracleAttestationData calldata _attest,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs
    ) external view returns (bool) {
        if (_attest.report.aggregatePower < powerThreshold) {
            revert NotConsensusValue();
        }
        return _verifyOracleData(_attest, _currentValidatorSet, _sigs);
    }

    /*Internal functions*/
    /// @dev Checks that enough voting power signed over a digest.
    /// It expects the signatures to be in the same order as the _currentValidators.
    /// @param _currentValidators The current validators.
    /// @param _sigs The current validators' signatures.
    /// @param _digest This is what we are checking they have signed.
    /// @param _powerThreshold At least this much power must have signed.
    function _checkValidatorSignatures(
        // The current validator set and their powers
        Validator[] calldata _currentValidators,
        Signature[] calldata _sigs,
        bytes32 _digest,
        uint256 _powerThreshold
    ) internal view {
        if (block.timestamp - validatorTimestamp > unbondingPeriod) {
            revert StaleValidatorSet();
        }
        uint256 _cumulativePower = 0;
        for (uint256 i = 0; i < _currentValidators.length; i++) {
            // If the signature is nil, then it's not present so continue.
            if (_sigs[i].r == 0 && _sigs[i].s == 0 && _sigs[i].v == 0) {
                continue;
            }
            // Check that the current validator has signed off on the hash.
            if (!_verifySig(_currentValidators[i].addr, _digest, _sigs[i])) {
                revert InvalidSignature();
            }
            _cumulativePower += _currentValidators[i].power;
            // Break early to avoid wasting gas.
            if (_cumulativePower >= _powerThreshold) {
                break;
            }
        }
        if (_cumulativePower < _powerThreshold) {
            revert InsufficientVotingPower();
        }
    }

    /// @dev Computes the hash of a validator set.
    /// @param _validators The validator set to hash.
    /// @return The hash of the validator set.
    function _computeValidatorSetHash(
        Validator[] calldata _validators
    ) internal pure returns (bytes32) {
        return keccak256(abi.encode(_validators));
    }

    /// @dev A hash of all relevant information about the oracle attestation.
    /// @param _attest The oracle attestation.
    /// @return The domain separated hash of the oracle attestation.
    function _domainSeparateOracleAttestationData(
        OracleAttestationData calldata _attest
    ) internal view returns (bytes32) {
        return
            keccak256(
                abi.encode(
                    NEW_REPORT_ATTESTATION_DOMAIN_SEPARATOR,
                    _attest.queryId,
                    _attest.report.value,
                    _attest.report.timestamp,
                    _attest.report.aggregatePower,
                    _attest.report.previousTimestamp,
                    _attest.report.nextTimestamp,
                    lastValidatorSetCheckpoint,
                    _attest.attestTimestamp
                )
            );
    }

    /// @dev A hash of all relevant information about the validator set.
    /// @param _powerThreshold Amount of voting power needed to approve operations. (2/3 of total)
    /// @param _validatorTimestamp The timestamp of the block where validator set is updated.
    /// @param _validatorSetHash Validator set hash.
    /// @return The domain separated hash of the validator set.
    function _domainSeparateValidatorSetHash(
        uint256 _powerThreshold,
        uint256 _validatorTimestamp,
        bytes32 _validatorSetHash
    ) internal pure returns (bytes32) {
        return
            keccak256(
                abi.encode(
                    VALIDATOR_SET_HASH_DOMAIN_SEPARATOR,
                    _powerThreshold,
                    _validatorTimestamp,
                    _validatorSetHash
                )
            );
    }

    /// @notice Used for verifying oracle data attestations
    function _verifyOracleData(
        OracleAttestationData calldata _attest,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs
    ) internal view returns (bool) {
        if (_currentValidatorSet.length != _sigs.length) {
            revert MalformedCurrentValidatorSet();
        }
        // Check that the supplied current validator set matches the saved checkpoint.
        bytes32 _currentValidatorSetHash = _computeValidatorSetHash(
            _currentValidatorSet
        );
        if (
            _domainSeparateValidatorSetHash(
                powerThreshold,
                validatorTimestamp,
                _currentValidatorSetHash
            ) != lastValidatorSetCheckpoint
        ) {
            revert SuppliedValidatorSetInvalid();
        }
        bytes32 _dataDigest = _domainSeparateOracleAttestationData(_attest);
        _checkValidatorSignatures(
            _currentValidatorSet,
            _sigs,
            _dataDigest,
            powerThreshold
        );
        return true;
    }

    /// @notice Utility function to verify EIP-191 signatures.
    /// @param _signer The address that signed the message.
    /// @param _digest The digest that was signed.
    /// @param _sig The signature.
    /// @return bool True if the signature is valid.
    function _verifySig(
        address _signer,
        bytes32 _digest,
        Signature calldata _sig
    ) internal pure returns (bool) {
        bytes32 digest_eip191 = ECDSA.toEthSignedMessageHash(_digest);
        return _signer == ECDSA.recover(digest_eip191, _sig.v, _sig.r, _sig.s);
    }
}
