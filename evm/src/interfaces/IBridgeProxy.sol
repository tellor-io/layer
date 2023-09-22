// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

interface IBridgeProxy {
    function paused() external returns (bool);
    function updateImplementation(address _newImplementation) external;
    function pauseBridge() external;
    function unpauseBridge() external;
}