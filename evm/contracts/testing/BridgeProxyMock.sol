// SPDX-License-Identifier: MIT
pragma solidity 0.8.22;

import { IBridgeProxy } from "../interfaces/IBridgeProxy.sol";

contract BridgeProxyMock is IBridgeProxy {
    address public implementation;
    bool public override paused;

    constructor(address _implementation) {
        implementation = _implementation;
    }

    function updateImplementation(address _newImplementation) external override {
        implementation = _newImplementation;
    }

    function pauseBridge() external override {
        paused = true;
    }

    function unpauseBridge() external override {
        paused = false;
    }
}

