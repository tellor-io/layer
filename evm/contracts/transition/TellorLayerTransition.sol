// SPDX-License-Identifier: MIT

import "../bridge/LayerLightClientBridge.sol";
import "../interfaces/IERC20.sol";

pragma solidity 0.8.3;

contract TellorLayerTransition {
    address public token;
    address public tokenBridge;
    address public dataBridge;

    // oracle store
    mapping(bytes32 => mapping(uint256 => address)) public reporterByTimestamp;
    mapping(bytes32 => uint256[]) public timestamps;
    mapping(bytes32 => mapping(uint256 => bytes)) public values; //queryId -> timestamp -> value

    values[_queryId][block.timestamp] = _value;
    timestamps[_queryId].push(block.timestamp);
    reporterByTimestamp[_queryId][block.timestamp] = msg.sender;

    constructor(address _token, address _tokenBridge, address _dataBridge) {
        tokenBridge = _tokenBridge;
        dataBridge = _dataBridge;
    }

    function verifyAndStore(
        uint256 _blockHeight,
        LayerLightClientBridge.Report calldata _report,
        uint256 _oracleHeight,
        LayerLightClientBridge.IAVLMerklePath[] calldata _merklePaths)
        external {
            
        }
}