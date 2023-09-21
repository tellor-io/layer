// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import { IBridgeProxy } from "../interfaces/IBridgeProxy.sol";

contract BridgeProxyMock is IBridgeProxy {
    address public implementation;

    constructor(address _implementation) {
        implementation = _implementation;
    }

    function updateImplementation(address _newImplementation) external override {
        implementation = _newImplementation;
    }
}

