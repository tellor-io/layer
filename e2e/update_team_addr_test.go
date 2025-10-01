package e2e_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestUpdateTeamAddr(t *testing.T) {
	require := require.New(t)

	cosmos.SetSDKConfig("tellor")

	chain, _, ctx := e2e.SetupChain(t, 2, 0)

	validatorsInfo, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	e2e.PrintValidatorInfo(ctx, validatorsInfo)

	type Validators struct {
		Addr    string
		ValAddr string
		Val     *cosmos.ChainNode
	}

	validators := make([]Validators, len(validatorsInfo))
	for i, v := range validatorsInfo {
		fmt.Println("val", i, " Account Address: ", v.AccAddr)
		fmt.Println("val", i, " Validator Address: ", v.ValAddr)
		validators[i] = Validators{
			Addr:    v.AccAddr,
			ValAddr: v.ValAddr,
			Val:     v.Node,
		}
	}

	// queryValidators to confirm that 2 validators are bonded
	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 2)

	// submit minting proposal and vote yes on it from all validators
	require.NoError(e2e.TurnOnMinting(ctx, chain, validators[0].Val))
	require.NoError(testutil.WaitForBlocks(ctx, 5, validators[0].Val))
	result, err := chain.GovQueryProposal(ctx, 1)
	require.NoError(err)
	fmt.Println("Proposal status: ", result.Status.String())
	require.Equal(result.Status.String(), "PROPOSAL_STATUS_PASSED")

	// val0 calls update team addr, tries to make val1 team addr, should err bc signer is not team
	newTeamAddrBad := validators[1].ValAddr
	txHash, err := validators[0].Val.ExecTx(ctx, "validator", "dispute", "update-team", newTeamAddrBad, "--fees", "5loya", "--keyring-dir", validators[0].Val.HomeDir(), "--chain-id", chain.Config().ChainID)
	require.ErrorContains(err, "expected teamaccount as only signer for updateTeam message")
	fmt.Println("TX HASH(Update Team Addr called (bad signer)): ", txHash)

	// team calls update team addr, make val1 val addr the team addr, should error with bech32  expected tellor got tellorvaloper
	txHash, err = validators[0].Val.ExecTx(ctx, "team", "dispute", "update-team", newTeamAddrBad, "--fees", "5loya", "--keyring-dir", validators[0].Val.HomeDir(), "--chain-id", chain.Config().ChainID)
	require.ErrorContains(err, "invalid Bech32 prefix;")
	fmt.Println("TX HASH(Update Team Addr called (bad new addr)): ", txHash)

	// team calls update team addr, make val1 addr the team addr, all square
	newTeamAddrGood := validators[1].Addr
	txHash, err = validators[0].Val.ExecTx(ctx, "team", "dispute", "update-team", newTeamAddrGood, "--fees", "5loya", "--keyring-dir", validators[0].Val.HomeDir(), "--chain-id", chain.Config().ChainID)
	require.NoError(err)
	fmt.Println("TX HASH(Update Team Addr called (success)): ", txHash)

	// query team addr to confirm update
	teamAddrBz, _, err := e2e.QueryWithTimeout(ctx, validators[0].Val, "dispute", "team-address")
	require.NoError(err)
	var teamAddr e2e.QueryTeamAddressResponse
	require.NoError(json.Unmarshal(teamAddrBz, &teamAddr))
	require.Equal(teamAddr.TeamAddress, newTeamAddrGood)
	height, err := validators[0].Val.Height(ctx)
	require.NoError(err)
	fmt.Println("current height: ", height)

	// wait 4 blocks
	require.NoError(testutil.WaitForBlocks(ctx, 4, validators[0].Val))
	height, err = validators[0].Val.Height(ctx)
	require.NoError(err)
	fmt.Println("current height: ", height)
}
