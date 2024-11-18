package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	layerutil "github.com/tellor-io/layer/testutil"
	layertypes "github.com/tellor-io/layer/types"
)

func TestEscalatedDispute(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()
	layer := e2e.LayerSpinup(t)

	// get a validator
	validatorI, valAccAddress, valAddress, err := e2e.GetValIAddress(ctx, layer)
	require.NoError(err)

	// make a user account with 100 trb
	user1 := interchaintest.GetAndFundTestUsers(t, ctx, "user1", math.NewInt(900_000*1e6), layer)[0]
	user1Addr := user1.FormattedAddress()
	fmt.Println("User1 address: ", user1Addr)

	// subscribe to events ?

	// submit minting proposal and vote yes on it from all validators
	require.NoError(e2e.TurnOnMinting(ctx, layer, validatorI))

	// custom gov params set voting period to 15s, block time is 5s
	// waits 4 blocks (3 sometimes works)
	require.NoError(testutil.WaitForBlocks(ctx, 4, validatorI))
	result, err := layer.GovQueryProposal(ctx, 1)
	require.NoError(err)

	fmt.Println("Proposal status: ", result.Status.String())
	require.Equal(result.FinalTallyResult.Yes.String(), "10000000000000")
	require.Equal(result.FinalTallyResult.No.String(), "0")
	require.Equal(result.FinalTallyResult.Abstain.String(), "0")
	require.Equal(result.FinalTallyResult.NoWithVeto.String(), "0")
	require.Equal(result.Status.String(), "PROPOSAL_STATUS_PASSED")

	// validatorI becomes a reporter
	txHash, err := validatorI.ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "--keyring-dir", "/var/cosmos-chain/layer-1")
	require.NoError(err)
	fmt.Println("TX HASH (validatorI becomes a reporter): ", txHash)

	// user tips random amount (<1 trb + 1 loya) for LTC/USD
	randomTipInt := rand.Int63n(1000000) + 1
	randomTip := sdk.NewCoin("loya", math.NewInt(randomTipInt))
	fmt.Println("ltc/usd tip: ", randomTip.String())

	stdout, _, err := validatorI.Exec(ctx, validatorI.TxCommand("user1", "oracle", "tip", user1Addr, ltcusdQData, randomTip.String(), "--keyring-dir", "/var/cosmos-chain/layer-1"), validatorI.Chain.Config().Env)
	require.NoError(err)
	txHash, err = e2e.GetTxHashFromExec(stdout)
	fmt.Println("TX HASH (user tips ltc/usd): ", txHash)
	require.NoError(testutil.WaitForBlocks(ctx, 1, validatorI))

	// validator/reporter submits good value for LTC/USD
	ltcusdValue := layerutil.EncodeValue(75.98)
	valI, err := layer.StakingQueryValidator(ctx, valAddress)
	require.NoError(err)
	expectedPower := valI.Tokens // loya

	txHash, err = validatorI.ExecTx(ctx, "validator", "oracle", "submit-value", valAccAddress, ltcusdQData, ltcusdValue, "--keyring-dir", "/var/cosmos-chain/layer-1")
	require.NoError(err)
	fmt.Println("TX HASH (user reports LTC/USD): ", txHash)
	// require.NoError(testutil.WaitForBlocks(ctx, 1, validatorI))

	// make sure all is square on aggregate report
	ltcusdReport, _, err := validatorI.ExecQuery(ctx, "oracle", "get-reportsby-reporter", valAccAddress)
	require.NoError(err)
	var microReports e2e.ReportsResponse
	require.NoError(json.Unmarshal(ltcusdReport, &microReports))

	require.Equal(microReports.MicroReports[0].Reporter, valAccAddress)
	require.Equal(microReports.MicroReports[0].Value, ltcusdValue)
	require.Equal(microReports.MicroReports[0].AggregateMethod, "weighted-median")
	require.Equal(microReports.MicroReports[0].Power, expectedPower.QuoRaw(layertypes.PowerReduction.Int64()).String()) // power is in trb
	require.Equal(microReports.MicroReports[0].QueryType, "SpotPrice")
	// require.Equal(microReports.MicroReports[0].QueryID, ltcusdQId) // GVhdkSr7cjeOOYanpT8erh+655LNF+HQ3wY2gTJoI64= expected ?
	txResp, err := validatorI.TxHashToResponse(ctx, txHash)
	fmt.Println("txResp.Events: ", txResp.Events[len(txResp.Events)-1])
	fmt.Println("txResp.Logs: ", txResp.Logs)
	fmt.Println("txResp.Tx: ", txResp.Tx)
	fmt.Println("txResp.Height: ", txResp.Height)
	// txQuery, _, err := validatorI.ExecQuery(ctx, "tx", txHash)
	// require.NotNil(txQuery)
	// require.NoError(err)
	// fmt.Println("tx query: ", string(txQuery))
	// require.Equal(microReports.MicroReports[0].BlockNumber, blockNum)
	// require.Equal(microReports.MicroReports[0].Timestamp, timestamp)

	// user opens warning dispute on report
	bz, err := json.Marshal(microReports.MicroReports[0])
	require.NoError(err)

	txHash, err = validatorI.ExecTx(ctx, user1Addr, "dispute", "propose-dispute", string(bz), "warning", "500000000000loya", "false", "--keyring-dir", "/var/cosmos-chain/layer-1", "--gas", "1000000", "--fees", "1000000loya")
	require.NoError(err)
	fmt.Println("TX HASH (user opens warning dispute on report): ", txHash)

}
