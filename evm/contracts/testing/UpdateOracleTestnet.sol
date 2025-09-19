// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import { TellorStorage } from "../tellor360/oldContracts/contracts/tellor3/TellorStorage.sol";
import { TellorVars } from "../tellor360/oldContracts/contracts/TellorVars.sol";

contract UpdateOracleTestnet is TellorStorage, TellorVars {
    address public immutable newTokenBridge;

    constructor(address _newTokenBridge) {
        newTokenBridge = _newTokenBridge;
    }
    function init() external {
        addresses[_ORACLE_CONTRACT] = newTokenBridge;
    }
}