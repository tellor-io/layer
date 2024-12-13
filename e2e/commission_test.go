package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
)

func TestMaxMins(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()
	layer := e2e.LayerSpinup(t)
	require.NotNil(layer)
	require.NotNil(ctx)

	// get a validator to call upon
	validators, valAccAddresses, valAddrs, err := e2e.GetValAddresses(ctx, layer)
	require.NoError(err)
	validatorI := validators[0]
	valIAccAddress := valAccAddresses[0]
	require.NotNil(valIAccAddress)
	valIValAddr := valAddrs[0]

	// submit minting proposal and vote yes on it from all validators
	require.NoError(e2e.TurnOnMinting(ctx, layer, validatorI))

	// validatorI sends a create reporter tx with commiss rate > 100...
	// layerd tx reporter create-reporter commission-rate min-tokens-required
	txHash, err := validatorI.ExecTx(ctx, "validator", "reporter", "create-reporter", "99.99999", math.NewUint(1*1e6).String(), "--keyring-dir", "/var/cosmos-chain/layer-1")
	require.NoError(err)
	fmt.Println("TX HASH (validatorI becomes a reporter): ", txHash)
	// check if reporter exists
	// layerd query reporter reporter reporteraddr
	reporters, _, err := validatorI.ExecQuery(ctx, "reporter", "reporters")
	require.NoError(err)
	var reportersResponse e2e.ReportersResponse
	require.NoError(json.Unmarshal(reporters, &reportersResponse))
	fmt.Println("REPORTERS RESPONSE commission rate: ", reportersResponse.Reporters[0].Metadata.CommissionRate)
	fmt.Println("REPORTERS RESPONSE min tokens: ", reportersResponse.Reporters[0].Metadata.MinTokens)
	// require.Equal(reportersResponse.Reporters[0].Metadata.CommissionRate, "500")
	// require.Equal(reportersResponse.Reporters[0].Metadata.MinTokens, "1000001")
	require.NoError(testutil.WaitForBlocks(ctx, 2, validatorI))

	// user delegates to validatorI validator and selects reporter
	// Create and fund a user who will be a delegator
	userName := "user1"
	userStartingBalance := math.NewInt(999_999 * 1e6)
	user := interchaintest.GetAndFundTestUsers(t, ctx, userName, userStartingBalance, layer)[0]
	require.NotNil(user)

	// user delegate to validatorI validator
	txHash, err = validatorI.ExecTx(ctx, user.FormattedAddress(), "staking", "delegate", valIValAddr, "1000005loya", "--keyring-dir", "/var/cosmos-chain/layer-1")
	require.NoError(err)
	fmt.Println("TX HASH (user delegates to validatorI): ", txHash)
	// check on delegation
	delegations, _, err := validatorI.ExecQuery(ctx, "staking", "delegations", valIAccAddress)
	require.NoError(err)
	var delegationsResponse e2e.QueryDelegatorDelegationsResponse
	require.NoError(json.Unmarshal(delegations, &delegationsResponse))
	// require.Equal(delegationsResponse.DelegationResponses[0].Delegation.Shares.String(), "1000000.000000000000000000")
	fmt.Println("DELEGATIONS RESPONSE: ", delegationsResponse)

	// user selects reporter
	// layerd tx delegation select-reporter reporteraddr
	txHash, err = validatorI.ExecTx(ctx, user.FormattedAddress(), "reporter", "select-reporter", valIAccAddress, "--keyring-dir", "/var/cosmos-chain/layer-1")
	require.NoError(err)
	fmt.Println("TX HASH (user selects reporter): ", txHash)
	// check on selection
	// layerd query delegation delegationaddr
	selection, _, err := validatorI.ExecQuery(ctx, "reporter", "selector-reporter", user.FormattedAddress())
	require.NoError(err)
	var selectionResponse e2e.QuerySelectorReporterResponse
	require.NoError(json.Unmarshal(selection, &selectionResponse))
	require.Equal(selectionResponse.Reporter, valIAccAddress)
	fmt.Println("SELECTION RESPONSE: ", selectionResponse)
}
