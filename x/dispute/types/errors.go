package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/dispute module sentinel errors
var (
	INSUFFICIENT_BALANCE       = sdkerrors.Register(ModuleName, 2, "insufficient balance")
	ErrDisputeDoesNotExist     = sdkerrors.Register(ModuleName, 3, "dispute not found")
	ErrDisputeTimeExpired      = sdkerrors.Register(ModuleName, 4, "dispute time expired")
	ErrDisputeFeeAlreadyMet    = sdkerrors.Register(ModuleName, 5, "dispute fee already met")
	ErrDisputeNotInVotingState = sdkerrors.Register(ModuleName, 6, "dispute not in voting state")
	ErrVoterHasAlreadyVoted    = sdkerrors.Register(ModuleName, 7, "voter has already voted")
	ErrVoteDoesNotExist        = sdkerrors.Register(ModuleName, 8, "vote does not exist")
	ErrVotingPeriodEnded       = sdkerrors.Register(ModuleName, 9, "voting period ended")
	ErrZeroFeeAmount           = sdkerrors.Register(ModuleName, 10, "zero fee amount")
	ErrInvalidFeeDenom         = sdkerrors.Register(ModuleName, 11, "invalid fee denom")
)
