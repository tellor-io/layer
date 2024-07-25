package testutils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TestAccount struct {
	Name    string
	Address sdk.AccAddress
}

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

func CreateKeyringAccounts(t *testing.T, kr keyring.Keyring, num int) []TestAccount {
	t.Helper()
	require := require.New(t)

	accounts := make([]TestAccount, num)
	for i := range accounts {
		record, _, err := kr.NewMnemonic(
			fmt.Sprintf("key-%d", i),
			keyring.English,
			sdk.FullFundraiserPath,
			keyring.DefaultBIP39Passphrase,
			hd.Secp256k1)
		require.NoError(err)

		addr, err := record.GetAddress()
		require.NoError(err)
		kr.Delete(fmt.Sprintf("key-%d", i))

		accounts[i] = TestAccount{Name: record.Name, Address: addr}
	}

	return accounts
}

func ClearKeyring(t *testing.T, kr keyring.Keyring) {
	t.Helper()

	records, err := kr.List()
	require.NoError(t, err)

	for _, record := range records {
		kr.Delete(record.Name)
	}
}
