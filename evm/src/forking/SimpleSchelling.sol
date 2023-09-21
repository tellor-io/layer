// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import { IBridgeProxy } from "../interfaces/IBridgeProxy.sol";

contract SimpleSchelling {
    IBridgeProxy public bridgeProxy;
    uint256 public minInitAmount;
    uint256 public proposalCount;
    uint256 public extensionPeriod;

    mapping(uint256 => ForkProposal) public proposals;

    struct ForkProposal {
        address proposedImplementation;
        uint256 amountFor;
        uint256 amountAgainst;
        uint256 expirationTime;
        bool executed;
        mapping(address => Vote) votesFor;
        mapping(address => Vote) votesAgainst;
    }

    struct Vote {
        uint256 amount;
        bool claimed;
    }

    constructor(address _bridgeProxy, uint256 _minInitAmount, uint256 _extensionPeriod) {
        bridgeProxy = IBridgeProxy(_bridgeProxy);
        minInitAmount = _minInitAmount;
        extensionPeriod = _extensionPeriod;
    }

    function proposeFork(address _proposedImplementation) public payable {
        require(msg.value >= minInitAmount, "insufficient amount");
        ForkProposal storage _proposal = proposals[proposalCount];
        _proposal.proposedImplementation = _proposedImplementation;
        _proposal.amountFor = msg.value;
        _proposal.expirationTime = block.timestamp + extensionPeriod;
        _proposal.votesFor[msg.sender].amount += msg.value;
        proposalCount++;
    }

    function vote(uint256 _proposalId, bool _for) public payable {
        require(msg.value > 0, "insufficient amount");
        ForkProposal storage _proposal = proposals[_proposalId];
        require(_proposal.expirationTime > block.timestamp, "proposal expired");
        // get current outcome before vote
        bool _forInLead = _proposal.amountFor > _proposal.amountAgainst;
        if (_for) {
            _proposal.amountFor += msg.value;
            _proposal.votesFor[msg.sender].amount += msg.value;
        } else {
            _proposal.amountAgainst += msg.value;
            _proposal.votesAgainst[msg.sender].amount += msg.value;
        }
        // if outcome changed, reset expiration time
        if (_forInLead && _proposal.amountAgainst >= _proposal.amountFor) {
            _proposal.expirationTime = block.timestamp + extensionPeriod;
        } else if (!_forInLead && _proposal.amountFor > _proposal.amountAgainst) {
            _proposal.expirationTime = block.timestamp + extensionPeriod;
        }
    }

    function executeFork(uint256 _proposalId) public {
        ForkProposal storage _proposal = proposals[_proposalId];
        require(_proposal.expirationTime < block.timestamp, "proposal not expired");
        require(!_proposal.executed, "already executed");
        _proposal.executed = true;
        if(_proposal.amountFor > _proposal.amountAgainst) {
            bridgeProxy.updateImplementation(_proposal.proposedImplementation);
        }
    }

    function claim(uint256 _proposalId) public {
        ForkProposal storage _proposal = proposals[_proposalId];
        require(_proposal.expirationTime < block.timestamp, "proposal not expired");
        require(_proposal.executed, "not executed");
        Vote storage _vote;
        bool _proposalPassed;
        if(_proposal.amountFor > _proposal.amountAgainst) {
            _vote = _proposal.votesFor[msg.sender];
            _proposalPassed = true;
        } else {
            _vote = _proposal.votesAgainst[msg.sender];
        }
        require(_vote.amount > 0, "no vote");
        require(!_vote.claimed, "already claimed");
        _vote.claimed = true;
        uint256 _payoutAmount;
        if(_proposalPassed) {
            _payoutAmount = _vote.amount * _proposal.amountAgainst / _proposal.amountFor + _vote.amount;
        } else {
            _payoutAmount = _vote.amount * _proposal.amountFor / _proposal.amountAgainst + _vote.amount;
        }
        payable(msg.sender).transfer(_payoutAmount);
    }
}