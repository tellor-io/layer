// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import "./interfaces/IOracle.sol";
import "./interfaces/IERC20.sol";
import "usingtellor/contracts/UsingTellor.sol";

/**
 @author Tellor Inc.
 @title Governance
 @dev This is a governance contract to be used with TellorFlex. It handles disputing
 * Tellor oracle data and voting on those disputes
*/
contract Governance is UsingTellor {
    // Storage
    IOracle public oracle; // Tellor oracle contract
    IERC20 public token; // token used for dispute fees, same as reporter staking token
    address public oracleAddress; //tellorFlex address
    address public teamMultisig; // address of team multisig wallet, one of four stakeholder groups
    uint256 public voteCount; // total number of votes initiated
    bytes32 public autopayAddrsQueryId =
        keccak256(abi.encode("AutopayAddresses", abi.encode(bytes("")))); // query id for autopay addresses array
    mapping(uint256 => Dispute) private disputeInfo; // mapping of dispute IDs to the details of the dispute
    mapping(bytes32 => uint256) private openDisputesOnId; // mapping of a query ID to the number of disputes on that query ID
    mapping(uint256 => Vote) private voteInfo; // mapping of dispute IDs to the details of the vote
    mapping(bytes32 => uint256[]) private voteRounds; // mapping of vote identifier hashes to an array of dispute IDs
    mapping(address => uint256) private voteTallyByAddress; // mapping of addresses to the number of votes they have cast
    mapping(address => uint256[]) private disputeIdsByReporter; // mapping of reporter addresses to an array of dispute IDs

    enum VoteResult {
        FAILED,
        PASSED,
        INVALID
    } // status of a potential vote

    // Structs
    struct Dispute {
        bytes32 queryId; // query ID of disputed value
        uint256 timestamp; // timestamp of disputed value
        bytes value; // disputed value
        address disputedReporter; // reporter who submitted the disputed value
        uint256 slashedAmount; // amount of tokens slashed from reporter
    }

    struct Tally {
        uint256 doesSupport; // number of votes in favor
        uint256 against; // number of votes against
        uint256 invalidQuery; // number of votes for invalid
    }

    struct Vote {
        bytes32 identifierHash; // identifier hash of the vote
        uint256 voteRound; // the round of voting on a given dispute or proposal
        uint256 startDate; // timestamp of when vote was initiated
        uint256 blockNumber; // block number of when vote was initiated
        uint256 fee; // fee paid to initiate the vote round
        uint256 tallyDate; // timestamp of when the votes were tallied
        Tally tokenholders; // vote tally of tokenholders
        Tally users; // vote tally of users
        Tally reporters; // vote tally of reporters
        Tally teamMultisig; // vote tally of teamMultisig
        bool executed; // boolean of whether the vote was executed
        VoteResult result; // VoteResult after votes were tallied
        address initiator; // address which initiated dispute/proposal
        mapping(address => bool) voted; // mapping of address to whether or not they voted
    }

    // Events
    event NewDispute(
        uint256 _disputeId,
        bytes32 _queryId,
        uint256 _timestamp,
        address _reporter
    ); // Emitted when a new dispute is opened

    event Voted(
        uint256 _disputeId,
        bool _supports,
        address _voter,
        bool _invalidQuery
    ); // Emitted when an address casts their vote
    event VoteExecuted(uint256 _disputeId, VoteResult _result); // Emitted when a vote is executed
    event VoteTallied(
        uint256 _disputeId,
        VoteResult _result,
        address _initiator,
        address _reporter
    ); // Emitted when all casting for a vote is tallied

    /**
     * @dev Initializes contract parameters
     * @param _tellor address of tellor oracle contract to be governed
     * @param _teamMultisig address of tellor team multisig, one of four voting
     * stakeholder groups
     */
    constructor(address payable _tellor, address _teamMultisig)
        UsingTellor(_tellor)
    {
        oracle = IOracle(_tellor);
        token = IERC20(oracle.getTokenAddress());
        oracleAddress = _tellor;
        teamMultisig = _teamMultisig;
    }

    /**
     * @dev Initializes a dispute/vote in the system
     * @param _queryId being disputed
     * @param _timestamp being disputed
     */
    function beginDispute(bytes32 _queryId, uint256 _timestamp) external {
        // Ensure value actually exists
        require(
            oracle.getBlockNumberByTimestamp(_queryId, _timestamp) != 0,
            "no value exists at given timestamp"
        );
        bytes32 _hash = keccak256(abi.encodePacked(_queryId, _timestamp));
        // Push new vote round
        uint256 _disputeId = voteCount + 1;
        uint256[] storage _voteRounds = voteRounds[_hash];
        _voteRounds.push(_disputeId);

        // Create new vote and dispute
        Vote storage _thisVote = voteInfo[_disputeId];
        Dispute storage _thisDispute = disputeInfo[_disputeId];

        // Initialize dispute information - query ID, timestamp, value, etc.
        _thisDispute.queryId = _queryId;
        _thisDispute.timestamp = _timestamp;
        _thisDispute.disputedReporter = oracle.getReporterByTimestamp(
            _queryId,
            _timestamp
        );
        // Initialize vote information - hash, initiator, block number, etc.
        _thisVote.identifierHash = _hash;
        _thisVote.initiator = msg.sender;
        _thisVote.blockNumber = block.number;
        _thisVote.startDate = block.timestamp;
        _thisVote.voteRound = _voteRounds.length;
        disputeIdsByReporter[_thisDispute.disputedReporter].push(_disputeId);
        uint256 _disputeFee = getDisputeFee();
        if (_voteRounds.length == 1) {
            require(
                block.timestamp - _timestamp < 12 hours,
                "Dispute must be started within reporting lock time"
            );
            openDisputesOnId[_queryId]++;
            // calculate dispute fee based on number of open disputes on query ID
            _disputeFee = _disputeFee * 2**(openDisputesOnId[_queryId] - 1);
            // slash a single stakeAmount from reporter
            _thisDispute.slashedAmount = oracle.slashReporter(_thisDispute.disputedReporter, address(this));
            _thisDispute.value = oracle.retrieveData(_queryId, _timestamp);
            oracle.removeValue(_queryId, _timestamp);
        } else {
            uint256 _prevId = _voteRounds[_voteRounds.length - 2];
            require(
                block.timestamp - voteInfo[_prevId].tallyDate < 1 days,
                "New dispute round must be started within a day"
            );
            _disputeFee = _disputeFee * 2**(_voteRounds.length - 1);
            _thisDispute.slashedAmount = disputeInfo[_voteRounds[0]].slashedAmount;
            _thisDispute.value = disputeInfo[_voteRounds[0]].value;
        }
        if (_disputeFee > oracle.getStakeAmount()) {
          _disputeFee = oracle.getStakeAmount();
        }
        _thisVote.fee = _disputeFee;
        voteCount++;
        require(
            token.transferFrom(msg.sender, address(this), _disputeFee),
            "Fee must be paid"
        ); // This is the dispute fee. Returned if dispute passes
        emit NewDispute(
            _disputeId,
            _queryId,
            _timestamp,
            _thisDispute.disputedReporter
        );
    }

    /**
     * @dev Executes vote and transfers corresponding balances to initiator/reporter
     * @param _disputeId is the ID of the vote being executed
     */
    function executeVote(uint256 _disputeId) external {
        // Ensure validity of vote ID, vote has been executed, and vote must be tallied
        Vote storage _thisVote = voteInfo[_disputeId];
        require(_disputeId <= voteCount && _disputeId > 0, "Dispute ID must be valid");
        require(!_thisVote.executed, "Vote has already been executed");
        require(_thisVote.tallyDate > 0, "Vote must be tallied");
        // Ensure vote must be final vote and that time has to be pass (86400 = 24 * 60 * 60 for seconds in a day)
        require(
            voteRounds[_thisVote.identifierHash].length == _thisVote.voteRound,
            "Must be the final vote"
        );
        //The time  has to pass after the vote is tallied
        require(
            block.timestamp - _thisVote.tallyDate >= 1 days,
            "1 day has to pass after tally to allow for disputes"
        );
        _thisVote.executed = true;
        Dispute storage _thisDispute = disputeInfo[_disputeId];
        openDisputesOnId[_thisDispute.queryId]--;
        uint256 _i;
        uint256 _voteID;
        if (_thisVote.result == VoteResult.PASSED) {
            // If vote is in dispute and passed, iterate through each vote round and transfer the dispute to initiator
            for (
                _i = voteRounds[_thisVote.identifierHash].length;
                _i > 0;
                _i--
            ) {
                _voteID = voteRounds[_thisVote.identifierHash][_i - 1];
                _thisVote = voteInfo[_voteID];
                // If the first vote round, also make sure to transfer the reporter's slashed stake to the initiator
                if (_i == 1) {
                    token.transfer(
                        _thisVote.initiator,
                        _thisDispute.slashedAmount
                    );
                }
                token.transfer(_thisVote.initiator, _thisVote.fee);
            }
        } else if (_thisVote.result == VoteResult.INVALID) {
            // If vote is in dispute and is invalid, iterate through each vote round and transfer the dispute fee to initiator
            for (
                _i = voteRounds[_thisVote.identifierHash].length;
                _i > 0;
                _i--
            ) {
                _voteID = voteRounds[_thisVote.identifierHash][_i - 1];
                _thisVote = voteInfo[_voteID];
                token.transfer(_thisVote.initiator, _thisVote.fee);
            }
            // Transfer slashed tokens back to disputed reporter
            token.transfer(
                _thisDispute.disputedReporter,
                _thisDispute.slashedAmount
            );
        } else if (_thisVote.result == VoteResult.FAILED) {
            // If vote is in dispute and fails, iterate through each vote round and transfer the dispute fee to disputed reporter
            uint256 _reporterReward = 0;
            for (
                _i = voteRounds[_thisVote.identifierHash].length;
                _i > 0;
                _i--
            ) {
                _voteID = voteRounds[_thisVote.identifierHash][_i - 1];
                _thisVote = voteInfo[_voteID];
                _reporterReward += _thisVote.fee;
            }
            _reporterReward += _thisDispute.slashedAmount;
            token.transfer(_thisDispute.disputedReporter, _reporterReward);
        }
        emit VoteExecuted(_disputeId, voteInfo[_disputeId].result);
    }

    /**
     * @dev Tallies the votes and begins the 1 day challenge period
     * @param _disputeId is the dispute id
     */
    function tallyVotes(uint256 _disputeId) external {
        // Ensure vote has not been executed and that vote has not been tallied
        Vote storage _thisVote = voteInfo[_disputeId];
        require(_thisVote.tallyDate == 0, "Vote has already been tallied");
        require(_disputeId <= voteCount && _disputeId > 0, "Vote does not exist");
        // Determine appropriate vote duration dispute round
        // Vote time increases as rounds increase but only up to 6 days (withdrawal period)
        require(
            block.timestamp - _thisVote.startDate >=
                86400 * _thisVote.voteRound ||
                block.timestamp - _thisVote.startDate >= 86400 * 6,
            "Time for voting has not elapsed"
        );
        // Get total votes from each separate stakeholder group.  This will allow
        // normalization so each group's votes can be combined and compared to
        // determine the vote outcome.
        uint256 _tokenVoteSum = _thisVote.tokenholders.doesSupport +
            _thisVote.tokenholders.against +
            _thisVote.tokenholders.invalidQuery;
        uint256 _reportersVoteSum = _thisVote.reporters.doesSupport +
            _thisVote.reporters.against +
            _thisVote.reporters.invalidQuery;
        uint256 _multisigVoteSum = _thisVote.teamMultisig.doesSupport +
            _thisVote.teamMultisig.against +
            _thisVote.teamMultisig.invalidQuery;
        uint256 _usersVoteSum = _thisVote.users.doesSupport +
            _thisVote.users.against +
            _thisVote.users.invalidQuery;
        // Cannot divide by zero
        if (_tokenVoteSum == 0) {
            _tokenVoteSum++;
        }
        if (_reportersVoteSum == 0) {
            _reportersVoteSum++;
        }
        if (_multisigVoteSum == 0) {
            _multisigVoteSum++;
        }
        if (_usersVoteSum == 0) {
            _usersVoteSum++;
        }
        // Normalize and combine each stakeholder group votes
        uint256 _scaledDoesSupport = ((_thisVote.tokenholders.doesSupport *
            1e18) / _tokenVoteSum) +
            ((_thisVote.reporters.doesSupport * 1e18) / _reportersVoteSum) +
            ((_thisVote.teamMultisig.doesSupport * 1e18) / _multisigVoteSum) +
            ((_thisVote.users.doesSupport * 1e18) / _usersVoteSum);
        uint256 _scaledAgainst = ((_thisVote.tokenholders.against * 1e18) /
            _tokenVoteSum) +
            ((_thisVote.reporters.against * 1e18) / _reportersVoteSum) +
            ((_thisVote.teamMultisig.against * 1e18) / _multisigVoteSum) +
            ((_thisVote.users.against * 1e18) / _usersVoteSum);
        uint256 _scaledInvalid = ((_thisVote.tokenholders.invalidQuery * 1e18) /
            _tokenVoteSum) +
            ((_thisVote.reporters.invalidQuery * 1e18) / _reportersVoteSum) +
            ((_thisVote.teamMultisig.invalidQuery * 1e18) / _multisigVoteSum) +
            ((_thisVote.users.invalidQuery * 1e18) / _usersVoteSum);

        // If votes in support outweight the sum of against and invalid, result is passed
        if (_scaledDoesSupport > _scaledAgainst + _scaledInvalid) {
            _thisVote.result = VoteResult.PASSED;
        // If votes in against outweight the sum of support and invalid, result is failed
        } else if (_scaledAgainst > _scaledDoesSupport + _scaledInvalid) {
            _thisVote.result = VoteResult.FAILED;
        // Otherwise, result is invalid
        } else {
            _thisVote.result = VoteResult.INVALID;
        }

        _thisVote.tallyDate = block.timestamp; // Update time vote was tallied
        emit VoteTallied(
            _disputeId,
            _thisVote.result,
            _thisVote.initiator,
            disputeInfo[_disputeId].disputedReporter
        );
    }

    /**
     * @dev Enables the sender address to cast a vote
     * @param _disputeId is the ID of the vote
     * @param _supports is the address's vote: whether or not they support or are against
     * @param _invalidQuery is whether or not the dispute is valid
     */
    function vote(
        uint256 _disputeId,
        bool _supports,
        bool _invalidQuery
    ) public {
        // Ensure that dispute has not been executed and that vote does not exist and is not tallied
        require(_disputeId <= voteCount && _disputeId > 0, "Vote does not exist");
        Vote storage _thisVote = voteInfo[_disputeId];
        require(_thisVote.tallyDate == 0, "Vote has already been tallied");
        require(!_thisVote.voted[msg.sender], "Sender has already voted");
        // Update voting status and increment total queries for support, invalid, or against based on vote
        _thisVote.voted[msg.sender] = true;
        uint256 _tokenBalance = token.balanceOf(msg.sender);
        (, uint256 _stakedBalance, uint256 _lockedBalance, , , , , ) = oracle.getStakerInfo(msg.sender);
        _tokenBalance += _stakedBalance + _lockedBalance;
        if (_invalidQuery) {
            _thisVote.tokenholders.invalidQuery += _tokenBalance;
            _thisVote.reporters.invalidQuery += oracle
                .getReportsSubmittedByAddress(msg.sender);
            _thisVote.users.invalidQuery += _getUserTips(msg.sender);
            if (msg.sender == teamMultisig) {
                _thisVote.teamMultisig.invalidQuery += 1;
            }
        } else if (_supports) {
            _thisVote.tokenholders.doesSupport += _tokenBalance;
            _thisVote.reporters.doesSupport += oracle.getReportsSubmittedByAddress(msg.sender);
            _thisVote.users.doesSupport += _getUserTips(msg.sender);
            if (msg.sender == teamMultisig) {
                _thisVote.teamMultisig.doesSupport += 1;
            }
        } else {
            _thisVote.tokenholders.against += _tokenBalance;
            _thisVote.reporters.against += oracle.getReportsSubmittedByAddress(
                msg.sender
            );
            _thisVote.users.against += _getUserTips(msg.sender);
            if (msg.sender == teamMultisig) {
                _thisVote.teamMultisig.against += 1;
            }
        }
        voteTallyByAddress[msg.sender]++;
        emit Voted(_disputeId, _supports, msg.sender, _invalidQuery);
    }

    /**
     * @dev Enables the sender address to cast votes for multiple disputes
     * @param _disputeIds is an array of vote IDs
     * @param _supports is an array of the address's votes: whether or not they support or are against
     * @param _invalidQuery is array of whether or not the dispute is valid
     */
    function voteOnMultipleDisputes(
        uint256[] memory _disputeIds, 
        bool[] memory _supports, 
        bool[] memory _invalidQuery
    ) external {
        for (uint256 _i = 0; _i < _disputeIds.length; _i++) {
            vote(_disputeIds[_i], _supports[_i], _invalidQuery[_i]);
        }
    }

    // *****************************************************************************
    // *                                                                           *
    // *                               Getters                                     *
    // *                                                                           *
    // *****************************************************************************

    /**
     * @dev Determines if an address voted for a specific vote
     * @param _disputeId is the ID of the vote
     * @param _voter is the address of the voter to check for
     * @return bool of whether or note the address voted for the specific vote
     */
    function didVote(uint256 _disputeId, address _voter)
        external
        view
        returns (bool)
    {
        return voteInfo[_disputeId].voted[_voter];
    }

    /**
     * @dev Get the latest dispute fee
     */
    function getDisputeFee() public view returns (uint256) {
        return (oracle.getStakeAmount() / 10);
    }


    function getDisputesByReporter(address _reporter) external view returns (uint256[] memory) {
        return disputeIdsByReporter[_reporter];
    }

    /**
     * @dev Returns info on a dispute for a given ID
     * @param _disputeId is the ID of a specific dispute
     * @return bytes32 of the data ID of the dispute
     * @return uint256 of the timestamp of the dispute
     * @return bytes memory of the value being disputed
     * @return address of the reporter being disputed
     */
    function getDisputeInfo(uint256 _disputeId)
        external
        view
        returns (
            bytes32,
            uint256,
            bytes memory,
            address
        )
    {
        Dispute storage _d = disputeInfo[_disputeId];
        return (_d.queryId, _d.timestamp, _d.value, _d.disputedReporter);
    }

    /**
     * @dev Returns the number of open disputes for a specific query ID
     * @param _queryId is the ID of a specific data feed
     * @return uint256 of the number of open disputes for the query ID
     */
    function getOpenDisputesOnId(bytes32 _queryId)
        external
        view
        returns (uint256)
    {
        return openDisputesOnId[_queryId];
    }

    /**
     * @dev Returns the total number of votes
     * @return uint256 of the total number of votes
     */
    function getVoteCount() external view returns (uint256) {
        return voteCount;
    }

    /**
     * @dev Returns info on a vote for a given vote ID
     * @param _disputeId is the ID of a specific vote
     * @return bytes32 identifier hash of the vote
     * @return uint256[17] memory of the pertinent round info (vote rounds, start date, fee, etc.)
     * @return bool memory of both whether or not the vote was executed
     * @return VoteResult result of the vote
     * @return address memory of the vote initiator
     */
    function getVoteInfo(uint256 _disputeId)
        external
        view
        returns (
            bytes32,
            uint256[17] memory,
            bool,
            VoteResult,
            address
        )
    {
        Vote storage _v = voteInfo[_disputeId];
        return (
            _v.identifierHash,
            [
                _v.voteRound,
                _v.startDate,
                _v.blockNumber,
                _v.fee,
                _v.tallyDate,
                _v.tokenholders.doesSupport,
                _v.tokenholders.against,
                _v.tokenholders.invalidQuery,
                _v.users.doesSupport,
                _v.users.against,
                _v.users.invalidQuery,
                _v.reporters.doesSupport,
                _v.reporters.against,
                _v.reporters.invalidQuery,
                _v.teamMultisig.doesSupport,
                _v.teamMultisig.against,
                _v.teamMultisig.invalidQuery
            ],
            _v.executed,
            _v.result,
            _v.initiator
        );
    }

    /**
     * @dev Returns an array of voting rounds for a given vote
     * @param _hash is the identifier hash for a vote
     * @return uint256[] memory dispute IDs of the vote rounds
     */
    function getVoteRounds(bytes32 _hash)
        external
        view
        returns (uint256[] memory)
    {
        return voteRounds[_hash];
    }

    /**
     * @dev Returns the total number of votes cast by an address
     * @param _voter is the address of the voter to check for
     * @return uint256 of the total number of votes cast by the voter
     */
    function getVoteTallyByAddress(address _voter)
        external
        view
        returns (uint256)
    {
        return voteTallyByAddress[_voter];
    }

    // Internal
    /**
     * @dev Retrieves total tips contributed to autopay by a given address
     * @param _user address of the user to check the tip count for
     * @return _userTipTally uint256 of total tips contributed to autopay by the address
     */
    function _getUserTips(address _user) internal returns (uint256 _userTipTally) {
        // get autopay addresses array from oracle
        (bytes memory _autopayAddrsBytes, uint256 _timestamp) = getDataBefore(
            autopayAddrsQueryId,
            block.timestamp - 12 hours
        );
        if (_timestamp > 0) {
            address[] memory _autopayAddrs = abi.decode(
                _autopayAddrsBytes,
                (address[])
            );
            // iterate through autopay addresses retrieve tips by user address
            for (uint256 _i = 0; _i < _autopayAddrs.length; _i++) {
                (bool _success, bytes memory _returnData) = _autopayAddrs[_i]
                    .call(
                        abi.encodeWithSignature(
                            "getTipsByAddress(address)",
                            _user
                        )
                    );
                if (_success) {
                    _userTipTally += abi.decode(_returnData, (uint256));
                }
            }
        }
    }
}
