// SPDX-License-Identifier: MIT
pragma solidity 0.7.4;

import "./SafeMath.sol";
import "./TellorStorage.sol";
import "./TellorVariables.sol";

/**
 @author Tellor Inc.
 @title TellorTransfer
 @dev Contains the methods related to transfers and ERC20, its storage and hashes of tellor variables
 * that are used to save gas on transactions.
*/
contract TellorTransfer is TellorStorage, TellorVariables {
    using SafeMath for uint256;

    /*Events*/
    event Approval(
        address indexed _owner,
        address indexed _spender,
        uint256 _value
    ); //ERC20 Approval event
    event Transfer(address indexed _from, address indexed _to, uint256 _value); //ERC20 Transfer Event

    /*Functions*/
    /**
     * @dev Getter function for remaining spender balance
     * @param _user address of party with the balance
     * @param _spender address of spender of parties said balance
     * @return Returns the remaining allowance of tokens granted to the _spender from the _user
    */
    function allowance(address _user, address _spender)
        public
        view
        returns (uint256)
    {
        return _allowances[_user][_spender];
    }

    /**
     * @dev This function returns whether or not a given user is allowed to trade a given amount
     * and removing the staked amount from their balance if they are staked
     * @param _user address of user
     * @param _amount to check if the user can spend
     * @return true if they are allowed to spend the amount being checked
    */
    function allowedToTrade(address _user, uint256 _amount)
        public
        view
        returns (bool)
    {
        if (
            stakerDetails[_user].currentStatus != 0 &&
            stakerDetails[_user].currentStatus < 5
        ) {
            //Subtracts the stakeAmount from balance if the _user is staked
            if (balanceOf(_user).sub(uints[_STAKE_AMOUNT]) >= _amount) {
                return true;
            }
            return false;
        }
        return (balanceOf(_user) >= _amount);
    }

    /**
     * @dev This function approves a _spender an _amount of tokens to use
     * @param _spender address
     * @param _amount amount the spender is being approved for
     * @return true if spender approved successfully
     */
    function approve(address _spender, uint256 _amount) public returns (bool) {
        require(_spender != address(0), "ERC20: approve to the zero address");

        _allowances[msg.sender][_spender] = _amount;
        emit Approval(msg.sender, _spender, _amount);
        return true;
    }

    /**
     * @dev Gets balance of owner specified
     * @param _user is the owner address used to look up the balance
     * @return Returns the balance associated with the passed in _user
     */
    function balanceOf(address _user) public view returns (uint256) {
        return balanceOfAt(_user, block.number);
    }

    /**
     * @dev Queries the balance of _user at a specific _blockNumber
     * @param _user The address from which the balance will be retrieved
     * @param _blockNumber The block number when the balance is queried
     * @return The balance at _blockNumber specified
     */
    function balanceOfAt(address _user, uint256 _blockNumber)
        public
        view
        returns (uint256)
    {
        TellorStorage.Checkpoint[] storage checkpoints = balances[_user];
        if (
            checkpoints.length == 0 || checkpoints[0].fromBlock > _blockNumber
        ) {
            return 0;
        } else {
            if (_blockNumber >= checkpoints[checkpoints.length - 1].fromBlock)
                return checkpoints[checkpoints.length - 1].value;
            // Binary search of the value in the array
            uint256 min = 0;
            uint256 max = checkpoints.length - 2;
            while (max > min) {
                uint256 mid = (max + min + 1) / 2;
                if (checkpoints[mid].fromBlock == _blockNumber) {
                    return checkpoints[mid].value;
                } else if (checkpoints[mid].fromBlock < _blockNumber) {
                    min = mid;
                } else {
                    max = mid - 1;
                }
            }
            return checkpoints[min].value;
        }
    }

    /**
     * @dev Allows for a transfer of tokens to _to
     * @param _to The address to send tokens to
     * @param _amount The amount of tokens to send
     */
    function transfer(address _to, uint256 _amount)
        public
        returns (bool success)
    {
        _doTransfer(msg.sender, _to, _amount);
        return true;
    }

    /**
     * @notice Send _amount tokens to _to from _from on the condition it
     * is approved by _from
     * @param _from The address holding the tokens being transferred
     * @param _to The address of the recipient
     * @param _amount The amount of tokens to be transferred
    */
    function transferFrom(
        address _from,
        address _to,
        uint256 _amount
    ) public returns (bool success) {
        require(
            _allowances[_from][msg.sender] >= _amount,
            "Allowance is wrong"
        );
        _allowances[_from][msg.sender] -= _amount;
        _doTransfer(_from, _to, _amount);
        return true;
    }

    /*Internal Functions*/
    /**
     * @dev Completes transfers by updating the balances on the current block number
     * and ensuring the amount does not contain tokens staked for mining
     * @param _from address to transfer from
     * @param _to address to transfer to
     * @param _amount to transfer
    */
    function _doTransfer(
        address _from,
        address _to,
        uint256 _amount
    ) internal {
        require(_amount != 0, "Tried to send non-positive amount");
        require(_to != address(0), "Receiver is 0 address");
        require(
            allowedToTrade(_from, _amount),
            "Should have sufficient balance to trade"
        );
        uint128 previousBalance = uint128(balanceOf(_from));
        uint128 _sizedAmount  = uint128(_amount);
        _updateBalanceAtNow(_from, previousBalance - _sizedAmount);
        previousBalance = uint128(balanceOf(_to));
        require(
            previousBalance + _sizedAmount >= previousBalance,
            "Overflow happened"
        ); // Check for overflow
        _updateBalanceAtNow(_to, previousBalance + _sizedAmount);
        emit Transfer(_from, _to, _amount);
    }

    /**
     * @dev Helps swap the old Tellor contract Tokens to the new one
     * @param _to is the adress to send minted amount to
     * @param _amount is the amount of TRB to send
    */
    function _doMint(address _to, uint256 _amount) internal {
        require(_amount != 0, "Tried to mint non-positive amount");
        require(_to != address(0), "Receiver is 0 address");
        uint128 previousBalance = uint128(balanceOf(_to));
        uint128 _sizedAmount  = uint128(_amount);
        require(
            previousBalance + _sizedAmount >= previousBalance,
            "Overflow happened"
        );
        uint256 previousSupply = uints[_TOTAL_SUPPLY];
        require(
            previousSupply + _amount >= previousSupply,
            "Overflow happened"
        );
        uints[_TOTAL_SUPPLY] += _amount;
        _updateBalanceAtNow(_to, previousBalance + _sizedAmount);
        emit Transfer(address(0), _to, _amount);
    }

    /**
     * @dev Helps burn TRB Tokens
     * @param _from is the adress to burn or remove TRB amount
     * @param _amount is the amount of TRB to burn
     */
    function _doBurn(address _from, uint256 _amount) internal {
        if (_amount == 0) return;
        require(
            allowedToTrade(_from, _amount),
            "Should have sufficient balance to trade"
        );
        uint128 previousBalance = uint128(balanceOf(_from));
        uint128 _sizedAmount  = uint128(_amount);
        require(
            previousBalance - _sizedAmount <= previousBalance,
            "Overflow happened"
        );
        uint256 previousSupply = uints[_TOTAL_SUPPLY];
        require(
            previousSupply - _amount <= previousSupply,
            "Overflow happened"
        );
        _updateBalanceAtNow(_from, previousBalance - _sizedAmount);
        uints[_TOTAL_SUPPLY] -= _amount;
    }

    /**
     * @dev Updates balance for from and to on the current block number via doTransfer
     * @param _value is the new balance
     */
    function _updateBalanceAtNow(address _user, uint128 _value) internal {
        Checkpoint[] storage checkpoints = balances[_user];
        if (
            checkpoints.length == 0 ||
            checkpoints[checkpoints.length - 1].fromBlock != block.number
        ) {
            checkpoints.push(
                TellorStorage.Checkpoint({
                    fromBlock: uint128(block.number),
                    value: _value
                })
            );
        } else {
            TellorStorage.Checkpoint storage oldCheckPoint =
                checkpoints[checkpoints.length - 1];
            oldCheckPoint.value = _value;
        }
    }
}
