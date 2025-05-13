// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import "../interfaces/ITellorDataBridge.sol";

contract SimpleLayerUser {
    ITellorDataBridge public dataBridge;
    PriceData[] public priceData;
    bytes32 public queryId;

    struct PriceData {
        uint256 price; // reported price
        uint256 timestamp; // aggregate report timestamp
        uint256 aggregatePower; // aggregate reporter power
        uint256 previousTimestamp; // previous report timestamp
        uint256 nextTimestamp; // next report timestamp
        uint256 relayTimestamp; // time relayed data included in block
        uint256 attestationTimestamp; // time of attestation
        uint256 userTriggerTimestamp; // time user decided to tip
        uint256 beginRelayTimestamp; // time relay tx initiated
    }

    constructor(address _dataBridge, bytes32 _queryId) {
        dataBridge = ITellorDataBridge(_dataBridge);
        queryId = _queryId;
    }

    function updateOracleData(
        OracleAttestationData calldata _attestData,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs,
        uint256 _userTriggerTimestamp,
        uint256 _beginRelayTimestamp
    ) external {
        require(_attestData.queryId == queryId, "Invalid queryId");
        dataBridge.verifyOracleData(_attestData, _currentValidatorSet, _sigs);
        require(block.timestamp - _attestData.report.timestamp / 1000 < 6 hours, "data too old");
        require(block.timestamp - _attestData.attestationTimestamp / 1000 < 10 minutes, "attestation too old");
        if(priceData.length > 0) {
            require(_attestData.report.timestamp > priceData[priceData.length - 1].timestamp, "report timestamp not increasing");
        }
        uint256 _price = abi.decode(_attestData.report.value, (uint256));
        priceData.push(PriceData(
            _price, 
            _attestData.report.timestamp, 
            _attestData.report.aggregatePower, 
            _attestData.report.previousTimestamp, 
            _attestData.report.nextTimestamp,
            block.timestamp,
            _attestData.attestationTimestamp,
            _userTriggerTimestamp,
            _beginRelayTimestamp
        ));
    }

    function getCurrentPriceData() external view returns (PriceData memory) {
        return priceData[priceData.length - 1];
    }

    function getValueCount() external view returns (uint256) {
        return priceData.length;
    }
}