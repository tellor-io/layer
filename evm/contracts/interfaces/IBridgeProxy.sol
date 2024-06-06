// SPDX-License-Identifier: MIT
pragma solidity 0.8.22;

interface IBridgeProxy {
    function pauseBridge() external;
    function paused() external returns (bool);
    function unpauseBridge() external;
    function updateImplementation(address _newImplementation) external;
}