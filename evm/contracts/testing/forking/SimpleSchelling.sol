// SPDX-License-Identifier: MIT
pragma solidity 0.8.22;

import {IBridgeProxy} from "../../interfaces/IBridgeProxy.sol";

/**
 @author Tellor Inc.
 @title SimpleSchelling
 @dev This contract provides a simple Schelling game for pausing and updating a Tellor Layer
 * Light Client Bridge contract in the event that Tellor Layer chain is attacked. Any user can pay
 * a high fee to begin the Schelling game. Then either the bridge is immediately paused or an optional 
 * second signature is required from a configured guardian address. The initiator has a period of time to
 * propose a new implementation address. Then anyone can pay to support or oppose the proposal. Once the voting
 * period expires, the bridge is unpaused, and if the proposal is supported by a majority of the tokens, the bridge 
 * implementation address is updated. The winning side gets their tokens refunded and splits the losing side's tokens.
*/
contract SimpleSchelling {
    // Storage
    IBridgeProxy public bridgeProxy; // bridge proxy contract
    uint256 public minInitAmount; // minimum amount of tokens required to initiate Schelling game. 10% is burned
    uint256 public proposalCount; // number of proposals
    uint256 public submissionPeriod; // period after pausing bridge to submit implementation
    uint256 public extensionPeriod; // amount of time added to expiration time when outcome changes
    address public guardian; // optional guardian to sign proposal to pause bridge
    bool public requireGuardianSignature; // whether or not guardian signature is required to pause bridge

    mapping(uint256 => ForkProposal) public proposals;

    // Structs
    struct ForkProposal {
        address initiator; // address of user who initiated Schelling game
        address proposedImplementation; // address of proposed bridge implementation
        uint256 amountFor; // amount of tokens in support of proposal
        uint256 amountAgainst; // amount of tokens against proposal
        uint256 submissionDeadline; // deadline for submitting implementation address
        uint256 expirationTime; // deadline for voting, extended when outcome changes
        bool executed; // whether or not proposal has been executed
        bool guardianSigned; // whether or not guardian has signed proposal
        ProposalOutcome outcome; // final outcome of proposal
        mapping(address => Vote) votesFor; // mapping of addresses to votes in support of proposal
        mapping(address => Vote) votesAgainst; // mapping of addresses to votes against proposal
    }

    struct Vote {
        uint256 amount;
    }

    enum ProposalOutcome {
        INVALID,
        FOR,
        AGAINST
    }

    // Functions
    /**
     * @dev Initializes system parameters
     * @param _bridgeProxy address of bridge proxy contract
     * @param _minInitAmount minimum amount of tokens required to initiate Schelling game. 10% is burned
     * @param _submissionPeriod maximum period after pausing bridge to submit implementation
     * @param _extensionPeriod total voting period, reset when outcome changes
     * @param _requireGuardianSignature whether or not guardian signature is required to pause bridge
     * @param _guardian optional guardian to sign proposal to pause bridge
     */
    constructor(
        address _bridgeProxy,
        uint256 _minInitAmount,
        uint256 _submissionPeriod,
        uint256 _extensionPeriod,
        bool _requireGuardianSignature,
        address _guardian
    ) {
        bridgeProxy = IBridgeProxy(_bridgeProxy);
        minInitAmount = _minInitAmount;
        submissionPeriod = _submissionPeriod;
        extensionPeriod = _extensionPeriod;
        requireGuardianSignature = _requireGuardianSignature;
        guardian = _guardian;
    }

    /**
     * @dev Claim tokens after proposal has been executed
     * @param _proposalId ID of proposal to claim tokens from
     */
    function claim(uint256 _proposalId) external {
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

    /**
     * @dev Execute proposal after voting period has expired
     * or after submission period has expired if guardian required but never signed
     * @param _proposalId ID of proposal to execute
     */
    function executeProposal(uint256 _proposalId) external {
        ForkProposal storage _proposal = proposals[_proposalId];
        require(
            _proposal.submissionDeadline < block.timestamp,
            "submission period not expired"
        );
        require(!_proposal.executed, "already executed");
        if (
            (requireGuardianSignature && _proposal.guardianSigned) ||
            !requireGuardianSignature
        ) {
            require(
                _proposal.expirationTime < block.timestamp,
                "proposal not expired"
            );
            if (
                _proposal.amountFor > _proposal.amountAgainst &&
                _proposal.proposedImplementation != address(0)
            ) {
                // if more tokens in support AND implementation submitted, update implementation
                bridgeProxy.updateImplementation(
                    _proposal.proposedImplementation
                );
                _proposal.outcome = ProposalOutcome.FOR;
            } else {
                _proposal.outcome = ProposalOutcome.AGAINST;
            }
            bridgeProxy.unpauseBridge();
        } else {
            // if guardian signature required and not signed, proposal is invalid
            _proposal.outcome = ProposalOutcome.INVALID;
        }
        _proposal.executed = true;
    }

    /**
     * @dev Guardian can sign proposal and pause bridge, only if required
     * @param _proposalId ID of proposal
     */
    function guardianPauseBridge(uint256 _proposalId) external {
        require(msg.sender == guardian, "not guardian");
        require(requireGuardianSignature, "guardian signature not required");
        require(!bridgeProxy.paused(), "bridge already paused");
        ForkProposal storage _proposal = proposals[_proposalId];
        require(
            _proposal.submissionDeadline > block.timestamp,
            "submission period expired"
        );
        _proposal.guardianSigned = true;
        bridgeProxy.pauseBridge();
    }

    /**
     * @dev Initiates Schelling game by paying minimum amount of tokens
     * @notice 10% of amount is burned
     */
    function initiateProposal() external payable {
        require(msg.value >= minInitAmount, "insufficient amount");
        require(!bridgeProxy.paused(), "bridge already paused");
        ForkProposal storage _proposal = proposals[proposalCount];
        uint256 _burnAmount = minInitAmount / 10;
        _proposal.initiator = msg.sender;
        _proposal.amountFor = msg.value - _burnAmount;
        _proposal.submissionDeadline = block.timestamp + submissionPeriod;
        _proposal.expirationTime =
            block.timestamp +
            extensionPeriod +
            submissionPeriod;
        _proposal.votesFor[msg.sender].amount += msg.value - _burnAmount;
        proposalCount++;
        payable(address(0)).transfer(_burnAmount);
        if (!requireGuardianSignature) {
            bridgeProxy.pauseBridge();
        }
    }

    /**
     * @dev Propose new implementation address after Schelling game initiated
     * @param _proposalId ID of proposal
     * @param _proposedImplementation address of proposed implementation
     */
    function submitImplementation(
        uint256 _proposalId,
        address _proposedImplementation
    ) external {
        ForkProposal storage _proposal = proposals[_proposalId];
        require(_proposal.initiator == msg.sender, "not initiator");
        require(_proposal.proposedImplementation == address(0), "already set");
        require(
            _proposal.submissionDeadline > block.timestamp,
            "proposal expired"
        );
        _proposal.proposedImplementation = _proposedImplementation;
    }

    /**
     * @dev Guardian can remove requirement for signature to pause bridge
     */
    function throwAwayGuardianship() external {
        require(msg.sender == guardian, "not guardian");
        guardian = address(0);
        requireGuardianSignature = false;
    }

    /**
     * @dev Transfer guardianship to new address
     * @param _newGuardian address of new guardian
     */
    function transferGuardianship(address _newGuardian) external {
        require(msg.sender == guardian, "not guardian");
        guardian = _newGuardian;
    }

    /**
     * @dev Allows a voter to change their vote before expiration
     * @notice Initiator cannot update vote, and 10% of amount is burned
     * @param _proposalId ID of proposal
     * @param _for whether or not to vote in support of proposal
     */
    function updateVote(uint256 _proposalId, bool _for) external {
        ForkProposal storage _proposal = proposals[_proposalId];
        require(
            msg.sender != _proposal.initiator,
            "initiator cannot update vote"
        );
        require(_proposal.expirationTime > block.timestamp, "proposal expired");
        uint256 _amount;
        // burn 10% so that griefing by flip flopping is expensive
        uint256 _burnAmount;
        // get current outcome before vote
        bool _forInLead = _proposal.amountFor > _proposal.amountAgainst;
        if (_for) {
            _burnAmount = _proposal.votesAgainst[msg.sender].amount / 10;
            _amount = _proposal.votesAgainst[msg.sender].amount - _burnAmount;
            _proposal.amountFor += _amount;
            _proposal.votesFor[msg.sender].amount += _amount;
            _proposal.amountAgainst -= _amount;
            _proposal.votesAgainst[msg.sender].amount = 0;
        } else {
            _burnAmount = _proposal.votesFor[msg.sender].amount / 10;
            _amount = _proposal.votesFor[msg.sender].amount - _burnAmount;
            _proposal.amountAgainst += _amount;
            _proposal.votesAgainst[msg.sender].amount += _amount;
            _proposal.amountFor -= _amount;
            _proposal.votesFor[msg.sender].amount = 0;
        }
        require(_amount > 0, "no vote");
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
        payable(address(0)).transfer(_burnAmount);
    }

    /**
     * @dev Vote in support of or against proposal
     * @param _proposalId ID of proposal
     * @param _for whether or not to vote in support of proposal
     */
    function vote(uint256 _proposalId, bool _for) external payable {
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

    // *****************************************************************************
    // *                                                                           *
    // *                               Getters                                     *
    // *                                                                           *
    // *****************************************************************************

    /**
     * @dev Get proposal details
     * @param _proposalId ID of proposal
     * @return initiator address of user who initiated Schelling game
     * @return proposedImplementation address of proposed bridge implementation
     * @return amountFor amount of tokens in support of proposal
     * @return amountAgainst amount of tokens against proposal
     * @return submissionDeadline deadline for submitting implementation address
     * @return expirationTime deadline for voting, extended when outcome changes
     * @return executed whether or not proposal has been executed
     * @return guardianSigned whether or not guardian has signed proposal
     * @return outcome final outcome of proposal
     */
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
            _proposal.guardianSigned,
            _proposal.outcome
        );
    }

    /**
     * @dev Get vote details
     * @param _proposalId ID of proposal
     * @param _voter address of voter
     * @return amountFor amount of tokens in support of proposal
     * @return amountAgainst amount of tokens against proposal
     */
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
