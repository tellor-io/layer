// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import { ITellorFlex } from "../interfaces/ITellorFlex.sol";
import { ITellorMaster } from "../interfaces/ITellorMaster.sol";

/// @title LayerTransition.
/// @dev The contract that enables users of really old tellor to keep using it (e.g. Liquity)
/// by forwarding calls to the Ethereum oracle contract
/// also disables all further changes of the oracle address for time based rewards
contract LayerTransition {
    /*Storage*/
    bytes32 updateOracleQueryId = keccak256(abi.encode("TellorOracleAddress", abi.encode(bytes(""))));
    ITellorMaster public token;
    ITellorFlex public tellorFlex;

    /*Functions*/
    /// @notice constructor
    /// @param _tellorFlex address of current tellor360 oracle contract
    /// @param _token address of the tellor token (tellorMaster)
    constructor(address _tellorFlex, address _token) {
        tellorFlex = ITellorFlex(_tellorFlex);
        token = ITellorMaster(_token);
    }

    /// @notice this is needed because it's called when calling mintToOracle.  We hijack it to keep it in the bridge
    /// @param _amount the amount of staking rewards to add to the token contract
    function addStakingRewards(uint256 _amount) external {
        token.transferFrom(msg.sender, address(this), _amount);
    }

    /// @notice This forwards getDataBefore calls to the old tellorFlex
    /// we're hijacking it a bit to disable further oracle updates
    /// @param _queryId queryId of interest
    /// @param _timestamp timestamp you want data to be older than
    function getDataBefore(bytes32 _queryId, uint256 _timestamp) external view returns(
        bool _ifRetrieve,
        bytes memory _value,
        uint256 _timestampRetrieved
    ) {
        if (_queryId == updateOracleQueryId) {
            return (true, abi.encode(address(this)), block.timestamp);
        }
        return tellorFlex.getDataBefore(_queryId, _timestamp);
    }

    /// @notice This forwards getIndexForDataBefore calls to the old tellorFlex
    /// @param _queryId queryId of interest
    /// @param _timestamp timestamp you want data for
    function getIndexForDataBefore(bytes32 _queryId, uint256 _timestamp) external view returns(bool _found, uint256 _index) {
        return tellorFlex.getIndexForDataBefore(_queryId, _timestamp);
    }

    /// @notice This forwards getNewValueCountbyQueryId calls to the old tellorFlex
    /// @param _queryId queryId of interest
    function getNewValueCountbyQueryId(bytes32 _queryId) external view returns(uint256) {
        return tellorFlex.getNewValueCountbyQueryId(_queryId);
    }

    /// @notice This forwards getReporterbyTimestamp calls to the old tellorFlex
    /// @param _queryId queryId of interest
    /// @param _timestamp timestamp you want data for
    function getReporterByTimestamp(bytes32 _queryId, uint256 _timestamp) external view returns(address) {
        return tellorFlex.getReporterByTimestamp(_queryId, _timestamp);
    }

    /// @notice This forwards getTimestampbyQueryIdandIndex calls to the old tellorFlex
    /// @param _queryId queryId of interest
    /// @param _index index you want data for
    function getTimestampbyQueryIdandIndex(bytes32 _queryId, uint256 _index) external view returns(uint256) {
        return tellorFlex.getTimestampbyQueryIdandIndex(_queryId, _index);
    }

    /// @notice This forwards getTimeOfLastNewValue calls to the old tellorFlex
    function getTimeOfLastNewValue() external view returns(uint256) {
        return tellorFlex.getTimeOfLastNewValue();
    }

    /// @notice This forwards isInDispute calls to the old tellorFlex
    /// @param _queryId queryId of interest
    /// @param _timestamp timestamp you want data for
    function isInDispute(bytes32 _queryId, uint256 _timestamp) external view returns(bool) {
        return tellorFlex.isInDispute(_queryId, _timestamp);
    }

    /**
     * @dev Retrieve value from oracle based on timestamp
     * @param _queryId being requested
     * @param _timestamp to retrieve data/value from
     * @return bytes value for timestamp submitted
     */
    function retrieveData(bytes32 _queryId, uint256 _timestamp) external view returns(bytes memory) {
        return tellorFlex.retrieveData(_queryId, _timestamp);
    }

    /// @notice This returns a big number.  Necessary for upgrading the contract
    function verify() external pure returns (uint256) {
        return 9999;
    }
}

