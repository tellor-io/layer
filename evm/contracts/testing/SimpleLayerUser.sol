// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import "../interfaces/IBlobstreamO.sol";

contract SimpleLayerUser {
    IBlobstreamO public blobstreamO;
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

    constructor(address _blobstreamO, bytes32 _queryId) {
        blobstreamO = IBlobstreamO(_blobstreamO);
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
        blobstreamO.verifyOracleData(_attestData, _currentValidatorSet, _sigs);
        require(block.timestamp - _attestData.report.timestamp < 6 hours, "data too old");
        require(block.timestamp - _attestData.attestationTimestamp < 10 minutes, "attestation too old");
        if(_attestData.report.aggregatePower < blobstreamO.powerThreshold()) {
            require(_attestData.report.aggregatePower > blobstreamO.powerThreshold() / 2, "optimistic power threshold not met");
            require(_attestData.attestationTimestamp - _attestData.report.timestamp >= 15 minutes, "dispute period not passed. request new attestations");
            require(block.timestamp - _attestData.report.nextTimestamp < 15 minutes, "more recent optimistic report available");
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

    function updateOracleData2(
        OracleAttestationData calldata _attestData,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs,
        uint256 _userTriggerTimestamp,
        uint256 _beginRelayTimestamp
    ) external {
        require(_attestData.queryId == queryId, "Invalid queryId");
        blobstreamO.verifyOracleData(_attestData, _currentValidatorSet, _sigs);
        require(block.timestamp - _attestData.report.timestamp < 6 hours, "data too old");
        require(block.timestamp - _attestData.attestationTimestamp < 10 minutes, "attestation too old");
        if (priceData.length > 0) {
            require(_attestData.report.timestamp > priceData[priceData.length - 1].timestamp, "report timestamp must increase");
        }
        // check if using data before dispute period
        if (block.timestamp - _attestData.report.timestamp > 15 minutes) {
            // using optimistic data with power below consensus threshold
            if(_attestData.report.aggregatePower < blobstreamO.powerThreshold()) {
                require(_attestData.report.aggregatePower > blobstreamO.powerThreshold() / 2, "optimistic power threshold not met");
                require(_attestData.attestationTimestamp - _attestData.report.timestamp >= 15 minutes, "dispute period not passed. request new attestations");
                require(block.timestamp - _attestData.report.nextTimestamp < 15 minutes, "more recent optimistic report available");
            } else {
                require(_attestData.report.nextTimestamp == 0, "should be no newer timestamp");
            }
            require(block.timestamp - _attestData.report.nextTimestamp < 15 minutes, "newer optimistic report available");
        } else {
            // using consensus data
            require(_attestData.report.nextTimestamp == 0, "should be no newer timestamp");
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

    function updateOracleData3(
        OracleAttestationData calldata _attestData,
        Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs,
        uint256 _userTriggerTimestamp,
        uint256 _beginRelayTimestamp
    ) external {
        require(_attestData.queryId == queryId, "Invalid queryId");
        blobstreamO.verifyOracleData(_attestData, _currentValidatorSet, _sigs);
        require(block.timestamp - _attestData.report.timestamp < 6 hours, "data too old");
        require(block.timestamp - _attestData.attestationTimestamp < 10 minutes, "attestation too old");
        if (priceData.length > 0) {
            require(_attestData.report.timestamp > priceData[priceData.length - 1].timestamp, "report timestamp must increase");
        }
        if (_attestData.report.timestamp != _attestData.report.lastConsensusTimestamp) {
            require(_attestData.report.timestamp > _attestData.report.lastConsensusTimestamp, "more recent consensus data available");
            require(_attestData.attestationTimestamp - _attestData.report.timestamp >= 15 minutes, "dispute period not passed. request new attestations");
            require(block.timestamp - _attestData.report.nextTimestamp < 15 minutes, "more recent optimistic report available");
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