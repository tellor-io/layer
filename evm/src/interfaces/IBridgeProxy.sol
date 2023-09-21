// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

interface IBridgeProxy {
    function updateImplementation(address _newImplementation) external;
}