// SPDX-License-Identifier: MIT
pragma solidity ^0.8.22;

import { TokenBridge } from "../token-bridge/TokenBridge.sol";

/// @title TokenBridge
/// @dev allows us to test deposit limit externally
contract TestTokenBridge is TokenBridge{

    constructor(address _token, address _blobstream, address _tellorFlex) TokenBridge(_token, _blobstream, _tellorFlex){
    }

    /// @notice refreshes the deposit limit every 12 hours so no one can spam layer with new tokens
    function refreshDepositLimit(uint256 _amount) external returns (uint256) {
        return _refreshDepositLimit(_amount);
    }
}
