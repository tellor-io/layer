// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import "./oldContracts/contracts/TellorVars.sol";
import "./oldContracts/contracts/interfaces/IGovernance.sol";
import "./oldContracts/contracts/tellor3/TellorStorage.sol";

/**
 @author Tellor Inc.
 @title BaseToken
 @dev Contains the methods related to ERC20 transfers, allowance, and storage
*/
contract BaseToken is TellorStorage, TellorVars {
    // Events
    event Approval(
        address indexed _owner,
        address indexed _spender,
        uint256 _value
    ); // ERC20 Approval event
    event Transfer(address indexed _from, address indexed _to, uint256 _value); // ERC20 Transfer Event

    // Functions
    /**
     * @dev This function approves a _spender an _amount of tokens to use
     * @param _spender address receiving the allowance
     * @param _amount amount the spender is being approved for
     * @return bool true if spender approved successfully
     */
    function approve(address _spender, uint256 _amount)
        external
        returns (bool)
    {
        require(_spender != address(0), "ERC20: approve to the zero address");
        _allowances[msg.sender][_spender] = _amount;
        emit Approval(msg.sender, _spender, _amount);
        return true;
    }

    /**
     * @notice Allows tellor team to transfer stake of disputed TellorX reporter
     * NOTE: this does not affect TellorFlex stakes, only disputes during 360 transition period
     * @param _from the staker address holding the tokens being transferred
     * @param _to the address of the recipient
     */
    function teamTransferDisputedStake(address _from, address _to) external {
        require(
            msg.sender == addresses[_OWNER],
            "only owner can transfer disputed staked"
        );
        require(
            stakerDetails[_from].currentStatus == 3,
            "_from address not disputed"
        );
        stakerDetails[_from].currentStatus = 0;
        _doTransfer(_from, _to, uints[_STAKE_AMOUNT]);
    }

    /**
     * @dev Transfers _amount tokens from message sender to _to address
     * @param _to token recipient
     * @param _amount amount of tokens to send
     * @return success whether the transfer was successful
     */
    function transfer(address _to, uint256 _amount)
        external
        returns (bool success)
    {
        _doTransfer(msg.sender, _to, _amount);
        return true;
    }

    /**
     * @notice Send _amount tokens to _to from _from on the condition it
     * is approved by _from
     * @param _from the address holding the tokens being transferred
     * @param _to the address of the recipient
     * @param _amount the amount of tokens to be transferred
     * @return success whether the transfer was successful
     */
    function transferFrom(
        address _from,
        address _to,
        uint256 _amount
    ) external returns (bool success) {
        require(
            _allowances[_from][msg.sender] >= _amount,
            "Allowance is wrong"
        );
        _allowances[_from][msg.sender] -= _amount;
        _doTransfer(_from, _to, _amount);
        return true;
    }

    // Getters
    /**
     * @dev Getter function for remaining spender balance
     * @param _user address of party with the balance
     * @param _spender address of spender of said user's balance
     * @return uint256 the remaining allowance of tokens granted to the _spender from the _user
     */
    function allowance(address _user, address _spender)
        external
        view
        returns (uint256)
    {
        return _allowances[_user][_spender];
    }

    /**
     * @dev This function returns whether or not a given user is allowed to trade a given amount
     * and removes the staked amount if they are staked in TellorX and disputed
     * @param _user address of user
     * @param _amount to check if the user can spend
     * @return bool true if they are allowed to spend the amount being checked
     */
    function allowedToTrade(address _user, uint256 _amount)
        public
        view
        returns (bool)
    {
        if (stakerDetails[_user].currentStatus == 3) {
            // Subtracts the stakeAmount from balance if the _user is staked and disputed in TellorX
            return (balanceOf(_user) - uints[_STAKE_AMOUNT] >= _amount);
        }
        return (balanceOf(_user) >= _amount); // Else, check if balance is greater than amount they want to spend
    }

    /**
     * @dev Gets the balance of a given address
     * @param _user the address whose balance to look up
     * @return uint256 the balance of the given _user address
     */
    function balanceOf(address _user) public view returns (uint256) {
        return balanceOfAt(_user, block.number);
    }

    /**
     * @dev Gets the historic balance of a given _user address at a specific _blockNumber
     * @param _user the address whose balance to look up
     * @param _blockNumber the block number at which the balance is queried
     * @return uint256 the balance of the _user address at the _blockNumber specified
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
            uint256 _min = 0;
            uint256 _max = checkpoints.length - 2;
            while (_max > _min) {
                uint256 _mid = (_max + _min + 1) / 2;
                if (checkpoints[_mid].fromBlock == _blockNumber) {
                    return checkpoints[_mid].value;
                } else if (checkpoints[_mid].fromBlock < _blockNumber) {
                    _min = _mid;
                } else {
                    _max = _mid - 1;
                }
            }
            return checkpoints[_min].value;
        }
    }

    /**
     * @dev Allows users to access the number of decimals
     */
    function decimals() external pure returns (uint8) {
        return 18;
    }

    /**
     * @dev Allows users to access the token's name
     */
    function name() external pure returns (string memory) {
        return "Tellor Tributes";
    }

    /**
     * @dev Allows users to access the token's symbol
     */
    function symbol() external pure returns (string memory) {
        return "TRB";
    }

    /**
     * @dev Getter for the total_supply of tokens
     * @return uint256 total supply
     */
    function totalSupply() external view returns (uint256) {
        return uints[_TOTAL_SUPPLY];
    }

    // Internal functions
    /**
     * @dev Helps mint new TRB
     * @param _to is the address to send minted amount to
     * @param _amount is the amount of TRB to mint and send
     */
    function _doMint(address _to, uint256 _amount) internal {
        // Ensure to address and mint amount are valid
        require(_amount != 0, "Tried to mint non-positive amount");
        require(_to != address(0), "Receiver is 0 address");
        uint128 _previousBalance = uint128(balanceOf(_to));
        uint128 _sizedAmount = uint128(_amount);
        // Update total supply and balance of _to address
        uints[_TOTAL_SUPPLY] += _amount;
        _updateBalanceAtNow(_to, _previousBalance + _sizedAmount);
        emit Transfer(address(0), _to, _amount);
    }

    /**
     * @dev Completes transfers by updating the balances at the current block number
     * and ensuring the amount does not contain tokens locked for tellorX disputes
     * @param _from address to transfer from
     * @param _to address to transfer to
     * @param _amount amount of tokens to transfer
     */
    function _doTransfer(
        address _from,
        address _to,
        uint256 _amount
    ) internal {
        if (_amount == 0) {
            return;
        }
        require(
            allowedToTrade(_from, _amount),
            "Should have sufficient balance to trade"
        );
        // Update balance of _from address
        uint128 _previousBalance = uint128(balanceOf(_from));
        uint128 _sizedAmount = uint128(_amount);
        _updateBalanceAtNow(_from, _previousBalance - _sizedAmount);
        // Update balance of _to address
        _previousBalance = uint128(balanceOf(_to));
        _updateBalanceAtNow(_to, _previousBalance + _sizedAmount);
        emit Transfer(_from, _to, _amount);
    }

    /**
     * @dev Updates balance checkpoint _amount for a given _user address at the current block number
     * @param _user is the address whose balance to update
     * @param _value is the new balance
     */
    function _updateBalanceAtNow(address _user, uint128 _value) internal {
        Checkpoint[] storage checkpoints = balances[_user];
        // Checks if no checkpoints exist, or if checkpoint block is not current block
        if (
            checkpoints.length == 0 ||
            checkpoints[checkpoints.length - 1].fromBlock != block.number
        ) {
            // If yes, push a new checkpoint into the array
            checkpoints.push(
                TellorStorage.Checkpoint({
                    fromBlock: uint128(block.number),
                    value: _value
                })
            );
        } else {
            // Else, update old checkpoint
            TellorStorage.Checkpoint storage oldCheckPoint = checkpoints[
                checkpoints.length - 1
            ];
            oldCheckPoint.value = _value;
        }
    }
}
