// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

import "../bridge/BlobstreamO.sol";

contract UsingTellor {
    BlobstreamO public bridge;

    constructor(address _blobstreamO) {
        bridge = BlobstreamO(_blobstreamO);
    }

    function isCurrentConsensusValue(
        OracleAttestationData calldata _attest,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs,
        uint256 _maxAttestationAge
    ) public view returns(bool) {
        require(bridge.verifyConsensusOracleData(_attest, _currentValidatorSet, _sigs), "UsingTellor: Invalid attestation");
        require(block.timestamp - _attest.attestationTimestamp <= _maxAttestationAge, "UsingTellor: Attestation is too old");
        require(_attest.report.nextTimestamp == 0, "UsingTellor: Report is not latest");
        return true;
    }

    function isAnyConsensusValue(
        OracleAttestationData calldata _attest,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs,
        uint256 _maxAttestationAge
    ) public view returns(bool) {
        require(bridge.verifyConsensusOracleData(_attest, _currentValidatorSet, _sigs), "UsingTellor: Invalid attestation");
        require(block.timestamp - _attest.attestationTimestamp <= _maxAttestationAge, "UsingTellor: Attestation is too old");
        return true;
    }

    function isValidDataBefore(
        OracleAttestationData calldata _attest,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs,
        uint256 _timestampBefore,
        uint256 _maxReportAge,        
        uint256 _minimumPower,
        uint256 _maxAttestationAge
    ) public view returns(bool){
        require(block.timestamp - _attest.attestationTimestamp <= _maxAttestationAge, "UsingTellor: Attestation is too old");
        require(_attest.report.timestamp < _timestampBefore, "UsingTellor: Report timestamp must be before _timestampBefore");
        require(_attest.report.nextTimestamp == 0 || _attest.report.nextTimestamp > _timestampBefore, "UsingTellor: Report is latest before timestamp");
        require(_attest.report.aggregatePower >= _minimumPower, "UsingTellor: Report aggregate power must be greater than or equal to _minimumPower");
        require(block.timestamp - _attest.report.timestamp < _maxReportAge, "UsingTellor: Report is too old");
        require(bridge.verifyOracleData(_attest, _currentValidatorSet, _sigs), "UsingTellor: Invalid signature");
        return true;
    }

    function isValidDataAfter(
        OracleAttestationData calldata _attest,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs,
        uint256 _timestampAfter,
        uint256 _maxAge,
        uint256 _minimumPower,
        uint256 _maxAttestationAge
    ) public view returns(bool){
        require(block.timestamp - _attest.attestationTimestamp <= _maxAttestationAge, "UsingTellor: Attestation is too old");
        require(_attest.report.timestamp > _timestampAfter, "UsingTellor: Report timestamp must be after _timestamp");
        require(_attest.report.nextTimestamp == 0 || _attest.report.nextTimestamp > _timestampAfter, "UsingTellor: Report is latest before timestamp");
        require(_attest.report.aggregatePower >= _minimumPower, "UsingTellor: Report aggregate power must be greater than or equal to _minimumPower");
        require(block.timestamp - _attest.report.timestamp < _maxAge, "UsingTellor: Report is too old");
        require(bridge.verifyOracleData(_attest, _currentValidatorSet, _sigs), "UsingTellor: Invalid signature");
        return true;
    }

    function getDataWithFallback(
        OracleAttestationData calldata _attest,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs,
        uint256 _fallbackTimestamp,
        uint256 _fallbackMinimumPower, // or use percentage?
        uint256 _maxAttestationAge
    ) public view returns(bool){
        require(bridge.verifyOracleData(_attest, _currentValidatorSet, _sigs), "UsingTellor: Invalid signature");
        require(block.timestamp - _attest.attestationTimestamp <= _maxAttestationAge, "UsingTellor: Attestation is too old");
        if(_attest.report.aggregatePower >= bridge.powerThreshold()) {
            require(_attest.report.timestamp < _fallbackTimestamp, "UsingTellor: Report timestamp must be before _fallbackTimestamp");
        }
        require(_attest.report.nextTimestamp == 0 || _attest.report.nextTimestamp > _fallbackTimestamp, "UsingTellor: Report is latest before timestamp");
        require(_attest.report.aggregatePower >= _fallbackMinimumPower, "UsingTellor: Report aggregate power must be greater than or equal to _minimumPower");
        require(_attest.report.nextTimestamp == 0 || _attest.report.nextTimestamp < _fallbackTimestamp, "UsingTellor: Report is latest after fallback timestamp");
        require(_attest.report.nextTimestamp == 0 || _attest.report.nextTimestamp > _fallbackTimestamp, "UsingTellor: Report is latest before fallback timestamp");
        return true;
    }
}