package testutils

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

func CreateTestContext(t *testing.T) sdk.Context {
	t.Helper()

	key := storetypes.NewKVStoreKey(oracletypes.StoreKey)

	testCtx := testutil.DefaultContextWithDB(
		t,
		key,
		storetypes.NewTransientStoreKey("test"),
	)

	return testCtx.Ctx
}
