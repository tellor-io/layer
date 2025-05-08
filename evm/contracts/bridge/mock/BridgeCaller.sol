// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "../../interfaces/ITellorDataBridge.sol";

contract BridgeCaller {
    ITellorDataBridge public dataBridge;
    bytes public oracleData;
    uint256 public oracleDataTimestamp;

    constructor(address _dataBridge) {
        dataBridge = ITellorDataBridge(_dataBridge);
    }

    function verifyAndSaveOracleData(
        OracleAttestationData calldata _attest,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs
    ) public {
        dataBridge.verifyOracleData(_attest, _currentValidatorSet, _sigs);
        oracleData = _attest.report.value;
        oracleDataTimestamp = _attest.report.timestamp;
    }
}

