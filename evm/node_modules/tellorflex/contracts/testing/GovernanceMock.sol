// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import "../TellorFlex.sol";

contract GovernanceMock {
    TellorFlex public tellor;
    uint256 public voteCount; // total number of votes initiated
    mapping(address => uint256) private voteTallyByAddress; // mapping of addresses to the number of votes they have cast
    mapping(address => mapping(uint256 => bool)) private voted; // mapping of addresses to mapping of voteIds to whether they have voted

    function setTellorAddress(address _tellor) public {
        tellor = TellorFlex(_tellor);
    }

    function beginDisputeMock() public {
        voteCount++;
    }

    function voteMock(uint256 _disputeId) public {
        require(_disputeId > 0, "Dispute ID must be greater than 0");
        require(_disputeId <= voteCount, "Vote does not exist");
        require(!voted[msg.sender][_disputeId], "Address already voted");
        voteTallyByAddress[msg.sender]++;
        voted[msg.sender][_disputeId] = true;
    }

    function getVoteTallyByAddress(address _voter) public view returns (uint256) {
        return voteTallyByAddress[_voter];
    }

    function getVoteCount() public view returns (uint256) {
        return voteCount;
    }

    fallback() external payable {}
    receive() external payable {}

}
