// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import "./Token.sol";
import "./interfaces/IGovernance.sol";

/**
 @author Tellor Inc.
 @title TellorStaking
 @dev This is the TellorStaking contract which defines the functionality for
 * updating staking statuses for reporters, including depositing and withdrawing
 * stakes.
*/
contract TellorStaking is Token {
    // Events
    event NewStaker(address _staker);
    event StakeWithdrawRequested(address _staker);
    event StakeWithdrawn(address _staker);

    // Functions
    /**
     * @dev Changes staking status of a reporter
     * Note: this function is only callable by the Governance contract.
     * @param _reporter is the address of the reporter to change staking status for
     * @param _status is the new status of the reporter
     */
    function changeStakingStatus(address _reporter, uint256 _status) external {
        require(
            IGovernance(addresses[_GOVERNANCE_CONTRACT])
                .isApprovedGovernanceContract(msg.sender),
            "Only approved governance contract can change staking status"
        );
        StakeInfo storage stakes = stakerDetails[_reporter];
        stakes.currentStatus = _status;
    }

    /**
     * @dev Allows a reporter to submit stake
     */
    function depositStake() external {
        // Ensure staker has enough balance to stake
        require(
            balances[msg.sender][balances[msg.sender].length - 1].value >=
                uints[_STAKE_AMOUNT],
            "Balance is lower than stake amount"
        );
        // Ensure staker is currently either not staked or locked for withdraw.
        // Note that slashed reporters cannot stake again from a slashed address.
        require(
            stakerDetails[msg.sender].currentStatus == 0 ||
                stakerDetails[msg.sender].currentStatus == 2,
            "Reporter is in the wrong state"
        );
        // Increment number of stakers, create new staker, and update dispute fee
        uints[_STAKE_COUNT] += 1;
        stakerDetails[msg.sender] = StakeInfo({
            currentStatus: 1,
            startDate: block.timestamp // This resets their stake start date to now
        });
        emit NewStaker(msg.sender);
        IGovernance(addresses[_GOVERNANCE_CONTRACT]).updateMinDisputeFee();
    }

    /**
     * @dev Allows a reporter to request to withdraw their stake
     */
    function requestStakingWithdraw() external {
        // Ensures reporter is already staked
        StakeInfo storage stakes = stakerDetails[msg.sender];
        require(stakes.currentStatus == 1, "Reporter is not staked");
        // Change status to reflect withdraw request and updates start date for staking
        stakes.currentStatus = 2;
        stakes.startDate = block.timestamp;
        // Update number of stakers and dispute fee
        uints[_STAKE_COUNT] -= 1;
        IGovernance(addresses[_GOVERNANCE_CONTRACT]).updateMinDisputeFee();
        emit StakeWithdrawRequested(msg.sender);
    }

    /**
     * @dev Slashes a reporter and transfers their stake amount to their disputer
     * Note: this function is only callable by the Governance contract.
     * @param _reporter is the address of the reporter being slashed
     * @param _disputer is the address of the disputer receiving the reporter's stake
     */
    function slashReporter(address _reporter, address _disputer) external {
        require(
            IGovernance(addresses[_GOVERNANCE_CONTRACT])
                .isApprovedGovernanceContract(msg.sender),
            "Only approved governance contract can slash reporter"
        );
        stakerDetails[_reporter].currentStatus = 5; // Change status of reporter to slashed
        // Transfer stake amount of reporter has a balance bigger than the stake amount
        if (balanceOf(_reporter) >= uints[_STAKE_AMOUNT]) {
            _doTransfer(_reporter, _disputer, uints[_STAKE_AMOUNT]);
        }
        // Else, transfer all of the reporter's balance
        else if (balanceOf(_reporter) > 0) {
            _doTransfer(_reporter, _disputer, balanceOf(_reporter));
        }
    }

    /**
     * @dev Withdraws a reporter's stake
     */
    function withdrawStake() external {
        StakeInfo storage _s = stakerDetails[msg.sender];
        // Ensure reporter is locked and that enough time has passed
        require(block.timestamp - _s.startDate >= 7 days, "7 days didn't pass");
        require(_s.currentStatus == 2, "Reporter not locked for withdrawal");
        _s.currentStatus = 0; // Updates status to withdrawn
        emit StakeWithdrawn(msg.sender);
    }

    /**GETTERS**/
    /**
     * @dev Allows users to retrieve all information about a staker
     * @param _staker address of staker inquiring about
     * @return uint current state of staker
     * @return uint startDate of staking
     */
    function getStakerInfo(address _staker)
        external
        view
        returns (uint256, uint256)
    {
        return (
            stakerDetails[_staker].currentStatus,
            stakerDetails[_staker].startDate
        );
    }
}
