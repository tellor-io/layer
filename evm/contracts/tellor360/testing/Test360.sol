// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import "../Tellor360.sol";

contract Test360 is Tellor360 {
    event Received(address, uint256);

    constructor(address _flexAddress) Tellor360(_flexAddress) {}

    receive() external payable {
        emit Received(msg.sender, msg.value);
    }

    function changeAddressVar(bytes32 _id, address _addy) external {
        addresses[_id] = _addy;
    }

    function sliceUintTest(bytes memory bs) external pure returns (uint256) {
        return _sliceUint(bs);
    }

    function isValid(address _contract) external returns(bool) {
        return _isValid(_contract);
    }

    function doMintTest(address _to, uint256 _amount) external {
        _doMint(_to, _amount);
    }
}
