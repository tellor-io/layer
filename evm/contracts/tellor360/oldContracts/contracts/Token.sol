// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import "./tellor3/TellorStorage.sol";
import "./TellorVars.sol";
import "./interfaces/IGovernance.sol";

/**
 @author Tellor Inc.
 @title Token
 @dev Contains the methods related to transfers and ERC20, its storage
 * and hashes of tellor variables that are used to save gas on transactions.
*/
contract Token is TellorStorage, TellorVars {
    // Events
    event Approval(
        address indexed _owner,
        address indexed _spender,
        uint256 _value
    ); // ERC20 Approval event
    event Transfer(address indexed _from, address indexed _to, uint256 _value); // ERC20 Transfer Event

    // Functions
    /**
     * @dev Getter function for remaining spender balance
     * @param _user address of party with the balance
     * @param _spender address of spender of parties said balance
     * @return uint256 Returns the remaining allowance of tokens granted to the _spender from the _user
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
     * and removing the staked amount from their balance if they are staked
     * @param _user address of user
     * @param _amount to check if the user can spend
     * @return bool true if they are allowed to spend the amount being checked
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
            // Subtracts the stakeAmount from balance if the _user is staked
            return (balanceOf(_user) - uints[_STAKE_AMOUNT] >= _amount);
        }
        return (balanceOf(_user) >= _amount); // Else, check if balance is greater than amount they want to spend
    }

    /**
     * @dev This function approves a _spender an _amount of tokens to use
     * @param _spender address
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
     * @dev This function approves a transfer of _amount tokens from _from to _to
     * @param _from is the address the tokens will be transferred from
     * @param _to is the address the tokens will be transferred to
     * @param _amount is the number of tokens to transfer
     * @return bool true if spender approved successfully
     */
    function approveAndTransferFrom(
        address _from,
        address _to,
        uint256 _amount
    ) external returns (bool) {
        require(
            (IGovernance(addresses[_GOVERNANCE_CONTRACT])
                .isApprovedGovernanceContract(msg.sender) ||
                msg.sender == addresses[_TREASURY_CONTRACT] ||
                msg.sender == addresses[_ORACLE_CONTRACT]),
            "Only the Governance, Treasury, or Oracle Contract can approve and transfer tokens"
        );
        _doTransfer(_from, _to, _amount);
        return true;
    }

    /**
     * @dev Gets balance of owner specified
     * @param _user is the owner address used to look up the balance
     * @return uint256 Returns the balance associated with the passed in _user
     */
    function balanceOf(address _user) public view returns (uint256) {
        return balanceOfAt(_user, block.number);
    }

    /**
     * @dev Queries the balance of _user at a specific _blockNumber
     * @param _user The address from which the balance will be retrieved
     * @param _blockNumber The block number when the balance is queried
     * @return uint256 The balance at _blockNumber specified
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
     * @dev Burns an amount of tokens
     * @param _amount is the amount of tokens to burn
     */
    function burn(uint256 _amount) external {
        _doBurn(msg.sender, _amount);
    }

    /**
     * @dev Allows for a transfer of tokens to _to
     * @param _to The address to send tokens to
     * @param _amount The amount of tokens to send
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
     * @param _from The address holding the tokens being transferred
     * @param _to The address of the recipient
     * @param _amount The amount of tokens to be transferred
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

    // Internal
    /**
     * @dev Helps burn TRB Tokens
     * @param _from is the address to burn or remove TRB amount
     * @param _amount is the amount of TRB to burn
     */
    function _doBurn(address _from, uint256 _amount) internal {
        // Ensure that amount of balance are valid
        if (_amount == 0) return;
        require(
            allowedToTrade(_from, _amount),
            "Should have sufficient balance to trade"
        );
        uint128 _previousBalance = uint128(balanceOf(_from));
        uint128 _sizedAmount = uint128(_amount);
        // Update total supply and balance of _from
        _updateBalanceAtNow(_from, _previousBalance - _sizedAmount);
        uints[_TOTAL_SUPPLY] -= _amount;
    }

    /**
     * @dev Helps mint new TRB
     * @param _to is the address to send minted amount to
     * @param _amount is the amount of TRB to send
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
     * @dev Completes transfers by updating the balances on the current block number
     * and ensuring the amount does not contain tokens staked for reporting
     * @param _from address to transfer from
     * @param _to address to transfer to
     * @param _amount to transfer
     */
    function _doTransfer(
        address _from,
        address _to,
        uint256 _amount
    ) internal {
        // Ensure user has a correct balance and to address
        require(_amount != 0, "Tried to send non-positive amount");
        require(_to != address(0), "Receiver is 0 address");
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
     * @dev Updates balance checkpoint for from and to on the current block number via doTransfer
     * @param _user is the address whose balance is updated
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
