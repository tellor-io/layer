// SPDX-License-Identifier: MIT
pragma solidity 0.8.22;

import "../interfaces/IBlobstreamO.sol";

contract SimpleLayerUser {
    IBlobstreamO public blobstreamO;
    PriceData[] public priceData;
    bytes32 public queryId;

    struct PriceData {
        uint256 price;
        uint256 timestamp;
        uint256 aggregatePower;
        uint256 previousTimestamp;
        uint256 nextTimestamp;
        uint256 relayTimestamp;
    }

    constructor(address _blobstreamO, bytes32 _queryId) {
        blobstreamO = IBlobstreamO(_blobstreamO);
        queryId = _queryId;
    }

    function updateOracleData(
        OracleAttestationData calldata _attestData,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs
    ) external {
        require(_attestData.queryId == queryId, "Invalid queryId");
        blobstreamO.verifyOracleData(_attestData, _currentValidatorSet, _sigs);
        uint256 _price = abi.decode(_attestData.report.value, (uint256));
        priceData.push(PriceData(
            _price, 
            _attestData.attestationTimestamp, 
            _attestData.report.aggregatePower, 
            _attestData.report.previousTimestamp, 
            _attestData.report.nextTimestamp,
            block.timestamp
            )
        );
    }

    function getCurrentPriceData() external view returns (PriceData memory) {
        return priceData[priceData.length - 1];
    }

    function getValueCount() external view returns (uint256) {
        return priceData.length;
    }
}