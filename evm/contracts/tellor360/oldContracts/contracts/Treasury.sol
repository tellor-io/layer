// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import "./interfaces/IController.sol";
import "./TellorVars.sol";
import "./interfaces/IGovernance.sol";

/**
 @author Tellor Inc.
 @title Treasury
 @dev This is the Treasury contract which defines the function for Tellor
 * treasuries, or staking pools.
*/
contract Treasury is TellorVars {
    // Storage
    uint256 public totalLocked; // amount of TRB locked across all treasuries
    uint256 public treasuryCount; // number of total treasuries
    mapping(uint256 => TreasuryDetails) public treasury; // maps an ID to a treasury and its corresponding details
    mapping(address => uint256) treasuryFundsByUser; // maps a treasury investor to their total treasury funds, in TRB

    // Structs
    // Internal struct used to keep track of an individual user in a treasury
    struct TreasuryUser {
        uint256 amount; // the amount the user has placed in a treasury, in TRB
        uint256 startVoteCount; // the amount of votes that have been cast when a user deposits their money into a treasury
        bool paid; // determines if a user has paid/voted in Tellor governance proposals
    }
    // Internal struct used to keep track of a treasury and its pertinent attributes (amount, interest rate, etc.)
    struct TreasuryDetails {
        uint256 dateStarted; // the date that treasury was started
        uint256 maxAmount; // the maximum amount stored in the treasury, in TRB
        uint256 rate; // the interest rate of the treasury (5% == 500)
        uint256 purchasedAmount; // the amount of TRB purchased from the treasury
        uint256 duration; // the time during which the treasury locks participants
        uint256 endVoteCount; // the end vote count when the treasury duration is over
        bool endVoteCountRecorded; // determines whether the end vote count has been calculated or not
        address[] owners; // the owners of the treasury
        mapping(address => TreasuryUser) accounts; // a mapping of a treasury user address and corresponding details
    }

    // Events
    event TreasuryIssued(uint256 _id, uint256 _amount, uint256 _rate);
    event TreasuryPaid(address _investor, uint256 _amount);
    event TreasuryPurchased(address _investor, uint256 _amount);

    // Functions
    /**
     * @dev This is an external function that is used to deposit money into a treasury.
     * @param _id is the ID for a specific treasury instance
     * @param _amount is the amount to deposit into a treasury
     */
    function buyTreasury(uint256 _id, uint256 _amount) external {
        // Transfer sender funds to Treasury
        require(_amount > 0, "Amount must be greater than zero.");
        require(
            IController(TELLOR_ADDRESS).approveAndTransferFrom(
                msg.sender,
                address(this),
                _amount
            ),
            "Insufficient balance. Try a lower amount."
        );
        treasuryFundsByUser[msg.sender] += _amount;
        // Check for sufficient treasury funds
        TreasuryDetails storage _treas = treasury[_id];
        require(
            _treas.dateStarted + _treas.duration > block.timestamp,
            "Treasury duration has expired."
        );
        require(
            _amount <= _treas.maxAmount - _treas.purchasedAmount,
            "Not enough money in treasury left to purchase."
        );
        // Update treasury details -- vote count, purchasedAmount, amount, and owners
        address _governanceContract = IController(TELLOR_ADDRESS).addresses(
            _GOVERNANCE_CONTRACT
        );
        if (_treas.accounts[msg.sender].amount == 0) {
            _treas.accounts[msg.sender].startVoteCount = IGovernance(
                _governanceContract
            ).getVoteCount();
            _treas.owners.push(msg.sender);
        }
        _treas.purchasedAmount += _amount;
        _treas.accounts[msg.sender].amount += _amount;
        totalLocked += _amount;
        emit TreasuryPurchased(msg.sender, _amount);
    }

    /**
     * @dev This is an external function that is used to issue a new treasury.
     * Note that only the governance contract can call this function.
     * @param _maxAmount is the amount of total TRB that treasury stores
     * @param _rate is the treasury's interest rate in BP
     * @param _duration is the amount of time the treasury locks participants
     */
    function issueTreasury(
        uint256 _maxAmount,
        uint256 _rate,
        uint256 _duration
    ) external {
        require(
            msg.sender ==
                IController(TELLOR_ADDRESS).addresses(_GOVERNANCE_CONTRACT),
            "Only governance contract is allowed to issue a treasury."
        );
        require(
            _maxAmount > 0 &&
                _maxAmount <= IController(TELLOR_ADDRESS).totalSupply(),
            "Invalid maxAmount value"
        );
        require(
            _duration > 0 && _duration <= 315360000,
            "Invalid duration value"
        );
        require(_rate > 0 && _rate <= 10000, "Invalid rate value");
        // Increment treasury count, and define new treasury and its details (start date, total amount, rate, etc.)
        treasuryCount++;
        TreasuryDetails storage _treas = treasury[treasuryCount];
        _treas.dateStarted = block.timestamp;
        _treas.maxAmount = _maxAmount;
        _treas.rate = _rate;
        _treas.duration = _duration;
        emit TreasuryIssued(treasuryCount, _maxAmount, _rate);
    }

    /**
     * @dev This functions allows an investor to pay the treasury. Internally, the function calculates the number of
     votes in governance contract when issued, and also transfers the amount individually locked + interest to the investor.
     * @param _id is the ID of the treasury the account is stored in
     * @param _investor is the address of the account in the treasury
     */
    function payTreasury(address _investor, uint256 _id) external {
        // Validate ID of treasury, duration for treasury has not passed, and the user has not paid
        TreasuryDetails storage _treas = treasury[_id];
        require(
            _id <= treasuryCount,
            "ID does not correspond to a valid treasury."
        );
        require(
            _treas.dateStarted + _treas.duration <= block.timestamp,
            "Treasury duration has not expired."
        );
        require(
            !_treas.accounts[_investor].paid,
            "Treasury investor has already been paid."
        );
        require(
            _treas.accounts[_investor].amount > 0,
            "Address is not a treasury investor"
        );
        // Calculate non-voting penalty (treasury holders have to vote)
        uint256 numVotesParticipated;
        uint256 votesSinceTreasury;
        address governanceContract = IController(TELLOR_ADDRESS).addresses(
            _GOVERNANCE_CONTRACT
        );
        // Find endVoteCount if not already calculated
        if (!_treas.endVoteCountRecorded) {
            uint256 voteCountIter = IGovernance(governanceContract)
                .getVoteCount();
            if (voteCountIter > 0) {
                (, uint256[8] memory voteInfo, , , , , ) = IGovernance(
                    governanceContract
                ).getVoteInfo(voteCountIter);
                while (
                    voteCountIter > 0 &&
                    voteInfo[1] > _treas.dateStarted + _treas.duration
                ) {
                    voteCountIter--;
                    if (voteCountIter > 0) {
                        (, voteInfo, , , , , ) = IGovernance(governanceContract)
                            .getVoteInfo(voteCountIter);
                    }
                }
            }
            _treas.endVoteCount = voteCountIter;
            _treas.endVoteCountRecorded = true;
        }
        // Add up number of votes _investor has participated in
        if (_treas.endVoteCount > _treas.accounts[_investor].startVoteCount) {
            for (
                uint256 voteCount = _treas.accounts[_investor].startVoteCount;
                voteCount < _treas.endVoteCount;
                voteCount++
            ) {
                bool voted = IGovernance(governanceContract).didVote(
                    voteCount + 1,
                    _investor
                );
                if (voted) {
                    numVotesParticipated++;
                }
                votesSinceTreasury++;
            }
        }
        // Determine amount of TRB to mint for interest
        uint256 _mintAmount = (_treas.accounts[_investor].amount *
            _treas.rate) / 10000;
        if (votesSinceTreasury > 0) {
            _mintAmount =
                (_mintAmount * numVotesParticipated) /
                votesSinceTreasury;
        }
        if (_mintAmount > 0) {
            IController(TELLOR_ADDRESS).mint(address(this), _mintAmount);
        }
        // Transfer locked amount + interest amount, and indicate user has paid
        totalLocked -= _treas.accounts[_investor].amount;
        IController(TELLOR_ADDRESS).transfer(
            _investor,
            _mintAmount + _treas.accounts[_investor].amount
        );
        treasuryFundsByUser[_investor] -= _treas.accounts[_investor].amount;
        _treas.accounts[_investor].paid = true;
        emit TreasuryPaid(
            _investor,
            _mintAmount + _treas.accounts[_investor].amount
        );
    }

    // Getters
    /**
     * @dev This function returns the details of an account within a treasury.
     * Note: refer to 'TreasuryUser' struct.
     * @param _id is the ID of the treasury the account is stored in
     * @param _investor is the address of the account in the treasury
     * @return uint256 of the amount of TRB the account has staked in the treasury
     * @return uint256 of the start vote count of when the account deposited money into the treasury
     * @return bool of whether the treasury account has paid or not
     */
    function getTreasuryAccount(uint256 _id, address _investor)
        external
        view
        returns (
            uint256,
            uint256,
            bool
        )
    {
        return (
            treasury[_id].accounts[_investor].amount,
            treasury[_id].accounts[_investor].startVoteCount,
            treasury[_id].accounts[_investor].paid
        );
    }

    /**
     * @dev This function returns the number of treasuries/TellorX staking pools.
     * @return uint256 of the number of treasuries
     */
    function getTreasuryCount() external view returns (uint256) {
        return treasuryCount;
    }

    function getTreasuryDetails(uint256 _id)
        external
        view
        returns (
            uint256,
            uint256,
            uint256,
            uint256
        )
    {
        return (
            treasury[_id].dateStarted,
            treasury[_id].maxAmount,
            treasury[_id].rate,
            treasury[_id].purchasedAmount
        );
    }

    /**
     * @dev This function returns the amount of TRB deposited by a user into treasuries.
     * @param _user is the specific account within a treasury to look up
     * @return uint256 of the amount of funds the user has, in TRB
     */
    function getTreasuryFundsByUser(address _user)
        external
        view
        returns (uint256)
    {
        return treasuryFundsByUser[_user];
    }

    /**
     * @dev This function returns the addresses of the owners of a treasury
     * @param _id is the ID of a specific treasury
     * @return address[] memory of the addresses of the owners of the treasury
     */
    function getTreasuryOwners(uint256 _id)
        external
        view
        returns (address[] memory)
    {
        return treasury[_id].owners;
    }

    /**
     * @dev This function is used during the upgrade process to verify valid Tellor Contracts
     */
    function verify() external pure returns (uint256) {
        return 9999;
    }

    /**
     * @dev This function determines whether or not an investor in a treasury has paid/voted on Tellor governance proposals
     * @param _id is the ID of the treasury the account is stored in
     * @param _investor is the address of the account in the treasury
     * @return bool of whether or not the investor was paid
     */
    function wasPaid(uint256 _id, address _investor)
        external
        view
        returns (bool)
    {
        return treasury[_id].accounts[_investor].paid;
    }
}
