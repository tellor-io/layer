// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import "../interfaces/IERC20.sol";

contract AutopayMock {
    IERC20 public token;
    mapping(address => uint256) public tipsByAddress;

    event TIP(bytes32 _id, bytes _data);

    constructor(address _token) {
        token = IERC20(_token);
    }

    function tip(
        bytes32 _queryId,
        uint256 _amount,
        bytes calldata _queryData
    ) external {
        token.transferFrom(msg.sender, address(this), _amount);
        tipsByAddress[msg.sender] += _amount;
        emit TIP(_queryId, _queryData);
    }

    function getTipsByAddress(address _user) public view returns (uint256) {
        return tipsByAddress[_user];
    }
}
