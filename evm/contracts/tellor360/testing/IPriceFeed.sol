// SPDX-License-Identifier: MIT

pragma solidity 0.8.3;

interface IPriceFeed {
    // --- Events ---
    event LastGoodPriceUpdated(uint _lastGoodPrice);
    // --- Function ---
    function fetchPrice() external returns (uint);
    function lastGoodPrice() external view returns(uint);
    function tellorCaller() external view returns(address);
}