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
	ErrNotEnoughStake              = sdkerrors.Register(ModuleName, 1104, "not enough stake")
	ErrCommitRevealWindowEarly     = sdkerrors.Register(ModuleName, 1105, "commit reveal window is too early")
	ErrReporterJailed              = sdkerrors.Register(ModuleName, 1106, "reporter is jailed")
	ErrNoAvailableReports          = sdkerrors.Register(ModuleName, 1107, "no available reports")
	ErrNoReportsToAggregate        = sdkerrors.Register(ModuleName, 1108, "no reports to aggregate")
	ErrQueryNotFound               = sdkerrors.Register(ModuleName, 1109, "query not found")
	ErrNoTipsNotInCycle            = sdkerrors.Register(ModuleName, 1110, "no tips not in cycle")
	ErrNotTokenDeposit             = sdkerrors.Register(ModuleName, 1111, "not a token deposit")
	ErrInvalidQueryData            = sdkerrors.Register(ModuleName, 1112, "invalid query data")
)
