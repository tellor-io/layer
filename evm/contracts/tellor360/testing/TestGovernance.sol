// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import { Governance } from "polygongovernance/contracts/Governance.sol";

contract TestGovernance is Governance {
    constructor(address payable _tellor, address _teamMultisig) Governance(_tellor, _teamMultisig) {}

    fallback() external payable {}
    receive() external payable {}
}