package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/reporter module sentinel errors
var (
	ErrInvalidSigner                = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrSample                       = sdkerrors.Register(ModuleName, 1101, "sample error")
	ErrReporterExists               = sdkerrors.Register(ModuleName, 1102, "reporter already registered!")
	ErrAddressDelegated             = sdkerrors.Register(ModuleName, 1103, "address currently delegated to a reporter")
	ErrCommissionLTMinRate          = sdkerrors.Register(ModuleName, 1104, "commission cannot be less than min rate")
	ErrInvalidReporter              = sdkerrors.Register(ModuleName, 1105, "invalid reporter")
	ErrInsufficientTokens           = sdkerrors.Register(ModuleName, 1106, "insufficient tokens")
	ErrTokenAmountMismatch          = sdkerrors.Register(ModuleName, 1107, "token amount mismatch")
	ErrReporterMismatch             = sdkerrors.Register(ModuleName, 1108, "reporter mismatch")
	ErrEmptyDelegationDistInfo      = sdkerrors.Register(ModuleName, 1109, "no delegation distribution info")
	ErrNoReporterCommission         = sdkerrors.Register(ModuleName, 1110, "no reporter commission to withdraw")
	ErrReporterDoesNotExist         = sdkerrors.Register(ModuleName, 1111, "reporter does not exist")
	ErrReporterJailed               = sdkerrors.Register(ModuleName, 1112, "reporter jailed")
	ErrReporterNotJailed            = sdkerrors.Register(ModuleName, 1113, "reporter not jailed")
	ErrNoUnbondingDelegationEntries = sdkerrors.Register(ModuleName, 1114, "no unbonding delegation entries")
	ErrExceedsMaxDelegations        = sdkerrors.Register(ModuleName, 1115, "exceeds max number of delegations")
)
