// SPDX-License-Identifier: MIT
pragma solidity 0.7.4;

import "../Tellor.sol";

contract TellorTest is Tellor {
    uint256 version = 3000;

    /*Functions*/
    /**
     * @dev Constructor to set extension address
     * @param _ext Extension address
    */
    constructor(address _ext) Tellor(_ext){
    }

    /*This is a cheat for demo purposes, is not on main Tellor*/
    function setBalanceTest(address _address, uint256 _amount) public {
        uints[_TOTAL_SUPPLY] += _amount;
        TellorTransfer._updateBalanceAtNow(_address, uint128(_amount));
    }

    /*This function uses all the functionality of submitMiningSolution, but bypasses verifyNonce*/
    function testSubmitMiningSolution(
        string calldata _nonce,
        uint256[5] calldata _requestId,
        uint256[5] calldata _value
    ) external {
        bytes32 _hashMsgSender = keccak256(abi.encode(msg.sender));
        require(
            uints[_hashMsgSender] == 0 ||
                block.timestamp - uints[_hashMsgSender] > 15 minutes,
            "Miner can only win rewards once per 15 min"
        );
        _submitMiningSolution(_nonce, _requestId, _value);
    }

    /*allows manually setting the difficulty in tests*/
    function manuallySetDifficulty(uint256 _diff) public {
        uints[_DIFFICULTY] = _diff;
    }

    function bumpVersion() external {
        version++;
    }

    function verify() external view override returns (uint256) {
        return version;
    }
}
