// SPDX-License-Identifier: MIT
pragma solidity 0.7.4;

import "./TellorTransfer.sol";
import "./TellorGetters.sol";
import "./Extension.sol";
import "./Utilities.sol";

/**
 @author Tellor Inc.
 @title TellorStake
 @dev Contains the methods related to initiating disputes and
 * voting on them.
 * Because of space limitations some functions are currently on the Extensions contract
*/
contract TellorStake is TellorTransfer {
    using SafeMath for uint256;
    using SafeMath for int256;

    //this belongs to Tellor, not to master
    uint256 private constant CURRENT_VERSION = 2999;

    //emitted when a new dispute is initialized
    event NewDispute(
        uint256 indexed _disputeId,
        uint256 indexed _requestId,
        uint256 _timestamp,
        address _miner
    );
    //emitted when a new vote happens
    event Voted(
        uint256 indexed _disputeID,
        bool _position,
        address indexed _voter,
        uint256 indexed _voteWeight
    );

    /**
     * @dev Helps initialize a dispute by assigning it a disputeId
     * when a miner returns a false/bad value on the validate array(in Tellor.ProofOfWork) it sends the
     * invalidated value information to POS voting
     * @param _requestId being disputed
     * @param _timestamp being disputed
     * @param _minerIndex the index of the miner that submitted the value being disputed. Since each official value
     * requires 5 miners to submit a value.
     */
    function beginDispute(
        uint256 _requestId,
        uint256 _timestamp,
        uint256 _minerIndex
    ) external {
        Request storage _request = requestDetails[_requestId];
        require(_request.minedBlockNum[_timestamp] != 0, "Mined block is 0");
        require(_minerIndex < 5, "Miner index is wrong");
        //_miner is the miner being disputed. For every mined value 5 miners are saved in an array and the _minerIndex
        //provided by the party initiating the dispute
        address _miner = _request.minersByValue[_timestamp][_minerIndex];
        uints[keccak256(abi.encodePacked(_miner,"DisputeCount"))]++;
        bytes32 _hash =
            keccak256(abi.encodePacked(_miner, _requestId, _timestamp));
        //Increase the dispute count by 1
        uints[_DISPUTE_COUNT]++;
        uint256 disputeId = uints[_DISPUTE_COUNT];
        //Ensures that a dispute is not already open for the that miner, requestId and timestamp
        uint256 hashId = disputeIdByDisputeHash[_hash];
        if (hashId != 0) {
            disputesById[disputeId].disputeUintVars[_ORIGINAL_ID] = hashId;
        } else {
            require(block.timestamp - _timestamp < 7 days, "Dispute must be started within a week of bad value");
            disputeIdByDisputeHash[_hash] = disputeId;
            hashId = disputeId;
        }
        uint256 dispRounds = _updateLastId(disputeId,hashId);
        uint256 _fee;
        if (_minerIndex == 2) {
            requestDetails[_requestId].apiUintVars[_DISPUTE_COUNT] =
                requestDetails[_requestId].apiUintVars[_DISPUTE_COUNT] +
                1;
            //update dispute fee for this case
            _fee =
                uints[_STAKE_AMOUNT] *
                requestDetails[_requestId].apiUintVars[_DISPUTE_COUNT];
        } else {
            _fee = uints[_DISPUTE_FEE] * dispRounds;
        }
        //maps the dispute to the Dispute struct
        disputesById[disputeId].hash = _hash;
        disputesById[disputeId].isPropFork = false;
        disputesById[disputeId].reportedMiner = _miner;
        disputesById[disputeId].reportingParty = msg.sender;
        disputesById[disputeId].proposedForkAddress = address(0);
        disputesById[disputeId].executed = false;
        disputesById[disputeId].disputeVotePassed = false;
        disputesById[disputeId].tally = 0;
        //Saves all the dispute variables for the disputeId
        disputesById[disputeId].disputeUintVars[_REQUEST_ID] = _requestId;
        disputesById[disputeId].disputeUintVars[_TIMESTAMP] = _timestamp;
        disputesById[disputeId].disputeUintVars[_VALUE] = _request
            .valuesByTimestamp[_timestamp][_minerIndex];
        disputesById[disputeId].disputeUintVars[_MIN_EXECUTION_DATE] =
            block.timestamp +
            2 days *
            dispRounds;
        disputesById[disputeId].disputeUintVars[_BLOCK_NUMBER] = block.number;
        disputesById[disputeId].disputeUintVars[_MINER_SLOT] = _minerIndex;
        disputesById[disputeId].disputeUintVars[_FEE] = _fee;
        _doTransfer(msg.sender, address(this), _fee);
        //Values are sorted as they come in and the official value is the median of the first five
        //So the "official value" miner is always minerIndex==2. If the official value is being
        //disputed, it sets its status to inDispute(currentStatus = 3) so that users are made aware it is under dispute
        if (_minerIndex == 2) {
            _request.inDispute[_timestamp] = true;
            _request.finalValues[_timestamp] = 0;
        }
        stakerDetails[_miner].currentStatus = 3;
        emit NewDispute(disputeId, _requestId, _timestamp, _miner);
    }

    /**
     * @dev Allows for a fork to be proposed
     * @param _propNewTellorAddress address for new proposed Tellor
    */
    function proposeFork(address _propNewTellorAddress) external {
        require(uints[_LOCK] == 0, "no rentrancy");
        uints[_LOCK] = 1;
        _verify(_propNewTellorAddress);
        uints[_LOCK] = 0;
        bytes32 _hash = keccak256(abi.encode(_propNewTellorAddress));
        uints[_DISPUTE_COUNT]++;
        uint256 disputeId = uints[_DISPUTE_COUNT];
        if (disputeIdByDisputeHash[_hash] != 0) {
            disputesById[disputeId].disputeUintVars[
                _ORIGINAL_ID
            ] = disputeIdByDisputeHash[_hash];
        } else {
            disputeIdByDisputeHash[_hash] = disputeId;
        }
        uint256 dispRounds = _updateLastId(disputeId,disputeIdByDisputeHash[_hash]);
        disputesById[disputeId].hash = _hash;
        disputesById[disputeId].isPropFork = true;
        // I don't think we need those
        disputesById[disputeId].reportedMiner = msg.sender;
        disputesById[disputeId].reportingParty = msg.sender;
        disputesById[disputeId].proposedForkAddress = _propNewTellorAddress;
        disputesById[disputeId].tally = 0;
        _doTransfer(msg.sender, address(this), 100e18 * 2**(dispRounds - 1)); //This is the fork fee (just 100 tokens flat, no refunds.  Goes up quickly to dispute a bad vote)
        disputesById[disputeId].disputeUintVars[_BLOCK_NUMBER] = block.number;
        disputesById[disputeId].disputeUintVars[_MIN_EXECUTION_DATE] =
            block.timestamp +
            7 days;
    }

    /**
     * @dev Allows disputer to unlock the dispute fee
     * @param _disputeId to unlock fee from
    */
    function unlockDisputeFee(uint256 _disputeId) external {
        require(_disputeId <= uints[_DISPUTE_COUNT], "dispute does not exist");
        uint256 origID = disputeIdByDisputeHash[disputesById[_disputeId].hash];
        uint256 lastID =
            disputesById[origID].disputeUintVars[
                keccak256(
                    abi.encode(
                        disputesById[origID].disputeUintVars[_DISPUTE_ROUNDS]
                    )
                )
            ];
        if (lastID == 0) {
            lastID = origID;
        }
        Dispute storage disp = disputesById[origID];
        Dispute storage last = disputesById[lastID];
        //disputeRounds is increased by 1 so that the _id is not a negative number when it is the first time a dispute is initiated
        uint256 dispRounds = disp.disputeUintVars[_DISPUTE_ROUNDS];
        if (dispRounds == 0) {
            dispRounds = 1;
        }
        uint256 _id;
        require(disp.disputeUintVars[_PAID] == 0, "already paid out");
        require(!disp.isPropFork, "function not callable fork fork proposals");
        require(disp.disputeUintVars[_TALLY_DATE] > 0, "vote needs to be tallied");
        require(
            block.timestamp - last.disputeUintVars[_TALLY_DATE] > 1 days,
            "Time for a follow up dispute hasn't elapsed"
        );
        StakeInfo storage stakes = stakerDetails[disp.reportedMiner];
        disp.disputeUintVars[_PAID] = 1;
        if (last.disputeVotePassed == true) {
            //Changing the currentStatus and startDate unstakes the reported miner and transfers the stakeAmount
            stakes.startDate = block.timestamp - (block.timestamp % 86400);
            //Reduce the staker count
            uints[_STAKE_COUNT] -= 1;
            //Decreases the stakerCount since the miner's stake is being slashed
            uint256 _transferAmount = uints[_STAKE_AMOUNT];
            if(balanceOf(disp.reportedMiner)  < uints[_STAKE_AMOUNT]){
                _transferAmount = balanceOf(disp.reportedMiner);
            }
            if (stakes.currentStatus == 4) {
                stakes.currentStatus = 5;
                _doTransfer(
                    disp.reportedMiner,
                    disp.reportingParty,
                    _transferAmount
                );
                stakes.currentStatus = 0;
            }
            for (uint256 i = 0; i < dispRounds; i++) {
                _id = disp.disputeUintVars[
                    keccak256(abi.encode(dispRounds - i))
                ];
                if (_id == 0) {
                    _id = origID;
                }
                Dispute storage disp2 = disputesById[_id];
                //transfer fee adjusted based on number of miners if the minerIndex is not 2(official value)
                _doTransfer(
                    address(this),
                    disp2.reportingParty,
                    disp2.disputeUintVars[_FEE]
                );
            }
        } else {
            if(uints[keccak256(abi.encodePacked(last.reportedMiner,"DisputeCount"))] == 1){
                stakes.currentStatus = 1;
            }
            TellorStorage.Request storage _request =
                requestDetails[disp.disputeUintVars[_REQUEST_ID]];
            if (disp.disputeUintVars[_MINER_SLOT] == 2) {
                //note we still don't put timestamp back into array (is this an issue? (shouldn't be))
                _request.finalValues[disp.disputeUintVars[_TIMESTAMP]] = disp
                    .disputeUintVars[_VALUE];
            }
            if (_request.inDispute[disp.disputeUintVars[_TIMESTAMP]] == true) {
                _request.inDispute[disp.disputeUintVars[_TIMESTAMP]] = false;
            }
            for (uint256 i = 0; i < dispRounds; i++) {
                _id = disp.disputeUintVars[
                    keccak256(abi.encode(dispRounds - i))
                ];
                if (_id != 0) {
                    last = disputesById[_id]; //handling if happens during an upgrade
                }
                _doTransfer(
                    address(this),
                    last.reportedMiner,
                    disputesById[_id].disputeUintVars[_FEE]
                );
            }
        }
        uints[keccak256(abi.encodePacked(last.reportedMiner,"DisputeCount"))]--;
        if (disp.disputeUintVars[_MINER_SLOT] == 2) {
            requestDetails[disp.disputeUintVars[_REQUEST_ID]].apiUintVars[
                _DISPUTE_COUNT
            ]--;
        }
    }

    /**
     * @dev Used during upgrade process to verify valid Tellor Contract
    */
    function verify() external virtual returns (uint256) {
        return CURRENT_VERSION;
    }

    /**
     * @dev Allows token holders to vote
     * @param _disputeId is the dispute id
     * @param _supportsDispute is the vote (true=the dispute has basis false = vote against dispute)
    */
    function vote(uint256 _disputeId, bool _supportsDispute) external {
        require(_disputeId <= uints[_DISPUTE_COUNT], "dispute does not exist");
        Dispute storage disp = disputesById[_disputeId];
        require(!disp.executed, "the dispute has already been executed");
        //Get the voteWeight or the balance of the user at the time/blockNumber the dispute began
        uint256 voteWeight = balanceOfAt(msg.sender, disp.disputeUintVars[_BLOCK_NUMBER]);
        //Require that the msg.sender has not voted
        require(disp.voted[msg.sender] != true, "Sender has already voted");
        //Require that the user had a balance >0 at time/blockNumber the dispute began
        require(voteWeight != 0, "User balance is 0");
        //ensures miners that are under dispute cannot vote
        require(
            stakerDetails[msg.sender].currentStatus != 3,
            "Miner is under dispute"
        );
        //Update user voting status to true
        disp.voted[msg.sender] = true;
        //Update the number of votes for the dispute
        disp.disputeUintVars[_NUM_OF_VOTES] += 1;
        //If the user supports the dispute increase the tally for the dispute by the voteWeight
        //otherwise decrease it
        if (_supportsDispute) {
            disp.tally = disp.tally.add(int256(voteWeight));
        } else {
            disp.tally = disp.tally.sub(int256(voteWeight));
        }
        //Let the network kblock.timestamp the user has voted on the dispute and their casted vote
        emit Voted(_disputeId, _supportsDispute, msg.sender, voteWeight);
    }

    /**
     * @dev Internal function with round checking logic from beginDispute/ proposeFork
     * @param _disputeId new dispute ID of round
     * @param _origId original ID of the hash
    */
    function _updateLastId(uint _disputeId,uint _origId) internal returns(uint256 _dispRounds){
        _dispRounds = disputesById[_origId].disputeUintVars[_DISPUTE_ROUNDS] + 1;
        disputesById[_origId].disputeUintVars[_DISPUTE_ROUNDS] = _dispRounds;
        disputesById[_origId].disputeUintVars[
            keccak256(abi.encode(_dispRounds))
        ] = _disputeId;
        if (_disputeId != _origId) {
            uint256 _lastId =
            disputesById[_origId].disputeUintVars[keccak256(abi.encode(_dispRounds - 1))];
            require(
                disputesById[_lastId].disputeUintVars[_MIN_EXECUTION_DATE] <=
                    block.timestamp,
                "Dispute is already open"
            );
            if (disputesById[_lastId].executed) {
                require(
                    block.timestamp - disputesById[_lastId].disputeUintVars[_TALLY_DATE] <=
                        1 days,
                    "Time for voting haven't elapsed"
                );
            }
        }
    }
    
    /**
     * @dev Used during upgrade process to verify valid Tellor Contract
    */
    function _verify(address _newTellor) internal {
        (bool success, bytes memory data) =
            address(_newTellor).call(
                abi.encodeWithSelector(0xfc735e99, "") //verify() signature
            );
        require(
            success && abi.decode(data, (uint256)) > CURRENT_VERSION, //we could enforce versioning through this return value, but we're almost in the size limit.
            "new tellor is invalid"
        );
    }

}
