// SPDX-LicenseIdentifier: MIT
pragma solidity 0.8.0;

import "../bridge/BlobstreamO.sol";

contract UsingTellor {
    BlobstreamO public bridge;
    mapping(bytes32 => uint256) public lastViewedReportTimestamp;
    uint256 public lastViewedBlockTimestamp;

    constructor(address _blobstreamO) {
        bridge = BlobstreamO(_blobstreamO);
    }

    function getDataBefore(
        BlobstreamO.OracleAttestationData calldata _attest,
        BlobstreamO.Validator[] calldata _currentValidatorSet,
        Signature[] calldata _sigs
        uint256 _beforeTimestamp;
    ) public returns(bytes, uint256){
        require(bridge.verifyOracleData(_attest, _currentValidatorSet, _sigs), "Invalid signature");
        require(_attest.timestamp < _beforeTimestamp, "Timestamp must be before _beforeTimestamp");
        require(_attest.timestamp > lastViewedReportTimestamp[_attest.dataRoot], "Timestamp must be after last viewed timestamp");
        lastViewedReportTimestamp[_attest.dataRoot] = _attest.timestamp;
        return (_attest.)
    }


}