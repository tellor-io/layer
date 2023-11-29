// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

interface IGovernance{
    enum VoteResult {FAILED,PASSED,INVALID}
    function setApprovedFunction(bytes4 _func, bool _val) external;
    function beginDispute(bytes32 _queryId,uint256 _timestamp) external;
    function delegate(address _delegate) external;
    function delegateOfAt(address _user, uint256 _blockNumber) external view returns (address);
    function executeVote(uint256 _disputeId) external;
    function proposeVote(address _contract,bytes4 _function, bytes calldata _data, uint256 _timestamp) external;
    function tallyVotes(uint256 _disputeId) external;
    function updateMinDisputeFee() external;
    function verify() external pure returns(uint);
    function vote(uint256 _disputeId, bool _supports, bool _invalidQuery) external;
    function voteFor(address[] calldata _addys,uint256 _disputeId, bool _supports, bool _invalidQuery) external;
    function getDelegateInfo(address _holder) external view returns(address,uint);
    function isApprovedGovernanceContract(address _contract) external view returns(bool);
    function isFunctionApproved(bytes4 _func) external view returns(bool);
    function getVoteCount() external view returns(uint256);
    function getVoteRounds(bytes32 _hash) external view returns(uint256[] memory);
    function getVoteInfo(uint256 _disputeId) external view returns(bytes32,uint256[8] memory,bool[2] memory,VoteResult,bytes memory,bytes4,address[2] memory);
    function getDisputeInfo(uint256 _disputeId) external view returns(uint256,uint256,bytes memory, address);
    function getOpenDisputesOnId(uint256 _queryId) external view returns(uint256);
    function didVote(uint256 _disputeId, address _voter) external view returns(bool);
    function getVoteTallyByAddress(address _voter) external view returns (uint256);
    //testing
    function testMin(uint256 a, uint256 b) external pure returns (uint256);
}
