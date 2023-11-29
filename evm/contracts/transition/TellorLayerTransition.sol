// SPDX-License-Identifier: MIT

import "../bridge/LayerLightClientBridge.sol";
import "../interfaces/IERC20.sol";

interface ITellorMaster {
    function mintToOracle() external;
}

pragma solidity 0.8.3;

contract TellorLayerTransition {
    IERC20 public token;
    address public tokenBridge;
    address public dataBridge;

    // oracle store
    mapping(bytes32 => mapping(uint256 => bool)) public isDisputed; //queryId -> timestamp -> value
    mapping(bytes32 => mapping(uint256 => address)) public reporterByTimestamp;
    mapping(bytes32 => uint256[]) public timestamps;
    mapping(bytes32 => mapping(uint256 => bytes)) public values; //queryId -> timestamp -> value

    

    constructor(address _token, address _tokenBridge, address _dataBridge) {
        token = IERC20(_token);
        tokenBridge = _tokenBridge;
        dataBridge = _dataBridge;
    }

    function verifyAndStore(
        uint256 _blockHeight,
        LayerLightClientBridge.Report calldata _report,
        uint256 _oracleHeight,
        LayerLightClientBridge.IAVLMerklePath[] calldata _merklePaths)
        external {
            // verify oracle data with data bridge
            // store data
            

            // values[_queryId][block.timestamp] = _value;
            // timestamps[_queryId].push(block.timestamp);
            // reporterByTimestamp[_queryId][block.timestamp] = msg.sender;
        }

    // needed for "mintToOracle" function
    function addStakingRewards(uint256 _amount) external {
        token.transferFrom(msg.sender, address(this), _amount);
    }

    function transferToTokenBridge() external {
        ITellorMaster(address(token)).mintToOracle();
        token.transfer(tokenBridge, token.balanceOf(address(this)));
    }

    // Getters
    /**
     * @dev Retrieves the latest value for the queryId before the specified timestamp
     * @param _queryId is the queryId to look up the value for
     * @param _timestamp before which to search for latest value
     * @return _ifRetrieve bool true if able to retrieve a non-zero value
     * @return _value the value retrieved
     * @return _timestampRetrieved the value's timestamp
     */
    function getDataBefore(bytes32 _queryId, uint256 _timestamp)
        external
        view
        returns (
            bool _ifRetrieve,
            bytes memory _value,
            uint256 _timestampRetrieved
        )
    {
        (bool _found, uint256 _index) = getIndexForDataBefore(
            _queryId,
            _timestamp
        );
        if (!_found) return (false, bytes(""), 0);
        _timestampRetrieved = getTimestampbyQueryIdandIndex(_queryId, _index);
        _value = values[_queryId][_timestampRetrieved];
        return (true, _value, _timestampRetrieved);
    }

    /**
     * @dev Retrieves latest array index of data before the specified timestamp for the queryId
     * @param _queryId is the queryId to look up the index for
     * @param _timestamp is the timestamp before which to search for the latest index
     * @return _found whether the index was found
     * @return _index the latest index found before the specified timestamp
     */
    // slither-disable-next-line calls-loop
    function getIndexForDataBefore(bytes32 _queryId, uint256 _timestamp)
        public
        view
        returns (bool _found, uint256 _index)
    {
        uint256 _count = getNewValueCountbyQueryId(_queryId);
        if (_count > 0) {
            uint256 _middle;
            uint256 _start = 0;
            uint256 _end = _count - 1;
            uint256 _time;
            //Checking Boundaries to short-circuit the algorithm
            _time = getTimestampbyQueryIdandIndex(_queryId, _start);
            if (_time >= _timestamp) return (false, 0);
            _time = getTimestampbyQueryIdandIndex(_queryId, _end);
            if (_time < _timestamp) {
                while (isInDispute(_queryId, _time) && _end > 0) {
                    _end--;
                    _time = getTimestampbyQueryIdandIndex(_queryId, _end);
                }
                if (_end == 0 && isInDispute(_queryId, _time)) {
                    return (false, 0);
                }
                return (true, _end);
            }
            //Since the value is within our boundaries, do a binary search
            while (true) {
                _middle = (_end - _start) / 2 + 1 + _start;
                _time = getTimestampbyQueryIdandIndex(_queryId, _middle);
                if (_time < _timestamp) {
                    //get immediate next value
                    uint256 _nextTime = getTimestampbyQueryIdandIndex(
                        _queryId,
                        _middle + 1
                    );
                    if (_nextTime >= _timestamp) {
                        if (!isInDispute(_queryId, _time)) {
                            // _time is correct
                            return (true, _middle);
                        } else {
                            // iterate backwards until we find a non-disputed value
                            while (
                                isInDispute(_queryId, _time) && _middle > 0
                            ) {
                                _middle--;
                                _time = getTimestampbyQueryIdandIndex(
                                    _queryId,
                                    _middle
                                );
                            }
                            if (_middle == 0 && isInDispute(_queryId, _time)) {
                                return (false, 0);
                            }
                            // _time is correct
                            return (true, _middle);
                        }
                    } else {
                        //look from middle + 1(next value) to end
                        _start = _middle + 1;
                    }
                } else {
                    uint256 _prevTime = getTimestampbyQueryIdandIndex(
                        _queryId,
                        _middle - 1
                    );
                    if (_prevTime < _timestamp) {
                        if (!isInDispute(_queryId, _prevTime)) {
                            // _prevTime is correct
                            return (true, _middle - 1);
                        } else {
                            // iterate backwards until we find a non-disputed value
                            _middle--;
                            while (
                                isInDispute(_queryId, _prevTime) && _middle > 0
                            ) {
                                _middle--;
                                _prevTime = getTimestampbyQueryIdandIndex(
                                    _queryId,
                                    _middle
                                );
                            }
                            if (
                                _middle == 0 && isInDispute(_queryId, _prevTime)
                            ) {
                                return (false, 0);
                            }
                            // _prevtime is correct
                            return (true, _middle);
                        }
                    } else {
                        //look from start to middle -1(prev value)
                        _end = _middle - 1;
                    }
                }
            }
        }
        return (false, 0);
    }

    /**
     * @dev Counts the number of values that have been submitted for a given ID
     * @param _queryId the ID to look up
     * @return uint256 count of the number of values received for the queryId
     */
    function getNewValueCountbyQueryId(bytes32 _queryId)
        public
        view
        returns (uint256)
    {
        return timestamps[_queryId].length;
    }

    /**
     * @dev Returns the reporter for a given timestamp and queryId
     * @param _queryId bytes32 version of the queryId
     * @param _timestamp uint256 timestamp of report
     * @return address of data reporter
     */
    function getReporterByTimestamp(bytes32 _queryId, uint256 _timestamp)
        external
        view
        returns (address)
    {
        return reporterByTimestamp[_queryId][_timestamp];
    }

    /**
     * @dev Gets the timestamp for the value based on their index
     * @param _queryId is the queryId to look up
     * @param _index is the value index to look up
     * @return uint256 timestamp
     */
    function getTimestampbyQueryIdandIndex(bytes32 _queryId, uint256 _index)
        public
        view
        returns (uint256)
    {
        uint256 _len = timestamps[_queryId].length;
        if (_len == 0 || _len <= _index) return 0;
        return timestamps[_queryId][_index];
    }

    /**
     * @dev Returns whether a given value is disputed
     * @param _queryId unique ID of the data feed
     * @param _timestamp timestamp of the value
     * @return bool whether the value is disputed
     */
    function isInDispute(bytes32 _queryId, uint256 _timestamp)
        public
        view
        returns (bool)
    {
        return isDisputed[_queryId][_timestamp];
    }

    /**
     * @dev Retrieves value from oracle based on queryId/timestamp
     * @param _queryId being requested
     * @param _timestamp to retrieve data/value from
     * @return bytes value for queryId/timestamp submitted
     */
    function retrieveData(bytes32 _queryId, uint256 _timestamp)
        external
        view
        returns (bytes memory)
    {
        return values[_queryId][_timestamp];
    }

    function verify() external pure returns (uint256) {
        return 9999;
    }
}