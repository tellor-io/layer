package v5

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func MigrateStore(ctx context.Context, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	// Create schema builder to use Collections API
	sb := collections.NewSchemaBuilder(storeService)

	// Create Collections API objects for params and domain separator
	paramsCollection := collections.NewItem(sb, bridgetypes.ParamsKey, "params", codec.CollValue[bridgetypes.Params](cdc))
	domainSeparatorCollection := collections.NewItem(sb, bridgetypes.ValsetCheckpointDomainSeparatorKey, "valset_checkpoint_domain_separator", collections.BytesValue)

	// Handle params migration
	params, err := paramsCollection.Get(ctx)
	if err != nil {
		// If params don't exist, create default params
		params = bridgetypes.DefaultParams()
	}
	params.MainnetChainId = "tellor-1"
	err = paramsCollection.Set(ctx, params)
	if err != nil {
		return err
	}

	// Set the domain separator using the same logic as SetValsetCheckpointDomainSeparator
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	mainnetChainId := "tellor-1" // We just set this in the params above

	var domainSeparator []byte
	if sdkCtx.ChainID() == mainnetChainId {
		// For mainnet, use the fixed domain separator: "checkpoint" padded to 32 bytes with zeros
		// This matches the Solidity constant: 0x636865636b706f696e7400000000000000000000000000000000000000000000
		domainSeparator = make([]byte, 32)
		copy(domainSeparator, []byte("checkpoint"))
	} else {
		// Create domain separator by ABI encoding "checkpoint" and chain ID
		// This matches the Solidity implementation: keccak256(abi.encode("checkpoint", chainId))
		StringType, err := abi.NewType("string", "", nil)
		if err != nil {
			return err
		}

		// ABI encode "checkpoint" and chain ID (both as strings)
		domainSeparatorArgs := abi.Arguments{
			{Type: StringType},
			{Type: StringType},
		}
		domainSeparatorEncoded, err := domainSeparatorArgs.Pack("checkpoint", sdkCtx.ChainID())
		if err != nil {
			return err
		}
		domainSeparator = crypto.Keccak256(domainSeparatorEncoded)
	}

	// Store the domain separator using Collections API
	err = domainSeparatorCollection.Set(ctx, domainSeparator)
	if err != nil {
		return err
	}

	return nil
}
