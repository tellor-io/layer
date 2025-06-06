package e2e_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// cd e2e
// go test -run TestAttestationSlashing -v --timeout 5m
func TestAttestationSlashing(t *testing.T) {
	require := require.New(t)

	t.Helper()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	cosmos.SetSDKConfig("tellor")

	// Create modified genesis for test
	modifyGenesis := []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.dispute.params.team_address", sdk.MustAccAddressFromBech32("tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf").Bytes()),
		cosmos.NewGenesisKV("consensus.params.abci.vote_extensions_enable_height", "1"),
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "15s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "loya"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices.0.amount", "0.0"),
	}

	// Set up validators
	nv := 2
	nf := 1
	chains := interchaintest.CreateChainsWithChainSpecs(t, []*interchaintest.ChainSpec{
		{
			NumValidators: &nv,
			NumFullNodes:  &nf,
			ChainConfig: ibc.ChainConfig{
				Type:           "cosmos",
				Name:           "layer",
				ChainID:        "layer",
				Bin:            "layerd",
				Denom:          "loya",
				Bech32Prefix:   "tellor",
				CoinType:       "118",
				GasPrices:      "0.0loya",
				GasAdjustment:  1.1,
				TrustingPeriod: "504h",
				NoHostMount:    false,
				Images: []ibc.DockerImage{
					{
						Repository: "layer",
						Version:    "local",
						UidGid:     "1025:1025",
					},
				},
				EncodingConfig:      e2e.LayerEncoding(),
				ModifyGenesis:       cosmos.ModifyGenesis(modifyGenesis),
				AdditionalStartArgs: []string{"--key-name", "validator"},
			},
		},
	})

	client, network := interchaintest.DockerSetup(t)

	chain := chains[0].(*cosmos.CosmosChain)

	ic := interchaintest.NewInterchain().
		AddChain(chain)

	ctx := context.Background()

	require.NoError(ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: false,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})
	require.NoError(chain.RecoverKey(ctx, "team", teamMnemonic))
	require.NoError(chain.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: "tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf",
		Amount:  math.NewInt(1000000000000),
		Denom:   "loya",
	}))

	// Setup validator info
	type Validators struct {
		AccAddr string
		ValAddr string
		Node    *cosmos.ChainNode
		EVMPriv *ecdsa.PrivateKey
		EVMAddr string
	}

	validators := make([]Validators, len(chain.Validators))
	for i := range chain.Validators {
		val := chain.Validators[i]
		valAddr, err := val.AccountKeyBech32(ctx, "validator")
		require.NoError(err)
		valvalAddr, err := val.KeyBech32(ctx, "validator", "val")
		require.NoError(err)
		fmt.Println("val", i, " Account Address: ", valAddr)
		fmt.Println("val", i, " Validator Address: ", valvalAddr)

		validators[i] = Validators{
			AccAddr: valAddr,
			ValAddr: valvalAddr,
			Node:    val,
		}
	}

	// get private keys and evm addresses from the validators
	for i, v := range validators {
		exportCmd := []string{
			"sh", "-c", "echo y | layerd keys export validator --unarmored-hex --unsafe --keyring-backend test --home " +
				v.Node.HomeDir(),
		}

		stdout, _, err := v.Node.Exec(ctx, exportCmd, v.Node.Chain.Config().Env)
		require.NoError(err)

		// Parse the output to extract the private key
		// The output will have a warning and then the key, we need to extract just the key
		lines := strings.Split(string(stdout), "\n")
		var privKeyHex string
		for _, line := range lines {
			// Skip warning lines, look for hex string
			if len(line) >= 64 && !strings.Contains(line, "WARNING") {
				privKeyHex = line
				break
			}
		}

		require.NotEmpty(privKeyHex, "Failed to extract private key")

		// Create ECDSA private key from the exported key
		privKeyBytes, err := hex.DecodeString(privKeyHex)
		require.NoError(err)

		// Create the private key from bytes
		exportedPrivKey, err := crypto.ToECDSA(privKeyBytes)
		require.NoError(err)

		// Generate Ethereum address from this key for verification
		exportedEVMAddr := crypto.PubkeyToAddress(exportedPrivKey.PublicKey).Hex()

		// Update validator info with the actual keys from the keyring
		validators[i].EVMPriv = exportedPrivKey
		validators[i].EVMAddr = exportedEVMAddr

		fmt.Printf("Validator %d - Exported EVM address: %s\n", i, exportedEVMAddr)

	}

	// Confirm that 2 validators are bonded
	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 2)

	// Record initial validator tokens
	initialTokens := make([]math.Int, len(validators))
	for i, v := range validators {
		valInfo, err := chain.StakingQueryValidator(ctx, v.ValAddr)
		require.NoError(err)
		initialTokens[i] = valInfo.Tokens
		fmt.Println("Validator", i, "initial tokens:", initialTokens[i])
	}

	// Submit minting proposal and vote yes on it from all validators
	require.NoError(e2e.TurnOnMinting(ctx, chain, validators[0].Node))
	require.NoError(testutil.WaitForBlocks(ctx, 5, validators[0].Node))
	result, err := chain.GovQueryProposal(ctx, 1)
	require.NoError(err)
	fmt.Println("Proposal status: ", result.Status.String())
	require.Equal(result.Status.String(), "PROPOSAL_STATUS_PASSED")

	// Validators become reporters
	for i, v := range validators {
		moniker := fmt.Sprintf("reporter_moniker%d", i)
		txHash, err := v.Node.ExecTx(ctx, "validator", "reporter", "create-reporter", "0.5", "100000000", moniker, "--keyring-dir", v.Node.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (val", i+1, "becomes a reporter): ", txHash)

		// Export validator private key (similar to your CLI example)
		// not working, but good sign version. printed the "help" document for the command:
		// exportCmd := []string{
		// 	"sh", "-c", "yes y | layerd keys export validator --unarmored-hex --unsafe --keyring-backend test --home " +
		// 		v.Val.HomeDir(),
		// }

		exportCmd := []string{
			"sh", "-c", "echo y | layerd keys export validator --unarmored-hex --unsafe --keyring-backend test --home " +
				v.Node.HomeDir(),
		}

		// exportCmd := []string{
		// 	"echo", "'y'", "|", "layerd", "keys", "export", "validator", "--unarmored-hex", "--unsafe", "--keyring-backend", "test", "--home",
		// 	v.Val.HomeDir(),
		// }

		// exportCmd := []string{
		// 	"bash", "-c",
		// 	fmt.Sprintf("echo y | layerd keys export validator "+
		// 		"--unarmored-hex --unsafe --keyring-backend test --home %s", v.Val.HomeDir()),
		// }

		stdout, _, err := v.Node.Exec(ctx, exportCmd, v.Node.Chain.Config().Env)
		require.NoError(err)

		// Parse the output to extract the private key
		// The output will have a warning and then the key, we need to extract just the key
		lines := strings.Split(string(stdout), "\n")
		var privKeyHex string
		for _, line := range lines {
			// Skip warning lines, look for hex string
			if len(line) >= 64 && !strings.Contains(line, "WARNING") {
				privKeyHex = line
				break
			}
		}

		require.NotEmpty(privKeyHex, "Failed to extract private key")

		// Create ECDSA private key from the exported key
		privKeyBytes, err := hex.DecodeString(privKeyHex)
		require.NoError(err)

		// Create the private key from bytes
		exportedPrivKey, err := crypto.ToECDSA(privKeyBytes)
		require.NoError(err)

		// Generate Ethereum address from this key for verification
		exportedEVMAddr := crypto.PubkeyToAddress(exportedPrivKey.PublicKey).Hex()

		// Update validator info with the actual keys from the keyring
		validators[i].EVMPriv = exportedPrivKey
		validators[i].EVMAddr = exportedEVMAddr

		fmt.Printf("Validator %d - Exported EVM address: %s\n", i, exportedEVMAddr)
	}

	// Query reporters to confirm creation
	res, _, err := validators[0].Node.ExecQuery(ctx, "reporter", "reporters")
	require.NoError(err)
	var reportersRes e2e.QueryReportersResponse
	err = json.Unmarshal(res, &reportersRes)
	require.NoError(err)
	require.Equal(len(reportersRes.Reporters), 2)

	// Query, decode, and print evm validators
	// command: `layerd query bridge get-evm-validators [flags]`
	// message QueryGetEvmValidatorsResponse {
	// 	repeated QueryBridgeValidator bridge_validator_set = 1;
	// }

	// message QueryBridgeValidator {
	// 	string ethereumAddress = 1;
	// 	uint64 power = 2;
	// }
	type QueryBridgeValidator struct {
		EthereumAddress string `json:"ethereumAddress"`
		Power           string `json:"power"`
	}
	type QueryGetEvmValidatorsResponse struct {
		BridgeValidatorSet []QueryBridgeValidator `json:"bridge_validator_set"`
	}
	// query
	evmValidatorsRes, _, err := validators[0].Node.ExecQuery(ctx, "bridge", "get-evm-validators")
	require.NoError(err)
	var evmValidators QueryGetEvmValidatorsResponse
	err = json.Unmarshal(evmValidatorsRes, &evmValidators)
	require.NoError(err)
	fmt.Println("evmValidatorsRes", evmValidatorsRes)
	for i, val := range evmValidators.BridgeValidatorSet {
		fmt.Println("EVM Validator", i, "Address:", val.EthereumAddress, "Power:", val.Power)

		// Find the matching validator in our list
		var found bool
		for j, v := range validators {
			if strings.EqualFold(val.EthereumAddress, v.EVMAddr) {
				fmt.Printf("Matched with validator %d (exported key)\n", j)
				found = true
				break
			}
		}

		// If not found, the validator's EVM address doesn't match any of our exported keys
		require.True(found, fmt.Sprintf("EVM validator address %s doesn't match any exported key", val.EthereumAddress))
	}

	// Validator reporters report for the cycle list to create oracle data
	currentCycleListRes, _, err := validators[0].Node.ExecQuery(ctx, "oracle", "current-cyclelist-query")
	require.NoError(err)
	var currentCycleList e2e.QueryCurrentCyclelistQueryResponse
	err = json.Unmarshal(currentCycleListRes, &currentCycleList)
	require.NoError(err)

	for i, v := range validators {
		// Report for the cycle list
		txHash, _, err := v.Node.Exec(ctx, v.Node.TxCommand("validator", "oracle", "submit-value", currentCycleList.QueryData, value, "--fees", "25loya", "--keyring-dir", v.Node.HomeDir()), v.Node.Chain.Config().Env)
		require.NoError(err)
		height, err := chain.Height(ctx)
		require.NoError(err)
		fmt.Println("validator [", i, "] reported at height ", height, "tx:", txHash)
	}

	// Wait for query to expire and be included in consensus
	err = testutil.WaitForBlocks(ctx, 3, validators[0].Node)
	require.NoError(err)

	// Get query ID and timestamp from report
	var queryId string
	var timestamp uint64

	reports, _, err := validators[0].Node.ExecQuery(ctx, "oracle", "get-reportsby-reporter", validators[0].AccAddr)
	require.NoError(err)
	var reportsRes e2e.QueryMicroReportsResponse
	err = json.Unmarshal(reports, &reportsRes)
	require.NoError(err)
	require.GreaterOrEqual(len(reportsRes.MicroReports), 1)

	// Query and print validator checkpoint `layerd query bridge get-validator-checkpoint [flags]`
	checkpointRes2, _, err := validators[0].Node.ExecQuery(ctx, "bridge", "get-validator-checkpoint")
	require.NoError(err)
	fmt.Println("Validator checkpoint:", checkpointRes2)

	queryId = reportsRes.MicroReports[0].QueryID
	tsInt, err := strconv.ParseUint(reportsRes.MicroReports[0].Timestamp, 10, 64)
	require.NoError(err)
	timestamp = tsInt
	fmt.Println("QueryID:", queryId, "Timestamp:", timestamp)

	require.NoError(err)

	// Get attestation data by snapshot
	snapshotsRes, _, err := validators[0].Node.ExecQuery(ctx, "bridge", "get-snapshots-by-report", queryId, fmt.Sprintf("%d", timestamp))
	require.NoError(err)
	var snapshotsResponse e2e.QueryGetSnapshotsByReportResponse
	err = json.Unmarshal(snapshotsRes, &snapshotsResponse)
	require.NoError(err)
	require.GreaterOrEqual(len(snapshotsResponse.Snapshots), 1)

	lastSnapshot := snapshotsResponse.Snapshots[len(snapshotsResponse.Snapshots)-1]
	fmt.Println("Last snapshot:", lastSnapshot)

	attestationDataRes, _, err := validators[0].Node.ExecQuery(ctx, "bridge", "get-attestation-databy-snapshot", lastSnapshot)
	require.NoError(err)
	var attestationData e2e.QueryGetAttestationDataBySnapshotResponse
	err = json.Unmarshal(attestationDataRes, &attestationData)
	require.NoError(err)
	fmt.Println("Attestation data:", attestationData)

	// Get the current checkpoint for use in malicious attestation
	checkpointRes, _, err := validators[0].Node.ExecQuery(ctx, "bridge", "get-checkpoint")
	require.NoError(err)
	var checkpointData struct {
		Checkpoint string `json:"checkpoint"`
	}
	err = json.Unmarshal(checkpointRes, &checkpointData)
	require.NoError(err)
	checkpointStr := checkpointData.Checkpoint
	fmt.Println("Current checkpoint:", checkpointStr)

	// Create a malicious attestation from validator[1] for slashing purposes
	// Modify the original attestation data but use same signatures
	maliciousValue := "9999" // Different from original 5000

	// Create attestation data bytes for signing
	queryIdBytes, err := hex.DecodeString(queryId)
	require.NoError(err)
	checkpointBytes, err := hex.DecodeString(checkpointStr)
	require.NoError(err)

	attestTimestamp := uint64(time.Now().UnixMilli())

	// Generate signature from validator[1]
	targetValidator := validators[1]

	// Create snapshot data for malicious attestation
	snapshotBytes, err := encodeOracleAttestationData(
		queryIdBytes,
		maliciousValue,
		timestamp,
		uint64(100),    // aggregate power
		timestamp-1000, // previous timestamp
		timestamp+1000, // next timestamp
		checkpointBytes,
		attestTimestamp,
		timestamp, // last consensus timestamp
	)
	require.NoError(err)

	// Sign the snapshot
	msgHash := sha256.Sum256(snapshotBytes)
	signature, err := crypto.Sign(msgHash[:], targetValidator.EVMPriv)
	require.NoError(err)
	sigHex := hex.EncodeToString(signature)

	// Submit attestation evidence
	_, err = validators[0].Node.ExecTx(
		ctx,
		"validator",
		"bridge",
		"submit-attestation-evidence",
		queryId,
		maliciousValue,
		fmt.Sprintf("%d", timestamp),
		"100",                             // aggregate power
		fmt.Sprintf("%d", timestamp-1000), // previous timestamp
		fmt.Sprintf("%d", timestamp+1000), // next timestamp
		checkpointStr,
		fmt.Sprintf("%d", attestTimestamp),
		fmt.Sprintf("%d", timestamp), // last consensus timestamp
		sigHex,
		"--keyring-dir", validators[0].Node.HomeDir(),
	)
	require.NoError(err)
	fmt.Println("Submitted attestation evidence against validator", targetValidator.ValAddr)

	// Wait for evidence to be processed
	err = testutil.WaitForBlocks(ctx, 3, validators[0].Node)
	require.NoError(err)

	// Check if the validator was slashed
	slashedValInfo, err := chain.StakingQueryValidator(ctx, targetValidator.ValAddr)
	require.NoError(err)

	// Verify the validator was slashed
	fmt.Println("Validator initial tokens:", initialTokens[1])
	fmt.Println("Validator final tokens:", slashedValInfo.Tokens)
	require.True(slashedValInfo.Tokens.LT(initialTokens[1]), "Validator should have been slashed")

	// Check if the validator was jailed
	require.True(slashedValInfo.Jailed, "Validator should have been jailed")
}

// Helper function to encode oracle attestation data in the same way as the bridge keeper
func encodeOracleAttestationData(
	queryId []byte,
	value string,
	timestamp uint64,
	aggregatePower uint64,
	previousTimestamp uint64,
	nextTimestamp uint64,
	checkpoint []byte,
	attestationTimestamp uint64,
	lastConsensusTimestamp uint64,
) ([]byte, error) {
	// Format matches the one in keeper.EncodeOracleAttestationData
	encoded := []byte{}
	encoded = append(encoded, queryId...)
	encoded = append(encoded, []byte(value)...)
	encoded = append(encoded, []byte(fmt.Sprintf("%d", timestamp))...)
	encoded = append(encoded, []byte(fmt.Sprintf("%d", aggregatePower))...)
	encoded = append(encoded, []byte(fmt.Sprintf("%d", previousTimestamp))...)
	encoded = append(encoded, []byte(fmt.Sprintf("%d", nextTimestamp))...)
	encoded = append(encoded, checkpoint...)
	encoded = append(encoded, []byte(fmt.Sprintf("%d", attestationTimestamp))...)
	encoded = append(encoded, []byte(fmt.Sprintf("%d", lastConsensusTimestamp))...)
	return encoded, nil
}
