// SPDX-License-Identifier: MIT

pragma solidity 0.8.3;

interface IPriceFeed {

    function lastGoodPrice() external view returns(uint);

    function tellorCaller() external view returns(address);

    // --- Events ---
    event LastGoodPriceUpdated(uint _lastGoodPrice);

    // --- Function ---
    function fetchPrice() external returns (uint);
}
