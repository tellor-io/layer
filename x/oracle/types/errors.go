package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/oracle module sentinel errors
var (
	ErrSample = sdkerrors.Register(ModuleName, 1100, "sample error")

	ErrValidatorNotBonded          = sdkerrors.Register(ModuleName, 1101, "validator is not staked")
	ErrSignatureVerificationFailed = sdkerrors.Register(ModuleName, 1102, "signature verification failed")
	ErrMissedCommitRevealWindow    = sdkerrors.Register(ModuleName, 1103, "missed commit reveal window")
)
