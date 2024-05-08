// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import "./interfaces/IController.sol";
import "./TellorVars.sol";

/**
 @author Tellor Inc.
 @title Oracle
 @dev This is the Oracle contract which defines the functionality for the Tellor
 * oracle, where reporters submit values on chain and users can retrieve values.
*/
contract Oracle is TellorVars {
    // Storage
    uint256 public reportingLock = 12 hours; // amount of time before a reporter is able to submit a value again
    uint256 public timeBasedReward = 5e17; // time based reward for a reporter for successfully submitting a value
    uint256 public timeOfLastNewValue = block.timestamp; // time of the last new submitted value, originally set to the block timestamp
    uint256 public tipsInContract; // amount of tips within the contract
    mapping(bytes32 => Report) private reports; // mapping of query IDs to a report
    mapping(bytes32 => uint256) public tips; // mapping of query IDs to the amount of TRB they are tipped
    mapping(address => uint256) private reporterLastTimestamp; // mapping of reporter addresses to the timestamp of their last reported value
    mapping(address => uint256) private reportsSubmittedByAddress; // mapping of reporter addresses to the number of reports they've submitted
    mapping(address => uint256) private tipsByUser; // mapping of a user to the amount of tips they've paid

    // Structs
    struct Report {
        uint256[] timestamps; // array of all newValueTimestamps reported
        mapping(uint256 => uint256) timestampIndex; // mapping of timestamps to respective indices
        mapping(uint256 => uint256) timestampToBlockNum; // mapping of timestamp to block number
        mapping(uint256 => bytes) valueByTimestamp; // mapping of timestamps to values
        mapping(uint256 => address) reporterByTimestamp; // mapping of timestamps to reporters
    }

    // Events
    event ReportingLockChanged(uint256 _newReportingLock);
    event NewReport(
        bytes32 _queryId,
        uint256 _time,
        bytes _value,
        uint256 _reward,
        uint256 _nonce,
        bytes _queryData,
        address _reporter
    );
    event TimeBasedRewardsChanged(uint256 _newTimeBasedReward);
    event TipAdded(
        address indexed _user,
        bytes32 indexed _queryId,
        uint256 _tip,
        uint256 _totalTip,
        bytes _queryData
    );

    /**
     * @dev Changes reporting lock for reporters.
     * Note: this function is only callable by the Governance contract.
     * @param _newReportingLock is the new reporting lock.
     */
    function changeReportingLock(uint256 _newReportingLock) external {
        require(
            msg.sender ==
                IController(TELLOR_ADDRESS).addresses(_GOVERNANCE_CONTRACT),
            "Only governance contract can change reporting lock."
        );
        require(_newReportingLock < 8640000, "Invalid _newReportingLock value");
        reportingLock = _newReportingLock;
        emit ReportingLockChanged(_newReportingLock);
    }

    /**
     * @dev Changes time based reward for reporters.
     * Note: this function is only callable by the Governance contract.
     * @param _newTimeBasedReward is the new time based reward.
     */
    function changeTimeBasedReward(uint256 _newTimeBasedReward) external {
        require(
            msg.sender ==
                IController(TELLOR_ADDRESS).addresses(_GOVERNANCE_CONTRACT),
            "Only governance contract can change time based reward."
        );
        timeBasedReward = _newTimeBasedReward;
        emit TimeBasedRewardsChanged(_newTimeBasedReward);
    }

    /**
     * @dev Removes a value from the oracle.
     * Note: this function is only callable by the Governance contract.
     * @param _queryId is ID of the specific data feed
     * @param _timestamp is the timestamp of the data value to remove
     */
    function removeValue(bytes32 _queryId, uint256 _timestamp) external {
        require(
            msg.sender ==
                IController(TELLOR_ADDRESS).addresses(_GOVERNANCE_CONTRACT) ||
                msg.sender ==
                IController(TELLOR_ADDRESS).addresses(_ORACLE_CONTRACT),
            "caller must be the governance contract or the oracle contract"
        );
        Report storage rep = reports[_queryId];
        uint256 _index = rep.timestampIndex[_timestamp];
        // Shift all timestamps back to reflect deletion of value
        for (uint256 i = _index; i < rep.timestamps.length - 1; i++) {
            rep.timestamps[i] = rep.timestamps[i + 1];
            rep.timestampIndex[rep.timestamps[i]] -= 1;
        }
        // Delete and reset timestamp and value
        delete rep.timestamps[rep.timestamps.length - 1];
        rep.timestamps.pop();
        rep.valueByTimestamp[_timestamp] = "";
        rep.timestampIndex[_timestamp] = 0;
    }

    /**
     * @dev Allows a reporter to submit a value to the oracle
     * @param _queryId is ID of the specific data feed. Equals keccak256(_queryData) for non-legacy IDs
     * @param _value is the value the user submits to the oracle
     * @param _nonce is the current value count for the query id
     * @param _queryData is the data used to fulfill the data query
     */
    function submitValue(
        bytes32 _queryId,
        bytes calldata _value,
        uint256 _nonce,
        bytes memory _queryData
    ) external {
        Report storage rep = reports[_queryId];
        require(
            _nonce == rep.timestamps.length,
            "nonce must match timestamp index"
        );
        // Require reporter to abide by given reporting lock
        require(
            block.timestamp - reporterLastTimestamp[msg.sender] > reportingLock,
            "still in reporter time lock, please wait!"
        );
        require(
            address(this) ==
                IController(TELLOR_ADDRESS).addresses(_ORACLE_CONTRACT),
            "can only submit to current oracle contract"
        );
        require(
            _queryId == keccak256(_queryData) || uint256(_queryId) <= 100,
            "id must be hash of bytes data"
        );
        reporterLastTimestamp[msg.sender] = block.timestamp;
        IController _tellor = IController(TELLOR_ADDRESS);
        // Checks that reporter is not already staking TRB
        (uint256 _status, ) = _tellor.getStakerInfo(msg.sender);
        require(_status == 1, "Reporter status is not staker");
        // Check is in case the stake amount increases
        require(
            _tellor.balanceOf(msg.sender) >= _tellor.uints(_STAKE_AMOUNT),
            "balance must be greater than stake amount"
        );
        // Checks for no double reporting of timestamps
        require(
            rep.reporterByTimestamp[block.timestamp] == address(0),
            "timestamp already reported for"
        );
        // Update number of timestamps, value for given timestamp, and reporter for timestamp
        rep.timestampIndex[block.timestamp] = rep.timestamps.length;
        rep.timestamps.push(block.timestamp);
        rep.timestampToBlockNum[block.timestamp] = block.number;
        rep.valueByTimestamp[block.timestamp] = _value;
        rep.reporterByTimestamp[block.timestamp] = msg.sender;
        // Send tips + timeBasedReward to reporter of value, and reset tips for ID
        (uint256 _tip, uint256 _reward) = getCurrentReward(_queryId);
        tipsInContract -= _tip;
        if (_reward + _tip > 0) {
            _tellor.transfer(msg.sender, _reward + _tip);
        }
        tips[_queryId] = 0;
        // Update last oracle value and number of values submitted by a reporter
        timeOfLastNewValue = block.timestamp;
        reportsSubmittedByAddress[msg.sender]++;
        emit NewReport(
            _queryId,
            block.timestamp,
            _value,
            _tip + _reward,
            _nonce,
            _queryData,
            msg.sender
        );
    }

    /**
     * @dev Adds tips to incentivize reporters to submit values for specific data IDs.
     * @param _queryId is ID of the specific data feed
     * @param _tip is the amount to tip the given data ID
     * @param _queryData is required for IDs greater than 100, informs reporters how to fulfill request. See github.com/tellor-io/dataSpecs
     */
    function tipQuery(
        bytes32 _queryId,
        uint256 _tip,
        bytes memory _queryData
    ) external {
        // Require tip to be greater than 1 and be paid
        require(_tip > 1, "Tip should be greater than 1");
        require(
            IController(TELLOR_ADDRESS).approveAndTransferFrom(
                msg.sender,
                address(this),
                _tip
            ),
            "tip must be paid"
        );
        require(
            _queryId == keccak256(_queryData) ||
                uint256(_queryId) <= 100 ||
                msg.sender ==
                IController(TELLOR_ADDRESS).addresses(_GOVERNANCE_CONTRACT),
            "id must be hash of bytes data"
        );
        // Burn half the tip
        _tip = _tip / 2;
        IController(TELLOR_ADDRESS).burn(_tip);
        // Update total tip amount for user, data ID, and in total contract
        tips[_queryId] += _tip;
        tipsByUser[msg.sender] += _tip;
        tipsInContract += _tip;
        emit TipAdded(msg.sender, _queryId, _tip, tips[_queryId], _queryData);
    }

    //Getters
    /**
     * @dev Returns the block number at a given timestamp
     * @param _queryId is ID of the specific data feed
     * @param _timestamp is the timestamp to find the corresponding block number for
     * @return uint256 block number of the timestamp for the given data ID
     */
    function getBlockNumberByTimestamp(bytes32 _queryId, uint256 _timestamp)
        external
        view
        returns (uint256)
    {
        return reports[_queryId].timestampToBlockNum[_timestamp];
    }

    /**
     * @dev Calculates the current reward for a reporter given tips
     * and time based reward
     * @param _queryId is ID of the specific data feed
     * @return uint256 tips on given queryId
     * @return uint256 time based reward
     */
    function getCurrentReward(bytes32 _queryId)
        public
        view
        returns (uint256, uint256)
    {
        IController _tellor = IController(TELLOR_ADDRESS);
        uint256 _timeDiff = block.timestamp - timeOfLastNewValue;
        uint256 _reward = (_timeDiff * timeBasedReward) / 300; //.5 TRB per 5 minutes (should we make this upgradeable)
        if (_tellor.balanceOf(address(this)) < _reward + tipsInContract) {
            _reward = _tellor.balanceOf(address(this)) - tipsInContract;
        }
        return (tips[_queryId], _reward);
    }

    /**
     * @dev Returns the current value of a data feed given a specific ID
     * @param _queryId is the ID of the specific data feed
     * @return bytes memory of the current value of data
     */
    function getCurrentValue(bytes32 _queryId)
        external
        view
        returns (bytes memory)
    {
        return
            reports[_queryId].valueByTimestamp[
                reports[_queryId].timestamps[
                    reports[_queryId].timestamps.length - 1
                ]
            ];
    }

    /**
     * @dev Returns the reporting lock time, the amount of time a reporter must wait to submit again
     * @return uint256 reporting lock time
     */
    function getReportingLock() external view returns (uint256) {
        return reportingLock;
    }

    /**
     * @dev Returns the address of the reporter who submitted a value for a data ID at a specific time
     * @param _queryId is ID of the specific data feed
     * @param _timestamp is the timestamp to find a corresponding reporter for
     * @return address of the reporter who reported the value for the data ID at the given timestamp
     */
    function getReporterByTimestamp(bytes32 _queryId, uint256 _timestamp)
        external
        view
        returns (address)
    {
        return reports[_queryId].reporterByTimestamp[_timestamp];
    }

    /**
     * @dev Returns the timestamp of the reporter's last submission
     * @param _reporter is address of the reporter
     * @return uint256 timestamp of the reporter's last submission
     */
    function getReporterLastTimestamp(address _reporter)
        external
        view
        returns (uint256)
    {
        return reporterLastTimestamp[_reporter];
    }

    /**
     * @dev Returns the number of values submitted by a specific reporter address
     * @param _reporter is the address of a reporter
     * @return uint256 of the number of values submitted by the given reporter
     */
    function getReportsSubmittedByAddress(address _reporter)
        external
        view
        returns (uint256)
    {
        return reportsSubmittedByAddress[_reporter];
    }

    /**
     * @dev Returns the timestamp of a reported value given a data ID and timestamp index
     * @param _queryId is ID of the specific data feed
     * @param _index is the index of the timestamp
     * @return uint256 timestamp of the given queryId and index
     */
    function getReportTimestampByIndex(bytes32 _queryId, uint256 _index)
        external
        view
        returns (uint256)
    {
        return reports[_queryId].timestamps[_index];
    }

    /**
     * @dev Returns the time based reward for submitting a value
     * @return uint256 of time based reward
     */
    function getTimeBasedReward() external view returns (uint256) {
        return timeBasedReward;
    }

    /**
     * @dev Returns the number of timestamps/reports for a specific data ID
     * @param _queryId is ID of the specific data feed
     * @return uint256 of the number of timestamps/reports for the given data ID
     */
    function getTimestampCountById(bytes32 _queryId)
        external
        view
        returns (uint256)
    {
        return reports[_queryId].timestamps.length;
    }

    /**
     * @dev Returns the timestamp for the last value of any ID from the oracle
     * @return uint256 of timestamp of the last oracle value
     */
    function getTimeOfLastNewValue() external view returns (uint256) {
        return timeOfLastNewValue;
    }

    /**
     * @dev Returns the index of a reporter timestamp in the timestamp array for a specific data ID
     * @param _queryId is ID of the specific data feed
     * @param _timestamp is the timestamp to find in the timestamps array
     * @return uint256 of the index of the reporter timestamp in the array for specific ID
     */
    function getTimestampIndexByTimestamp(bytes32 _queryId, uint256 _timestamp)
        external
        view
        returns (uint256)
    {
        return reports[_queryId].timestampIndex[_timestamp];
    }

    /**
     * @dev Returns the amount of tips available for a specific query ID
     * @param _queryId is ID of the specific data feed
     * @return uint256 of the amount of tips added for the specific ID
     */
    function getTipsById(bytes32 _queryId) external view returns (uint256) {
        return tips[_queryId];
    }

    /**
     * @dev Returns the amount of tips made by a user
     * @param _user is the address of the user
     * @return uint256 of the amount of tips made by the user
     */
    function getTipsByUser(address _user) external view returns (uint256) {
        return tipsByUser[_user];
    }

    /**
     * @dev Returns the value of a data feed given a specific ID and timestamp
     * @param _queryId is the ID of the specific data feed
     * @param _timestamp is the timestamp to look for data
     * @return bytes memory of the value of data at the associated timestamp
     */
    function getValueByTimestamp(bytes32 _queryId, uint256 _timestamp)
        external
        view
        returns (bytes memory)
    {
        return reports[_queryId].valueByTimestamp[_timestamp];
    }

    /**
     * @dev Used during the upgrade process to verify valid Tellor Contracts
     */
    function verify() external pure returns (uint256) {
        return 9999;
    }
}
