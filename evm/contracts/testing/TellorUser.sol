// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// import "../usingtellor/UsingTellor.sol";

// contract TellorUser is UsingTellor {
//     uint256 public ethPrice;
//     uint256 public ethPriceTimestamp;
//     BlobstreamO public bridge;

//     constructor(address _bridge) UsingTellor(_bridge) {
//         bridge = BlobstreamO(_bridge);
//     }

//     function updateEthPrice(
//         OracleAttestationData calldata _attest,
//         Validator[] calldata _currentValidatorSet,
//         Signature[] calldata _sigs
//     ) public {
//         require(isCurrentConsensusValue(_attest, _currentValidatorSet, _sigs), "Invalid attestation");
//         ethPrice = abi.decode(_attest.report.value, (uint256));
//         ethPriceTimestamp = _attest.report.timestamp;
//     }
// }

