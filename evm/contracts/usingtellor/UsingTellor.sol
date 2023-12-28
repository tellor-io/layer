// // SPDX-License-Identifier: MIT
// pragma solidity >=0.8.0;

// import "../bridge/BlobstreamO.sol";

// contract UsingTellor {
//     BlobstreamO public bridge;
//     uint256 public lastViewedBlockTimestamp;
//     uint256 public constant MAX_ATTESTATION_AGE = 5 minutes;
//     mapping(bytes32 => uint256) public lastViewedReportTimestamp;

//     constructor(address _blobstreamO) {
//         bridge = BlobstreamO(_blobstreamO);
//     }

//     // // getdataoptimistic
//     // function getDataBefore(
//     //     BlobstreamO.OracleAttestationData calldata _attest,
//     //     BlobstreamO.Validator[] calldata _currentValidatorSet,
//     //     Signature[] calldata _sigs,
//     //     uint256 _beforeTimestamp,
//     //     uint256 _afterTimestamp
//     // ) public returns(bytes, uint256){
//     //     require(bridge.verifyOracleData(_attest, _currentValidatorSet, _sigs), "Invalid signature");
//     //     require(_attest.timestamp < _beforeTimestamp, "Timestamp must be before _beforeTimestamp");
//     //     require(_attest.report.timestamp >= lastViewedReportTimestamp[_attest.dataRoot], "Timestamp must be after last viewed timestamp");
//     //     lastViewedReportTimestamp[_attest.dataRoot] = _attest.timestamp;
//     //     return (_attest.)
//     // }

//     // getDataWithFallback
//     function getDataWithFallback(uint256 _optimisticBuffer)

//     // getData - consensus
//     function getConsensusData(
//         OracleAttestationData calldata _attest,
//         Validator[] calldata _currentValidatorSet,
//         Signature[] calldata _sigs
//     ) public returns(bytes memory, uint256){
//         require(bridge.verifyConsensusOracleData(_attest, _currentValidatorSet, _sigs), "Invalid attestation");
//         require(block.timestamp - _attest.blockTimestamp <= MAX_ATTESTATION_AGE, "Attestation is too old");
//         if(_attest.report.timestamp != lastViewedReportTimestamp[_attest.queryId]) {
//             require(_attest.report.timestamp > lastViewedReportTimestamp[_attest.queryId], "Timestamp must be greater than or equal to last viewed timestamp");
//             lastViewedReportTimestamp[_attest.queryId] = _attest.report.timestamp;
//         }
//         // require(_attest.timestamp >= lastViewedReportTimestamp[_attest.dataRoot], "Timestamp must be after last viewed timestamp");
//         return (_attest.report.value, _attest.report.timestamp);
//     }

// }