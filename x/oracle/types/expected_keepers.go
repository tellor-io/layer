package types

import (
	context "context"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	rktypes "github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ICS4Wrapper defines the expected ICS4Wrapper for middleware
type ICS4Wrapper interface {
	SendPacket(
		ctx sdk.Context,
		chanCap *capabilitytypes.Capability,
		sourcePort string,
		sourceChannel string,
		timeoutHeight clienttypes.Height,
		timeoutTimestamp uint64,
		data []byte,
	) (sequence uint64, err error)
}

// ChannelKeeper defines the expected IBC channel keeper
type ChannelKeeper interface {
	GetChannel(ctx sdk.Context, portID, channelID string) (channeltypes.Channel, bool)
	GetNextSequenceSend(ctx sdk.Context, portID, channelID string) (uint64, bool)
}

// PortKeeper defines the expected IBC port keeper
type PortKeeper interface {
	BindPort(ctx sdk.Context, portID string) *capabilitytypes.Capability
	IsBound(ctx sdk.Context, portID string) bool
}

// ScopedKeeper defines the expected x/capability scoped keeper interface
type ScopedKeeper interface {
	GetCapability(ctx sdk.Context, name string) (*capabilitytypes.Capability, bool)
	AuthenticateCapability(ctx sdk.Context, capability *capabilitytypes.Capability, name string) bool
	ClaimCapability(ctx sdk.Context, capability *capabilitytypes.Capability, name string) error
}

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	GetModuleAccount(ctx context.Context, name string) sdk.ModuleAccountI
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	BurnCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	// Methods imported from bank should be defined here
}

// BridgeKeeper
type BridgeKeeper interface {
	GetDepositStatus(ctx context.Context, depositId uint64) (bool, error)
}

type RegistryKeeper interface {
	// Methods imported from registry should be defined here
	GetSpec(ctx context.Context, queryType string) (rktypes.DataSpec, error)
}

type ReporterKeeper interface {
	// Methods imported from reporter should be defined here
	ReporterStake(ctx context.Context, repAddress sdk.AccAddress, queryId []byte) (math.Int, error)
	AddTip(ctx context.Context, metaId uint64, tip Reward) error
	AddTbr(ctx context.Context, metaId uint64, tbr Reward) error
}

type RegistryHooks interface {
	AfterDataSpecUpdated(ctx context.Context, querytype string, dataspec rktypes.DataSpec) error
}
