package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/dispute module sentinel errors
var (
	INSUFFICIENT_BALANCE          = sdkerrors.Register(ModuleName, 2, "insufficient balance")
	ErrDisputeDoesNotExist        = sdkerrors.Register(ModuleName, 3, "dispute not found")
	ErrDisputeTimeExpired         = sdkerrors.Register(ModuleName, 4, "dispute time expired")
	ErrDisputeFeeAlreadyMet       = sdkerrors.Register(ModuleName, 5, "dispute fee already met")
	ErrDisputeNotInVotingState    = sdkerrors.Register(ModuleName, 6, "dispute not in voting state")
	ErrVoterHasAlreadyVoted       = sdkerrors.Register(ModuleName, 7, "voter has already voted")
	ErrVoteDoesNotExist           = sdkerrors.Register(ModuleName, 8, "vote does not exist")
	ErrVotingPeriodEnded          = sdkerrors.Register(ModuleName, 9, "voting period ended")
	ErrMinimumTRBrequired         = sdkerrors.Register(ModuleName, 10, "Minimum fee amount is 1 TRB")
	ErrInvalidFeeDenom            = sdkerrors.Register(ModuleName, 11, "invalid fee denom")
	ErrInvalidDisputeCategory     = sdkerrors.Register(ModuleName, 12, "invalid dispute category")
	ErrInvalidSigner              = sdkerrors.Register(ModuleName, 13, "expected teamaccount as only signer for updateTeam message")
	ErrNoQuorumStillVoting        = sdkerrors.Register(ModuleName, 14, "vote period not ended and quorum not reached")
	ErrSelfDisputeFromBond        = sdkerrors.Register(ModuleName, 15, "proposer cannot pay from their bond when creating a dispute on themselves")
	ErrDisputedReportDoesNotExist = sdkerrors.Register(ModuleName, 16, "the report passed into the dispute does not exist")
)
