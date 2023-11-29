// SPDX-License-Identifier: MIT
pragma solidity >=0.7.4;

import "./SafeMath.sol";
import "./TellorGetters.sol";
import "./TellorVariables.sol";
import "./Utilities.sol";

/** 
 @author Tellor Inc.
 @title Extension
 @dev  This contract holds staking functions, tallyVotes and updateDisputeFee
 * Because of space limitations and will be consolidated in future iterations
**/
contract Extension is TellorGetters {
    using SafeMath for uint256;
    
    /*Events*/
    //emitted upon dispute tally
    event DisputeVoteTallied(
        uint256 indexed _disputeID,
        int256 _result,
        address indexed _reportedMiner,
        address _reportingParty,
        bool _passed
    );
    event StakeWithdrawn(address indexed _sender); //Emits when a staker is block.timestamp no longer staked
    event StakeWithdrawRequested(address indexed _sender); //Emits when a staker begins the 7 day withdraw period
    event NewStake(address indexed _sender); //Emits upon new staker
    event NewTellorAddress(address _newTellor);
    /*Functions*/
    /**
     * @dev This function allows miners to deposit their stake.
     */
    function depositStake() external{
        _newStake(msg.sender);
        updateMinDisputeFee();
    }

    /**
     * @dev This function allows stakers to request to withdraw their stake (no longer stake)
     * once they lock for withdraw(stakes.currentStatus = 2) they are locked for 7 days before they
     * can withdraw the deposit
     */
    function requestStakingWithdraw() external {
        StakeInfo storage stakes = stakerDetails[msg.sender];
        //Require that the miner is staked
        require(stakes.currentStatus == 1, "Miner is not staked");
        //Change the miner staked to locked to be withdrawStake
        stakes.currentStatus = 2;
        //Change the startDate to block.timestamp since the lock up period begins block.timestamp
        //and the miner can only withdraw 7 days later from block.timestamp(check the withdraw function)
        stakes.startDate = block.timestamp - (block.timestamp % 86400);
        //Reduce the staker count
        uints[_STAKE_COUNT] -= 1;
        //Update the minimum dispute fee that is based on the number of stakers
        updateMinDisputeFee();
        emit StakeWithdrawRequested(msg.sender);
    }

    /**
     * @dev tallies the votes and locks the stake disbursement(currentStatus = 4) if the vote passes
     * @param _disputeId is the dispute id
     */
    function tallyVotes(uint256 _disputeId) external {
        Dispute storage disp = disputesById[_disputeId];
        //Ensure this has not already been executed/tallied
        require(disp.executed == false, "Dispute has been already executed");
        //Ensure that the vote has been open long enough
        require(
            block.timestamp >= disp.disputeUintVars[_MIN_EXECUTION_DATE],
            "Time for voting haven't elapsed"
        );
        //Ensure that it's a valid disputeId
        require(
            disp.reportingParty != address(0),
            "reporting Party is address 0"
        );
        int256 _tally = disp.tally;
        if (_tally > 0) {
        //If the vote is not a proposed fork
            if (disp.isPropFork == false) {
                //Set the dispute state to passed/true
                disp.disputeVotePassed = true;
                //Ensure the time for voting has elapsed
                StakeInfo storage stakes = stakerDetails[disp.reportedMiner];
                //If the vote for disputing a value is successful(disp.tally >0) then unstake the reported
                if (stakes.currentStatus == 3) {
                    stakes.currentStatus = 4;
                }
            } else if (uint256(_tally) >= ((uints[_TOTAL_SUPPLY] * 5) / 100)) {
                disp.disputeVotePassed = true;
            }
        }
        disp.disputeUintVars[_TALLY_DATE] = block.timestamp;
        disp.executed = true;
        emit DisputeVoteTallied(
            _disputeId,
            _tally,
            disp.reportedMiner,
            disp.reportingParty,
            disp.disputeVotePassed
        );
    }

    /**
     * @dev This function updates the minimum dispute fee as a function of the amount
     * of staked miners
     */
    function updateMinDisputeFee() public {
        uint256 _stakeAmt = uints[_STAKE_AMOUNT];
        uint256 _trgtMiners = uints[_TARGET_MINERS];
        uints[_DISPUTE_FEE] = SafeMath.max(
            15e18,
            (_stakeAmt -
                ((_stakeAmt *
                    (SafeMath.min(_trgtMiners, uints[_STAKE_COUNT]) * 1000)) /
                    _trgtMiners) /
                1000)
        );
    }

    /**
     * @dev Updates the Tellor address after a proposed fork has
     * passed the vote and day has gone by without a dispute
     * @param _disputeId the disputeId for the proposed fork
    */
    function updateTellor(uint256 _disputeId) external {
        bytes32 _hash = disputesById[_disputeId].hash;
        uint256 origID = disputeIdByDisputeHash[_hash];
        //this checks the "lastID" or the most recent if this is a multiple dispute case
        uint256 lastID =
            disputesById[origID].disputeUintVars[
                keccak256(
                    abi.encode(
                        disputesById[origID].disputeUintVars[_DISPUTE_ROUNDS]
                    )
                )
            ];
        TellorStorage.Dispute storage disp = disputesById[lastID];
        require(disp.isPropFork, "must be a fork proposal");
        require(
            disp.disputeUintVars[_FORK_EXECUTED] == 0,
            "update Tellor has already been run"
        );
        require(disp.disputeVotePassed == true, "vote needs to pass");
        require(disp.disputeUintVars[_TALLY_DATE] > 0, "vote needs to be tallied");
        require(
            block.timestamp - disp.disputeUintVars[_TALLY_DATE] > 1 days,
            "Time for voting for further disputes has not passed"
        );
        disp.disputeUintVars[_FORK_EXECUTED] = 1;
        address _newTellor =disp.proposedForkAddress;
        addresses[_TELLOR_CONTRACT] = _newTellor; 
        assembly {
            sstore(_EIP_SLOT, _newTellor)
        }
        emit NewTellorAddress(_newTellor);
    }

    /**
     * @dev This function allows users to withdraw their stake after a 7 day waiting
     * period from request
     */
    function withdrawStake() external {
        StakeInfo storage stakes = stakerDetails[msg.sender];
        //Require the staker has locked for withdraw(currentStatus ==2) and that 7 days have
        //passed by since they locked for withdraw
        require(
            block.timestamp - (block.timestamp % 86400) - stakes.startDate >=
                7 days,
            "7 days didn't pass"
        );
        require(
            stakes.currentStatus == 2,
            "Miner was not locked for withdrawal"
        );
        stakes.currentStatus = 0;
        emit StakeWithdrawn(msg.sender);
    }

    /**
     * @dev This internal function is used the depositStake function to successfully stake miners.
     * The function updates their status/state and status start date so they are locked it so they can't withdraw
     * and updates the number of stakers in the system.
     * @param _staker the address of the new staker
    */
    function _newStake(address _staker) internal {
        require(
            balances[_staker][balances[_staker].length - 1].value >=
                uints[_STAKE_AMOUNT],
            "Balance is lower than stake amount"
        );
        //Ensure they can only stake if they are not currently staked or if their stake time frame has ended
        //and they are currently locked for withdraw
        require(
            stakerDetails[_staker].currentStatus == 0 ||
                stakerDetails[_staker].currentStatus == 2,
            "Miner is in the wrong state"
        );
        uints[_STAKE_COUNT] += 1;
        stakerDetails[_staker] = StakeInfo({
            currentStatus: 1, 
            startDate: block.timestamp//this resets their stake start date to now
        });
        emit NewStake(_staker);
    }
}
