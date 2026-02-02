// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import { TokenBridgeV2 } from "../token-bridge/TokenBridgeV2.sol";

/// @title TokenBridgeV2
/// @dev allows us to test internal limit refreshes externally
contract TestTokenBridgeV2 is TokenBridgeV2 {
    constructor(address _token, address _blobstream, address _tellorFlex) TokenBridgeV2(_token, _blobstream, _tellorFlex) {}

    /// @notice refreshes the deposit limit every 12 hours so no one can spam layer with new tokens
    function refreshDepositLimit(uint256 _amount) external returns (uint256) {
        return _refreshDepositLimit(_amount);
    }

    /// @notice refreshes the withdraw limit every 12 hours so no one can spam layer with new tokens
    function refreshWithdrawLimit(uint256 _amount) external returns (uint256) {
        return _refreshWithdrawLimit(_amount);
    }
}

