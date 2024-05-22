// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import "./tellor3/TellorStorage.sol";
import "./TellorVars.sol";
import "./interfaces/IOracle.sol";
import "./interfaces/IController.sol";

/**
 @author Tellor Inc.
 @title Transition
* @dev The Transition contract links to the Oracle contract and
* allows parties (like Liquity) to continue to use the master
* address to access values. All parties should be reading values
* through this address
*/
contract Transition is TellorStorage, TellorVars {
    // Functions
    /**
     * @dev Saves new Tellor contract addresses. Available to init function after fork vote
     * @param _governance is the address of the Governance contract
     * @param _oracle is the address of the Oracle contract
     * @param _treasury is the address of the Treasury contract
     */
    constructor(
        address _governance,
        address _oracle,
        address _treasury
    ) {
        require(_governance != address(0), "must set governance address");
        addresses[_GOVERNANCE_CONTRACT] = _governance;
        addresses[_ORACLE_CONTRACT] = _oracle;
        addresses[_TREASURY_CONTRACT] = _treasury;
    }

    /**
     * @dev Runs once Tellor is migrated over. Changes the underlying storage.
     */
    function init() external {
        require(
            addresses[_GOVERNANCE_CONTRACT] == address(0),
            "Only good once"
        );
        // Set state amount, switch time, and minimum dispute fee
        uints[_STAKE_AMOUNT] = 100e18;
        uints[_SWITCH_TIME] = block.timestamp;
        uints[_MINIMUM_DISPUTE_FEE] = 10e18;
        // Define contract addresses
        Transition _controller = Transition(addresses[_TELLOR_CONTRACT]);
        addresses[_GOVERNANCE_CONTRACT] = _controller.addresses(
            _GOVERNANCE_CONTRACT
        );
        addresses[_ORACLE_CONTRACT] = _controller.addresses(_ORACLE_CONTRACT);
        addresses[_TREASURY_CONTRACT] = _controller.addresses(
            _TREASURY_CONTRACT
        );
        // Mint to oracle, parachute, and team operating grant contracts
        IController(TELLOR_ADDRESS).mint(
            addresses[_ORACLE_CONTRACT],
            105120e18
        );
        IController(TELLOR_ADDRESS).mint(
            0xAa304E98f47D4a6a421F3B1cC12581511dD69C55,
            105120e18
        );
        IController(TELLOR_ADDRESS).mint(
            0x83eB2094072f6eD9F57d3F19f54820ee0BaE6084,
            18201e18
        );
    }

    //Getters
    /**
     * @dev Allows users to access the number of decimals
     */
    function decimals() external pure returns (uint8) {
        return 18;
    }

    /**
     * @dev Allows Tellor to read data from the addressVars mapping
     * @param _data is the keccak256("variable_name") of the variable that is being accessed.
     * These are examples of how the variables are saved within other functions:
     * addressVars[keccak256("_owner")]
     * addressVars[keccak256("tellorContract")]
     * @return address of the requested variable
     */
    function getAddressVars(bytes32 _data) external view returns (address) {
        return addresses[_data];
    }

    /**
     * @dev Gets all dispute variables
     * @param _disputeId to look up
     * @return bytes32 hash of dispute
     * bool executed where true if it has been voted on
     * bool disputeVotePassed
     * bool isPropFork true if the dispute is a proposed fork
     * address of reportedMiner
     * address of reportingParty
     * address of proposedForkAddress
     * uint256 of requestId
     * uint256 of timestamp
     * uint256 of value
     * uint256 of minExecutionDate
     * uint256 of numberOfVotes
     * uint256 of blocknumber
     * uint256 of minerSlot
     * uint256 of quorum
     * uint256 of fee
     * int256 count of the current tally
     */
    function getAllDisputeVars(uint256 _disputeId)
        external
        view
        returns (
            bytes32,
            bool,
            bool,
            bool,
            address,
            address,
            address,
            uint256[9] memory,
            int256
        )
    {
        Dispute storage disp = disputesById[_disputeId];
        return (
            disp.hash,
            disp.executed,
            disp.disputeVotePassed,
            disp.isPropFork,
            disp.reportedMiner,
            disp.reportingParty,
            disp.proposedForkAddress,
            [
                disp.disputeUintVars[_REQUEST_ID],
                disp.disputeUintVars[_TIMESTAMP],
                disp.disputeUintVars[_VALUE],
                disp.disputeUintVars[_MIN_EXECUTION_DATE],
                disp.disputeUintVars[_NUM_OF_VOTES],
                disp.disputeUintVars[_BLOCK_NUMBER],
                disp.disputeUintVars[_MINER_SLOT],
                disp.disputeUintVars[keccak256("quorum")],
                disp.disputeUintVars[_FEE]
            ],
            disp.tally
        );
    }

    /**
     * @dev Gets id if a given hash has been disputed
     * @param _hash is the sha256(abi.encodePacked(_miners[2],_requestId,_timestamp));
     * @return uint256 disputeId
     */
    function getDisputeIdByDisputeHash(bytes32 _hash)
        external
        view
        returns (uint256)
    {
        return disputeIdByDisputeHash[_hash];
    }

    /**
     * @dev Checks for uint variables in the disputeUintVars mapping based on the disputeId
     * @param _disputeId is the dispute id;
     * @param _data the variable to pull from the mapping. _data = keccak256("variable_name") where variable_name is
     * the variables/strings used to save the data in the mapping. The variables names are
     * commented out under the disputeUintVars under the Dispute struct
     * @return uint256 value for the bytes32 data submitted
     */
    function getDisputeUintVars(uint256 _disputeId, bytes32 _data)
        external
        view
        returns (uint256)
    {
        return disputesById[_disputeId].disputeUintVars[_data];
    }

    /**
     * @dev Returns the latest value for a specific request ID.
     * @param _requestId the requestId to look up
     * @return uint256 of the value of the latest value of the request ID
     * @return bool of whether or not the value was successfully retrieved
     */
    function getLastNewValueById(uint256 _requestId)
        external
        view
        returns (uint256, bool)
    {
        // Try the new contract first
        uint256 _timeCount = IOracle(addresses[_ORACLE_CONTRACT])
            .getTimestampCountById(bytes32(_requestId));
        if (_timeCount != 0) {
            // If timestamps for the ID exist, there is value, so return the value
            return (
                retrieveData(
                    _requestId,
                    IOracle(addresses[_ORACLE_CONTRACT])
                        .getReportTimestampByIndex(
                            bytes32(_requestId),
                            _timeCount - 1
                        )
                ),
                true
            );
        } else {
            // Else, look at old value + timestamps since mining has not started
            Request storage _request = requestDetails[_requestId];
            if (_request.requestTimestamps.length != 0) {
                return (
                    retrieveData(
                        _requestId,
                        _request.requestTimestamps[
                            _request.requestTimestamps.length - 1
                        ]
                    ),
                    true
                );
            } else {
                return (0, false);
            }
        }
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
     * @dev Counts the number of values that have been submitted for the request.
     * @param _requestId the requestId to look up
     * @return uint256 count of the number of values received for the requestId
     */
    function getNewValueCountbyRequestId(uint256 _requestId)
        external
        view
        returns (uint256)
    {
        // Defaults to new one, but will give old value if new mining has not started
        uint256 _val = IOracle(addresses[_ORACLE_CONTRACT])
            .getTimestampCountById(bytes32(_requestId));
        if (_val > 0) {
            return _val;
        } else {
            return requestDetails[_requestId].requestTimestamps.length;
        }
    }

    /**
     * @dev Gets the timestamp for the value based on their index
     * @param _requestId is the requestId to look up
     * @param _index is the value index to look up
     * @return uint256 timestamp
     */
    function getTimestampbyRequestIDandIndex(uint256 _requestId, uint256 _index)
        external
        view
        returns (uint256)
    {
        // Try new contract first, but give old timestamp if new mining has not started
        try
            IOracle(addresses[_ORACLE_CONTRACT]).getReportTimestampByIndex(
                bytes32(_requestId),
                _index
            )
        returns (uint256 _val) {
            return _val;
        } catch {
            return requestDetails[_requestId].requestTimestamps[_index];
        }
    }

    /**
     * @dev Getter for the variables saved under the TellorStorageStruct uints variable
     * @param _data the variable to pull from the mapping. _data = keccak256("variable_name")
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
     * @dev Allows users to access the token's name
     */
    function name() external pure returns (string memory) {
        return "Tellor Tributes";
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
        if (_timestamp < uints[_SWITCH_TIME]) {
            return requestDetails[_requestId].finalValues[_timestamp];
        }
        return
            _sliceUint(
                IOracle(addresses[_ORACLE_CONTRACT]).getValueByTimestamp(
                    bytes32(_requestId),
                    _timestamp
                )
            );
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

    /**
     * @dev Allows Tellor X to fallback to the old Tellor if there are current open disputes
     * (or disputes on old Tellor values)
     */
    fallback() external {
        address _addr = 0x2754da26f634E04b26c4deCD27b3eb144Cf40582; // Main Tellor address (Harcode this in?)
        // Obtain function header from msg.data
        bytes4 _function;
        for (uint256 i = 0; i < 4; i++) {
            _function |= bytes4(msg.data[i] & 0xFF) >> (i * 8);
        }
        // Ensure that the function is allowed and related to disputes, voting, and dispute fees
        require(
            _function ==
                bytes4(
                    bytes32(keccak256("beginDispute(uint256,uint256,uint256)"))
                ) ||
                _function == bytes4(bytes32(keccak256("vote(uint256,bool)"))) ||
                _function ==
                bytes4(bytes32(keccak256("tallyVotes(uint256)"))) ||
                _function ==
                bytes4(bytes32(keccak256("unlockDisputeFee(uint256)"))),
            "function should be allowed"
        ); //should autolock out after a week (no disputes can begin past a week)
        // Calls the function in msg.data from main Tellor address
        (bool _result, ) = _addr.delegatecall(msg.data);
        assembly {
            returndatacopy(0, 0, returndatasize())
            switch _result
            // delegatecall returns 0 on error.
            case 0 {
                revert(0, returndatasize())
            }
            default {
                return(0, returndatasize())
            }
        }
    }

    // Internal
    /**
     * @dev Utilized to help slice a bytes variable into a uint
     * @param _b is the bytes variable to be sliced
     * @return _x of the sliced uint256
     */
    function _sliceUint(bytes memory _b) public pure returns (uint256 _x) {
        uint256 _number = 0;
        for (uint256 _i = 0; _i < _b.length; _i++) {
            _number = _number * 2**8;
            _number = _number + uint8(_b[_i]);
        }
        return _number;
    }
}
