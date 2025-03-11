// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

struct OracleAttestationData {
    bytes32 queryId;
    ReportData report;
    uint256 attestationTimestamp; //timestamp of validatorSignatures on report
}

struct ReportData {
    bytes value;
    uint256 timestamp; //timestamp of reporter signature aggregation
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

interface IBlobstreamO {
    function guardian() external view returns (address);
    function powerThreshold() external view returns (uint256);
    function unbondingPeriod() external view returns (uint256);
    function validatorTimestamp() external view returns (uint256);
    function verifyOracleData(
        OracleAttestationData calldata _attestData,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs
    ) external view;
}