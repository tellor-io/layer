// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import "../bridge/LayerLightClientBridge.sol";
import "../interfaces/IERC20.sol";
import "../interfaces/ITellorFlex.sol";

interface ITellorMaster {
    function mintToOracle() external;
}

contract TellorLayerTransition {
    IERC20 public token;
    address public tokenBridge;
    address public dataBridge;
    ITellorFlex public tellorFlex;

    constructor(address _token, address _tokenBridge, address _dataBridge, address _tellorFlex) {
        token = IERC20(_token);
        tokenBridge = _tokenBridge;
        dataBridge = _dataBridge;
        tellorFlex = ITellorFlex(_tellorFlex);
    }

    // needed for "mintToOracle" function
    function addStakingRewards(uint256 _amount) external {
        token.transferFrom(msg.sender, address(this), _amount);
    }

    function transferToTokenBridge() external {
        ITellorMaster(address(token)).mintToOracle();
        token.transfer(tokenBridge, token.balanceOf(address(this)));
    }

    // forward to tellor360:
    function getDataBefore(
        bytes32 _queryId,
        uint256 _timestamp
    )
        external
        view
        returns (
            bool _ifRetrieve,
            bytes memory _value,
            uint256 _timestampRetrieved
        ) {
            return tellorFlex.getDataBefore(_queryId, _timestamp);
        }

    function getIndexForDataBefore(
        bytes32 _queryId,
        uint256 _timestamp
    ) external view returns (bool _found, uint256 _index) {
        return tellorFlex.getIndexForDataBefore(_queryId, _timestamp);
    }

    function getNewValueCountbyQueryId(
        bytes32 _queryId
    ) external view returns (uint256) {
        return tellorFlex.getNewValueCountbyQueryId(_queryId);
    }

    function getReporterByTimestamp(
        bytes32 _queryId,
        uint256 _timestamp
    ) external view returns (address) {
        return tellorFlex.getReporterByTimestamp(_queryId, _timestamp);
    }

    function getTimestampbyQueryIdandIndex(
        bytes32 _queryId,
        uint256 _index
    ) external view returns (uint256) {
        return tellorFlex.getTimestampbyQueryIdandIndex(_queryId, _index);
    }

    function getTimeOfLastNewValue() external view returns (uint256) {
        // note: should parachute use old flex, or something else?
        return tellorFlex.getTimeOfLastNewValue();
    }

    function isInDispute(
        bytes32 _queryId,
        uint256 _timestamp
    ) external view returns (bool) {
        return tellorFlex.isInDispute(_queryId, _timestamp);
    }

    function verify() external pure returns (uint256) {
        return 9999;
    }
}