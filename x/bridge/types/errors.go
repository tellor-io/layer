package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/bridge module sentinel errors
var (
	ErrSample                    = sdkerrors.Register(ModuleName, 1100, "sample error")
	ErrNoAggregate               = sdkerrors.Register(ModuleName, 1101, "no aggregate found")
	ErrAggregateFlagged          = sdkerrors.Register(ModuleName, 1102, "aggregate flagged")
	ErrInsufficientReporterPower = sdkerrors.Register(ModuleName, 1103, "insufficient reporter power")
	ErrReportTooYoung            = sdkerrors.Register(ModuleName, 1104, "report too young")
	ErrInvalidDepositReportValue = sdkerrors.Register(ModuleName, 1105, "invalid deposit report value")
	ErrDepositAlreadyClaimed     = sdkerrors.Register(ModuleName, 1106, "deposit already claimed")
)
