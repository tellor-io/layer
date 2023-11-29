// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import "../Governance.sol";

contract TestGovernance is Governance {
    event Received(address, uint256);

    receive() external payable {
        emit Received(msg.sender, msg.value);
    }

    //allows the contract to recieve funds for gas via harhdat-impersonate account
    fallback() external payable {
        emit Received(msg.sender, msg.value);
    }
}
