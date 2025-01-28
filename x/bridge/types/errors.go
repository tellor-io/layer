package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/bridge module sentinel errors
var (
	ErrSample                            = sdkerrors.Register(ModuleName, 1100, "sample error")
	ErrAggregateFlagged                  = sdkerrors.Register(ModuleName, 1101, "aggregate flagged")
	ErrInsufficientReporterPower         = sdkerrors.Register(ModuleName, 1102, "insufficient reporter power")
	ErrReportTooYoung                    = sdkerrors.Register(ModuleName, 1103, "report too young")
	ErrInvalidDepositReportValue         = sdkerrors.Register(ModuleName, 1104, "invalid deposit report value")
	ErrDepositAlreadyClaimed             = sdkerrors.Register(ModuleName, 1105, "deposit already claimed")
	ErrInvalidDepositIdsAndIndicesLength = sdkerrors.Register(ModuleName, 1106, "invalid deposit ids and indices length")
	ErrInvalidSigner                     = sdkerrors.Register(ModuleName, 1107, "invalid signer")
)
