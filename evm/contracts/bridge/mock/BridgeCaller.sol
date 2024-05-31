// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "../BlobstreamO.sol";

contract BridgeCaller {
    BlobstreamO public bridge;
    bytes public oracleData;
    uint256 public oracleDataTimestamp;

    constructor(address _bridge) {
        bridge = BlobstreamO(_bridge);
    }

    function verifyAndSaveOracleData(
        OracleAttestationData calldata _attest,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs
    ) public {
        bridge.verifyOracleData(_attest, _currentValidatorSet, _sigs);
        oracleData = _attest.report.value;
        oracleDataTimestamp = _attest.report.timestamp;
    }
}

