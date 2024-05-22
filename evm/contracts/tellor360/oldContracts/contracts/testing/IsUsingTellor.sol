// SPDX-License-Identifier: MIT

pragma solidity 0.8.3;

import "./UsingTellor.sol";

contract IsUsingTellor is UsingTellor {
    constructor(address payable _tellor) UsingTellor(_tellor) {}
}
