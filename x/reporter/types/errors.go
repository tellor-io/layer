package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/reporter module sentinel errors
var (
	ErrInvalidSigner       = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrSample              = sdkerrors.Register(ModuleName, 1101, "sample error")
	ErrReporterExists      = sdkerrors.Register(ModuleName, 1102, "reporter already registered!")
	ErrAddressDelegated    = sdkerrors.Register(ModuleName, 1103, "address currently delegated to a reporter")
	ErrCommissionLTMinRate = sdkerrors.Register(ModuleName, 1104, "commission cannot be less than min rate")
	ErrInvalidReporter     = sdkerrors.Register(ModuleName, 1105, "invalid reporter")
	ErrInsufficientTokens  = sdkerrors.Register(ModuleName, 1106, "insufficient tokens")
	ErrTokenAmountMismatch = sdkerrors.Register(ModuleName, 1107, "token amount mismatch")
)
