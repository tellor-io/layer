// SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.19;

import {ECDSA} from "./ECDSA.sol";
import "./Constants.sol";

struct OracleAttestationData {
    bytes32 queryId;
    ReportData report;
    uint256 attestationTimestamp;//timestamp of validatorSignatures on report
}

struct ReportData {
    bytes value;
    uint256 timestamp;//timestamp of reporter signature aggregation
    uint256 aggregatePower;
    uint256 previousTimestamp;
    uint256 nextTimestamp;
    uint256 lastConsensusTimestamp;
}

struct Signature {
    uint8 v;
    bytes32 r;
    bytes32 s;
}

struct Validator {
    address addr;
    uint256 power;
}


/// @title TellorDataBridge: Tellor Layer -> EVM, Oracle relay.
/// @dev The relay relies on a set of signers to attest to some event on
/// Tellor Layer. These signers are the validator set, who sign over every
/// block. At least 2/3 of the voting power of the current
/// view of the validator set must sign off on new relayed events.
contract TellorDataBridge is ECDSA {

    /*Storage*/
    address public guardian; /// Able to reset the validator set only if the validator set becomes stale.
    bytes32 public lastValidatorSetCheckpoint; ///Domain-separated commitment to the latest validator set.
    uint256 public powerThreshold; /// Voting power required to submit a new update.
    uint256 public unbondingPeriod; /// Time period after which a validator can withdraw their stake.
    uint256 public validatorTimestamp; /// Timestamp of the block where validator set is updated.
    address public deployer; /// Address that deployed the contract.
    bool public initialized; /// True if the contract is initialized.
    uint256 public constant MS_PER_SECOND = 1000; // factor to convert milliseconds to seconds

    /*Events*/
    event GuardianResetValidatorSet(uint256 _powerThreshold, uint256 _validatorTimestamp, bytes32 _validatorSetHash);
    event ValidatorSetUpdated(uint256 _powerThreshold, uint256 _validatorTimestamp, bytes32 _validatorSetHash);

    /*Errors*/
    error AlreadyInitialized();
    error InsufficientVotingPower();
    error InvalidPowerThreshold();
    error InvalidSignature();
    error MalformedCurrentValidatorSet();
    error NotDeployer();
    error NotGuardian();
    error StaleValidatorSet();
    error SuppliedValidatorSetInvalid();
    error ValidatorSetNotStale();
    error ValidatorTimestampMustIncrease();

    /*Functions*/
    /// @notice Constructor for the TellorDataBridge contract.
    /// @param _guardian Guardian address.
    constructor(
        address _guardian
    ) {
        guardian = _guardian;
        deployer = msg.sender;
    }

    /// @notice This function is called only once by the deployer to initialize the contract
    /// @param _powerThreshold Initial voting power that is needed to approve operations
    /// @param _validatorTimestamp Timestamp of the block where validator set is updated.
    /// @param _unbondingPeriod Time period after which a validator can withdraw their stake.
    /// @param _validatorSetCheckpoint Initial checkpoint of the validator set.
    function init(
        uint256 _powerThreshold,
        uint256 _validatorTimestamp,
        uint256 _unbondingPeriod,
        bytes32 _validatorSetCheckpoint
    ) external {
        if (msg.sender != deployer) {
            revert NotDeployer();
        }
        if (initialized) {
            revert AlreadyInitialized();
        }
        initialized = true;
        powerThreshold = _powerThreshold;
        validatorTimestamp = _validatorTimestamp;
        unbondingPeriod = _unbondingPeriod;
        lastValidatorSetCheckpoint = _validatorSetCheckpoint;
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
        if (block.timestamp - (validatorTimestamp / MS_PER_SECOND) < unbondingPeriod) {
            revert ValidatorSetNotStale();
        }
        if (_validatorTimestamp <= validatorTimestamp) {
            revert ValidatorTimestampMustIncrease();
        }
        powerThreshold = _powerThreshold;
        validatorTimestamp = _validatorTimestamp;
        lastValidatorSetCheckpoint = _validatorSetCheckpoint;
        emit GuardianResetValidatorSet(_powerThreshold, _validatorTimestamp, _validatorSetCheckpoint);
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
        if (_newValidatorTimestamp < validatorTimestamp) {
            revert ValidatorTimestampMustIncrease();
        }
        if (_newPowerThreshold == 0) {
            revert InvalidPowerThreshold();
        }
        // Check that the supplied current validator set matches the saved checkpoint.
        bytes32 _currentValidatorSetHash = keccak256(abi.encode(_currentValidatorSet));
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
    /// @notice This getter verifies a given piece of data vs Validator signatures
    /// @param _attestData The data being verified
    /// @param _currentValidatorSet array of current validator set
    /// @param _sigs Signatures.
    function verifyOracleData(
        OracleAttestationData calldata _attestData,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs
    ) external view{
        if (_currentValidatorSet.length != _sigs.length) {
            revert MalformedCurrentValidatorSet();
        }
        // Check that the supplied current validator set matches the saved checkpoint.
        if (
            _domainSeparateValidatorSetHash(
                powerThreshold,
                validatorTimestamp,
                keccak256(abi.encode(_currentValidatorSet))
            ) != lastValidatorSetCheckpoint
        ) {
            revert SuppliedValidatorSetInvalid();
        }
        bytes32 _dataDigest = keccak256(
                abi.encode(
                    NEW_REPORT_ATTESTATION_DOMAIN_SEPARATOR,
                    _attestData.queryId,
                    _attestData.report.value,
                    _attestData.report.timestamp,
                    _attestData.report.aggregatePower,
                    _attestData.report.previousTimestamp,
                    _attestData.report.nextTimestamp,
                    lastValidatorSetCheckpoint,
                    _attestData.attestationTimestamp,
                    _attestData.report.lastConsensusTimestamp
                )
            );
        _checkValidatorSignatures(
            _currentValidatorSet,
            _sigs,
            _dataDigest,
            powerThreshold
        );
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
        if (block.timestamp - (validatorTimestamp / MS_PER_SECOND) > unbondingPeriod) {
            revert StaleValidatorSet();
        }
        uint256 _cumulativePower = 0;
        for (uint256 _i = 0; _i < _currentValidators.length; _i++) {
            // If the signature is nil, then it's not present so continue.
            if (_sigs[_i].r == 0 && _sigs[_i].s == 0 && _sigs[_i].v == 0) {
                continue;
            }
            // Check that the current validator has signed off on the hash.
            if (!_verifySig(_currentValidators[_i].addr, _digest, _sigs[_i])) {
                revert InvalidSignature();
            }
            _cumulativePower += _currentValidators[_i].power;
            // Break early to avoid wasting gas.
            if (_cumulativePower >= _powerThreshold) {
                break;
            }
        }
        if (_cumulativePower < _powerThreshold) {
            revert InsufficientVotingPower();
        }
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

    /// @notice Utility function to verify Tellor Layer signatures
    /// @param _signer The address that signed the message.
    /// @param _digest The digest that was signed.
    /// @param _sig The signature.
    /// @return bool True if the signature is valid.
    function _verifySig(
        address _signer,
        bytes32 _digest,
        Signature calldata _sig
    ) internal pure returns (bool) {
        _digest = sha256(abi.encodePacked(_digest));
        (address _recovered, RecoverError error, ) = tryRecover(_digest, _sig.v, _sig.r, _sig.s);
        if (error != RecoverError.NoError) {
            revert InvalidSignature();
        }
        return _signer == _recovered;
    }
}

