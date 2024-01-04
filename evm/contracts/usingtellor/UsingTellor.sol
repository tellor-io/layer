// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

import "../bridge/BlobstreamO.sol";

contract UsingTellor {
    BlobstreamO public bridge;
    uint256 public constant MAX_ATTESTATION_AGE = 5 minutes;

    constructor(address _blobstreamO) {
        bridge = BlobstreamO(_blobstreamO);
    }

    function isCurrentConsensusValue(
        OracleAttestationData calldata _attest,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs
    ) public view returns(bool) {
        require(bridge.verifyConsensusOracleData(_attest, _currentValidatorSet, _sigs), "Invalid attestation");
        require(block.timestamp - _attest.attestTimestamp <= MAX_ATTESTATION_AGE, "Attestation is too old");
        require(_attest.report.nextTimestamp == 0, "Report is not latest");
        return true;
    }

    function isValidDataBefore(
        OracleAttestationData calldata _attest,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs,
        uint256 _timestamp,
        uint256 _maxAge,        
        uint256 _minimumPower
    ) public view returns(bool){
        require(bridge.verifyOracleData(_attest, _currentValidatorSet, _sigs), "Invalid signature");
        require(block.timestamp - _attest.attestTimestamp <= MAX_ATTESTATION_AGE, "Attestation is too old");
        require(_attest.report.timestamp < _timestamp, "Report timestamp must be before _timestamp");
        require(_attest.report.nextTimestamp == 0 || _attest.report.nextTimestamp > _timestamp, "Report is latest before timestamp");
        require(_attest.report.aggregatePower >= _minimumPower, "Report aggregate power must be greater than or equal to _minimumPower");
        return true;
    }

    function isValidDataAfter(
        OracleAttestationData calldata _attest,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs,
        uint256 _timestamp,
        uint256 _maxAge, //?
        uint256 _minimumPower
    ) public view returns(bool){
        require(bridge.verifyOracleData(_attest, _currentValidatorSet, _sigs), "Invalid signature");
        require(block.timestamp - _attest.attestTimestamp <= MAX_ATTESTATION_AGE, "Attestation is too old");
        require(_attest.report.timestamp > _timestamp, "Report timestamp must be after _timestamp");
        require(_attest.report.nextTimestamp == 0 || _attest.report.nextTimestamp > _timestamp, "Report is latest before timestamp");
        require(_attest.report.aggregatePower >= _minimumPower, "Report aggregate power must be greater than or equal to _minimumPower");
        return true;
    }

    function getDataWithFallback(
        OracleAttestationData calldata _attest,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs,
        uint256 _fallbackTimestamp,
        uint256 _fallbackMinimumPower
    ) public view returns(bool){
        require(bridge.verifyOracleData(_attest, _currentValidatorSet, _sigs), "Invalid signature");
        require(block.timestamp - _attest.attestTimestamp <= MAX_ATTESTATION_AGE, "Attestation is too old");
        if(_attest.report.aggregatePower >= bridge.powerThreshold()) {
            require(_attest.report.timestamp < _fallbackTimestamp, "Report timestamp must be before _fallbackTimestamp");
        }
        require(_attest.report.nextTimestamp == 0 || _attest.report.nextTimestamp > _fallbackTimestamp, "Report is latest before timestamp");
        require(_attest.report.aggregatePower >= _fallbackMinimumPower, "Report aggregate power must be greater than or equal to _minimumPower");
        require(_attest.report.nextTimestamp == 0 || _attest.report.nextTimestamp < _fallbackTimestamp, "Report is latest after fallback timestamp");
        require(_attest.report.nextTimestamp == 0 || _attest.report.nextTimestamp > _fallbackTimestamp, "Report is latest before fallback timestamp");
        return true;
    }
}