// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

import "../interfaces/ITellorDataBridge.sol";

contract UsingTellor {
    ITellorDataBridge public dataBridge;

    constructor(address _dataBridge) {
        dataBridge = ITellorDataBridge(_dataBridge);
    }

    function getDataWithFallback(
        OracleAttestationData calldata _attest,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs,
        uint256 _fallbackTimestamp,
        uint256 _fallbackMinimumPower, // or use percentage?
        uint256 _maxAttestationAge
    ) public view{
        dataBridge.verifyOracleData(_attest, _currentValidatorSet, _sigs);
        require(block.timestamp - _attest.attestationTimestamp <= _maxAttestationAge, "Attestation is too old");
        if(_attest.report.aggregatePower >= dataBridge.powerThreshold()) {
            require(_attest.report.timestamp < _fallbackTimestamp, "Report timestamp must be before _fallbackTimestamp");
        }
        require(_attest.report.nextTimestamp == 0 || _attest.report.nextTimestamp > _fallbackTimestamp, "Report is latest before timestamp");
        require(_attest.report.aggregatePower >= _fallbackMinimumPower, "Report aggregate power must be greater than or equal to _minimumPower");
        require(_attest.report.nextTimestamp == 0 || _attest.report.nextTimestamp < _fallbackTimestamp, "Report is latest after fallback timestamp");
        require(_attest.report.nextTimestamp == 0 || _attest.report.nextTimestamp > _fallbackTimestamp, "Report is latest before fallback timestamp");
    }
    
    function isAnyConsensusValue(
        OracleAttestationData calldata _attest,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs,
        uint256 _maxAttestationAge
    ) public view{
        dataBridge.verifyOracleData(_attest, _currentValidatorSet, _sigs);
        require(_attest.report.aggregatePower >= dataBridge.powerThreshold(), "Report aggregate power must be greater than or equal to _minimumPower");
        require(block.timestamp - _attest.attestationTimestamp <= _maxAttestationAge, "Attestation is too old");
    }

    function isCurrentConsensusValue(
        OracleAttestationData calldata _attest,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs,
        uint256 _maxAttestationAge
    ) public view {
        dataBridge.verifyOracleData(_attest, _currentValidatorSet, _sigs);
        require(_attest.report.aggregatePower >= dataBridge.powerThreshold(), "Report aggregate power must be greater than or equal to _minimumPower");
        require(block.timestamp - _attest.attestationTimestamp <= _maxAttestationAge, "Attestation is too old");
        require(_attest.report.nextTimestamp == 0, "Report is not latest");
    }

    function isValidDataAfter(
        OracleAttestationData calldata _attest,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs,
        uint256 _timestampAfter,
        uint256 _maxAge,
        uint256 _minimumPower,
        uint256 _maxAttestationAge
    ) public view{
        require(block.timestamp - _attest.attestationTimestamp <= _maxAttestationAge, "Attestation is too old");
        require(_attest.report.timestamp > _timestampAfter, "Report timestamp must be after _timestamp");
        require(_attest.report.nextTimestamp == 0 || _attest.report.nextTimestamp > _timestampAfter, "Report is latest before timestamp");
        require(_attest.report.aggregatePower >= _minimumPower, "Report aggregate power must be greater than or equal to _minimumPower");
        require(block.timestamp - _attest.report.timestamp < _maxAge, "Report is too old");
        dataBridge.verifyOracleData(_attest, _currentValidatorSet, _sigs);
    }

    function isValidDataBefore(
        OracleAttestationData calldata _attest,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs,
        uint256 _timestampBefore,
        uint256 _maxReportAge,        
        uint256 _minimumPower,
        uint256 _maxAttestationAge
    ) public view{
        require(block.timestamp - _attest.attestationTimestamp <= _maxAttestationAge, "Attestation is too old");
        require(_attest.report.timestamp < _timestampBefore, "Report timestamp must be before _timestampBefore");
        require(_attest.report.nextTimestamp == 0 || _attest.report.nextTimestamp > _timestampBefore, "Report is latest before timestamp");
        require(_attest.report.aggregatePower >= _minimumPower, "Report aggregate power must be greater than or equal to _minimumPower");
        require(block.timestamp - _attest.report.timestamp < _maxReportAge, "Report is too old");
        dataBridge.verifyOracleData(_attest, _currentValidatorSet, _sigs);
    }
}