// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import { IERC20 } from "../interfaces/IERC20.sol";

contract TokenBridge {
    IERC20 public token;

    mapping(bytes32 => bool) public receivePaid;

    event CrossChainSend(uint256 amount, uint256 fee, address recipient);
 
    constructor(address _token) {
        token = IERC20(_token);
    }

    function crossChainSend(uint256 _amount, uint256 _fee, address _recipient) external {
        require(_fee < _amount, "fee must be less than amount");
        token.transferFrom(msg.sender, address(this), _amount);
        if (_recipient == address(0)) {
            _recipient = msg.sender;
        }
        emit CrossChainSend(_amount, _fee, _recipient);
    }

    function crossChainReceive(bytes memory _proof) external {
        bytes32 _proofHash = keccak256(_proof);
        require(!receivePaid(_proofHash), "invalid proof");
        require(_verifyReceiveProof(_proof), "invalid proof");
        receivePaid[_proofHash] = true;
        token.transfer(_recipient, _amount);
    }

    function _verifyReceiveProof(bytes memory _proof) internal returns(bool) {
        // verify inclusion proof in the bridge contract
    }
}