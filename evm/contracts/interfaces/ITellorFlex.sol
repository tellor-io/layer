// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface ITellorFlex {
    function getDataBefore(bytes32 _queryId, uint256 _timestamp) external view returns (bool _ifRetrieve, bytes memory _value, uint256 _timestampRetrieved);
    function getIndexForDataBefore(bytes32 _queryId, uint256 _timestamp) external view returns (bool _found, uint256 _index);
    function getNewValueCountbyQueryId(bytes32 _queryId) external view returns (uint256);
    function getReporterByTimestamp(bytes32 _queryId, uint256 _timestamp) external view returns (address);
    function getTimestampbyQueryIdandIndex(bytes32 _queryId, uint256 _index) external view returns (uint256);
    function getTimeOfLastNewValue() external view returns (uint256);
    function isInDispute(bytes32 _queryId,uint256 _timestamp) external view returns (bool);
    function retrieveData(bytes32 _queryId, uint256 _timestamp) external view returns (bytes memory);
    function verify() external pure returns (uint256);
    function getNewValueCountbyQueryId() external view returns (uint256);
}
