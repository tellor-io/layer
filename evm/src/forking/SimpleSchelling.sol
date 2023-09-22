// SPDX-License-Identifier: MIT
pragma solidity 0.8.3;

import {IBridgeProxy} from "../interfaces/IBridgeProxy.sol";

contract SimpleSchelling {
    IBridgeProxy public bridgeProxy;
    uint256 public minInitAmount;
    uint256 public proposalCount;
    uint256 public submissionPeriod; // period after pausing bridge to submit implementation
    uint256 public extensionPeriod; // amount of time added to expiration time when outcome changes
    address public owner;
    bool public requireOwnerSignature;

    mapping(uint256 => ForkProposal) public proposals;

    enum ProposalOutcome {
        FOR,
        AGAINST,
        INVALID
    }

    struct ForkProposal {
        address initiator;
        address proposedImplementation;
        uint256 amountFor;
        uint256 amountAgainst;
        uint256 submissionDeadline;
        uint256 expirationTime;
        bool executed;
        bool ownerSigned;
        ProposalOutcome outcome;
        mapping(address => Vote) votesFor;
        mapping(address => Vote) votesAgainst;
    }

    struct Vote {
        uint256 amount;
    }

    constructor(
        address _bridgeProxy,
        uint256 _minInitAmount,
        uint256 _submissionPeriod,
        uint256 _extensionPeriod,
        bool _requireOwnerSignature,
        address _owner
    ) {
        bridgeProxy = IBridgeProxy(_bridgeProxy);
        minInitAmount = _minInitAmount;
        submissionPeriod = _submissionPeriod;
        extensionPeriod = _extensionPeriod;
        requireOwnerSignature = _requireOwnerSignature;
        owner = _owner;
    }

    function pauseBridge() public payable {
        require(msg.value >= minInitAmount, "insufficient amount");
        require(!bridgeProxy.paused(), "bridge already paused");
        ForkProposal storage _proposal = proposals[proposalCount];
        uint256 _burnAmount = minInitAmount / 10;
        _proposal.amountFor = msg.value - _burnAmount;
        _proposal.submissionDeadline = block.timestamp + submissionPeriod;
        _proposal.expirationTime =
            block.timestamp +
            extensionPeriod +
            submissionPeriod;
        _proposal.votesFor[msg.sender].amount += msg.value - _burnAmount;
        proposalCount++;
        payable(address(0)).transfer(_burnAmount);
        if (!requireOwnerSignature) {
            bridgeProxy.pauseBridge();
        }
    }

    function ownerPauseBridge(uint256 _proposalId) public {
        require(msg.sender == owner, "not owner");
        require(requireOwnerSignature, "owner signature not required");
        require(!bridgeProxy.paused(), "bridge already paused");
        ForkProposal storage _proposal = proposals[_proposalId];
        require(
            _proposal.submissionDeadline > block.timestamp,
            "submission period expired"
        );
        bridgeProxy.pauseBridge();
    }

    function transferOwnership(address _newOwner) public {
        require(msg.sender == owner, "not owner");
        owner = _newOwner;
    }

    function throwAwayOwnership() public {
        require(msg.sender == owner, "not owner");
        owner = address(0);
        requireOwnerSignature = false;
    }

    function submitImplementation(
        uint256 _proposalId,
        address _proposedImplementation
    ) public {
        ForkProposal storage _proposal = proposals[_proposalId];
        require(_proposal.initiator == msg.sender, "not initiator");
        require(
            _proposal.submissionDeadline > block.timestamp,
            "proposal expired"
        );
        _proposal.proposedImplementation = _proposedImplementation;
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
        // if outcome changed and past submission deadline, reset expiration time
        if (
            _forInLead &&
            _proposal.amountAgainst >= _proposal.amountFor &&
            _proposal.submissionDeadline < block.timestamp
        ) {
            _proposal.expirationTime = block.timestamp + extensionPeriod;
        } else if (
            !_forInLead &&
            _proposal.amountFor > _proposal.amountAgainst &&
            _proposal.submissionDeadline < block.timestamp
        ) {
            _proposal.expirationTime = block.timestamp + extensionPeriod;
        }
    }

    function executeProposal(uint256 _proposalId) public {
        ForkProposal storage _proposal = proposals[_proposalId];
        require(
            _proposal.submissionDeadline < block.timestamp,
            "submission period not expired"
        );
        require(!_proposal.executed, "already executed");
        if (
            (requireOwnerSignature && _proposal.ownerSigned) ||
            !requireOwnerSignature
        ) {
            require(
                _proposal.expirationTime < block.timestamp,
                "proposal not expired"
            );
            if (
                _proposal.amountFor > _proposal.amountAgainst &&
                _proposal.proposedImplementation != address(0)
            ) {
                bridgeProxy.updateImplementation(
                    _proposal.proposedImplementation
                );
                _proposal.outcome = ProposalOutcome.FOR;
            } else {
                _proposal.outcome = ProposalOutcome.AGAINST;
            }
        } else {
            _proposal.outcome = ProposalOutcome.INVALID;
        }
        _proposal.executed = true;
        bridgeProxy.unpauseBridge();
    }

    function claim(uint256 _proposalId) public {
        ForkProposal storage _proposal = proposals[_proposalId];
        require(_proposal.executed, "not executed");

        uint256 _payoutAmount;
        if (_proposal.outcome == ProposalOutcome.INVALID) {
            _payoutAmount =
                _proposal.votesFor[msg.sender].amount +
                _proposal.votesAgainst[msg.sender].amount;
            _proposal.votesFor[msg.sender].amount = 0;
            _proposal.votesAgainst[msg.sender].amount = 0;
        } else if (_proposal.outcome == ProposalOutcome.FOR) {
            _payoutAmount =
                (_proposal.votesFor[msg.sender].amount *
                    _proposal.amountAgainst) /
                _proposal.amountFor +
                _proposal.votesFor[msg.sender].amount;
            _proposal.votesFor[msg.sender].amount = 0;
        } else {
            _payoutAmount =
                (_proposal.votesAgainst[msg.sender].amount *
                    _proposal.amountFor) /
                _proposal.amountAgainst +
                _proposal.votesAgainst[msg.sender].amount;
            _proposal.votesAgainst[msg.sender].amount = 0;
        }
        require(_payoutAmount > 0, "no vote");
        payable(msg.sender).transfer(_payoutAmount);
    }

    // Getters

    function getProposal(
        uint256 _proposalId
    )
        public
        view
        returns (
            address,
            address,
            uint256,
            uint256,
            uint256,
            uint256,
            bool,
            bool,
            ProposalOutcome
        )
    {
        ForkProposal storage _proposal = proposals[_proposalId];
        return (
            _proposal.initiator,
            _proposal.proposedImplementation,
            _proposal.amountFor,
            _proposal.amountAgainst,
            _proposal.submissionDeadline,
            _proposal.expirationTime,
            _proposal.executed,
            _proposal.ownerSigned,
            _proposal.outcome
        );
    }

    function getVotes(
        uint256 _proposalId,
        address _voter
    ) public view returns (uint256, uint256) {
        ForkProposal storage _proposal = proposals[_proposalId];
        return (
            _proposal.votesFor[_voter].amount,
            _proposal.votesAgainst[_voter].amount
        );
    }
}
