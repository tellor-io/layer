// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import "./oldContracts/contracts/TellorVars.sol";
import "./oldContracts/contracts/interfaces/IOracle.sol";
import "./oldContracts/contracts/tellor3/TellorStorage.sol";

/**
 @author Tellor Inc.
 @title NewTransition
* @dev The Transition contract links to the Oracle contract and
* allows parties (like Liquity) to continue to use the master
* address to access values which use legacy query IDs (request IDs). 
*/
contract NewTransition is TellorStorage, TellorVars {
    // Functions
    //Getters
    /**
     * @dev Allows Tellor to read data from the addressVars mapping
     * @param _data is the keccak256("_VARIABLE_NAME") of the variable that is being accessed.
     * These are examples of how the variables are saved within other functions:
     * addressVars[keccak256("_OWNER")]
     * addressVars[keccak256("_TELLOR_CONTRACT")]
     * @return address of the requested variable
     */
    function getAddressVars(bytes32 _data) external view returns (address) {
        return addresses[_data];
    }

    /**
     * @dev Returns the latest value for a specific request ID.
     * @param _requestId the requestId to look up
     * @return uint256 the latest value of the request ID
     * @return bool whether or not the value was successfully retrieved
     */
    function getLastNewValueById(uint256 _requestId)
        external
        view
        returns (uint256, bool)
    {
        uint256 _count = getNewValueCountbyRequestId(_requestId);
        if (_count == 0) {
            return (0, false);
        }
        uint256 _latestTimestamp = getTimestampbyRequestIDandIndex(
            _requestId,
            _count - 1
        );
        return (retrieveData(_requestId, _latestTimestamp), true);
    }

    /**
     * @dev Function is solely for the parachute contract
     */
    function getNewCurrentVariables()
        external
        view
        returns (
            bytes32 _c,
            uint256[5] memory _r,
            uint256 _diff,
            uint256 _tip
        )
    {
        _r = [uint256(1), uint256(1), uint256(1), uint256(1), uint256(1)];
        _diff = 0;
        _tip = 0;
        _c = keccak256(
            abi.encode(
                IOracle(addresses[_ORACLE_CONTRACT]).getTimeOfLastNewValue()
            )
        );
    }

    /**
     * @dev Counts the number of values that have been submitted for the requestId.
     * @param _requestId the requestId to look up
     * @return uint256 count of the number of values received for the requestId
     */
    function getNewValueCountbyRequestId(uint256 _requestId)
        public
        view
        returns (uint256)
    {
        (bytes32 _queryId, ) = _getQueryIdAndDecimals(_requestId);
        IOracle _oracle = IOracle(addresses[_ORACLE_CONTRACT]);
        // try the new oracle first
        try _oracle.getNewValueCountbyQueryId(_queryId) returns (
            uint256 _valueCount
        ) {
            if (_valueCount == 0) {
                return 0;
            }
            // if last value is disputed, subtract 1 from the count until a non-disputed value is found
            uint256 _timestamp = _oracle.getTimestampbyQueryIdandIndex(
                _queryId,
                _valueCount - 1
            );
            while (
                _oracle.isInDispute(_queryId, _timestamp) &&
                _valueCount > 1
            ) {
                _valueCount--;
                _timestamp = _oracle.getTimestampbyQueryIdandIndex(
                    _queryId,
                    _valueCount - 1
                );
            }
            if (
                _valueCount == 1 &&
                _oracle.isInDispute(_queryId, _timestamp)
            ) {
                return 0;
            }
            return _valueCount;
        } catch {
            return
                IOracle(addresses[_ORACLE_CONTRACT]).getTimestampCountById(
                    bytes32(_requestId)
                );
        }
    }

    /**
     * @dev Gets the timestamp for the value based on its index
     * @param _requestId is the requestId to look up
     * @param _index is the value index to look up
     * @return uint256 timestamp
     */
    function getTimestampbyRequestIDandIndex(uint256 _requestId, uint256 _index)
        public
        view
        returns (uint256)
    {
        (bytes32 _queryId, ) = _getQueryIdAndDecimals(_requestId);
        try
            IOracle(addresses[_ORACLE_CONTRACT]).getTimestampbyQueryIdandIndex(
                _queryId,
                _index
            )
        returns (uint256 _val) {
            if(_requestId == 1 && _val > block.timestamp - 15 minutes) {
                ( , , _val) = IOracle(addresses[_ORACLE_CONTRACT]).getDataBefore(_queryId, block.timestamp - 15 minutes);
            }
            return _val;
        } catch {
            return
                IOracle(addresses[_ORACLE_CONTRACT]).getReportTimestampByIndex(
                    bytes32(_requestId),
                    _index
                );
        }
    }

    /**
     * @dev Getter for the variables saved under the TellorStorageStruct uints variable
     * @param _data the variable to pull from the mapping. _data = keccak256("_VARIABLE_NAME")
     * where variable_name is the variables/strings used to save the data in the mapping.
     * The variables names in the TellorVariables contract
     * @return uint256 of specified variable
     */
    function getUintVar(bytes32 _data) external view returns (uint256) {
        return uints[_data];
    }

    /**
     * @dev Getter for if the party is migrated
     * @param _addy address of party
     * @return bool if the party is migrated
     */
    function isMigrated(address _addy) external view returns (bool) {
        return migrated[_addy];
    }

    /**
     * @dev Retrieve value from oracle based on timestamp
     * @param _requestId being requested
     * @param _timestamp to retrieve data/value from
     * @return uint256 value for timestamp submitted
     */
    function retrieveData(uint256 _requestId, uint256 _timestamp)
        public
        view
        returns (uint256)
    {
        (bytes32 _queryId, uint256 _decimalsAdjustment) = _getQueryIdAndDecimals(
            _requestId
        );
        try
            IOracle(addresses[_ORACLE_CONTRACT]).getValueByTimestamp(
                bytes32(_requestId),
                _timestamp
            )
        returns (bytes memory _val) {
            return _sliceUint(_val);
        } catch {
            bytes memory _val;
            if (_requestId == 1) {
                (, _val, ) = IOracle(addresses[_ORACLE_CONTRACT])
                    .getDataBefore(_queryId, block.timestamp - 15 minutes);
            } else {
                 _val = IOracle(addresses[_ORACLE_CONTRACT])
                .retrieveData(_queryId, _timestamp);
            }
            return (_sliceUint(_val) / (10**_decimalsAdjustment));
        }
    }



    // Internal functions
    /**
     * @dev Utilized to help slice a bytes variable into a uint
     * @param _b is the bytes variable to be sliced
     * @return _number of the sliced uint256
     */
    function _sliceUint(bytes memory _b)
        internal
        pure
        returns (uint256 _number)
    {
        for (uint256 _i = 0; _i < _b.length; _i++) {
            _number = _number * 2**8;
            _number = _number + uint8(_b[_i]);
        }
    }

    function _getQueryIdAndDecimals(uint256 _requestId) internal pure returns (bytes32, uint256) {
        bytes32 _queryId;
        uint256 _decimalsAdjustment;
        if(_requestId == 1) {
            _queryId = 0x83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992; // SpotPrice(eth, usd)
            _decimalsAdjustment = 12;
        } else if(_requestId == 10) {
            _queryId = 0x0d12ad49193163bbbeff4e6db8294ced23ff8605359fd666799d4e25a3aa0e3a; // AmpleforthCustomSpotPrice(0x)
            _decimalsAdjustment = 0;
        } else if(_requestId == 41) {
            _queryId = 0x612ec1d9cee860bb87deb6370ed0ae43345c9302c085c1dfc4c207cbec2970d7; // AmpleforthUSPCE(0x)
            _decimalsAdjustment = 0;
        } else {
            _queryId = bytes32(_requestId);
            _decimalsAdjustment = 0;
        }
        return(_queryId, _decimalsAdjustment);
    }
}
