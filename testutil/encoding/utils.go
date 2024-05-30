package encoding

import (
	"testing"

	"github.com/cosmos/ibc-go/modules/capability"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibc "github.com/cosmos/ibc-go/v8/modules/core"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/x/upgrade"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/staking"
)

// GetTestEncodingCfg returns an encoding config for testing purposes.
func GetTestEncodingCfg() testutil.TestEncodingConfig {
	encodingCfg := testutil.MakeTestEncodingConfig(
		auth.AppModuleBasic{},
		genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		bank.AppModuleBasic{},
		capability.AppModuleBasic{},
		staking.AppModuleBasic{},
		distr.AppModuleBasic{},
		gov.NewAppModuleBasic([]govclient.ProposalHandler{}),
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		ibc.AppModuleBasic{},
		ibctm.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		transfer.AppModuleBasic{},
		consensus.AppModuleBasic{},
	)

	return encodingCfg
}

// EncodeMessageToAny converts a message to an Any object for protobuf encoding.
func EncodeMessageToAny(t *testing.T, msg sdk.Msg) *codectypes.Any {
	t.Helper()
	any, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)
	return any
}
