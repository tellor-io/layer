// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

contract MedianOracle {
    uint256 public payload_;

    function pushReport(uint256 payload) external {
        payload_ = payload;
    }
}
