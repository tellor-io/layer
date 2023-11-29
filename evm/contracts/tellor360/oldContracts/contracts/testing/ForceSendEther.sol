// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

contract ForceSendEther {
    function forceSendEther(address payable _beneficiary) public payable {
        selfdestruct(_beneficiary);
    }
    
    receive() external payable {}
}