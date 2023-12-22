package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/oracle module sentinel errors
var (
	ErrInvalidSigner = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")

	ErrValidatorNotBonded          = sdkerrors.Register(ModuleName, 1101, "validator is not staked")
	ErrSignatureVerificationFailed = sdkerrors.Register(ModuleName, 1102, "signature verification failed")
	ErrMissedCommitRevealWindow    = sdkerrors.Register(ModuleName, 1103, "missed commit reveal window")
)
