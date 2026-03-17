// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import { TokenBridgeV2 } from "../token-bridge/TokenBridgeV2.sol";

/// @title TokenBridgeV2
/// @dev allows us to test internal limit refreshes externally
contract TestTokenBridgeV2 is TokenBridgeV2 {
    constructor(
        address _token,
        address _blobstream,
        address _tellorFlex,
        address _mainGuardian,
        address _subGuardian,
        uint256 _defaultRoleUpdateDelay
    ) TokenBridgeV2(_token, _blobstream, _tellorFlex, _mainGuardian, _subGuardian, _defaultRoleUpdateDelay) {}

    /// @notice refreshes the deposit limit every 12 hours so no one can spam layer with new tokens
    function refreshDepositLimit() external returns (uint256) {
        return _refreshDepositLimit();
    }

    /// @notice refreshes the withdraw limit every 12 hours so no one can spam layer with new tokens
    function refreshWithdrawLimit() external returns (uint256) {
        return _refreshWithdrawLimit();
    }
}
