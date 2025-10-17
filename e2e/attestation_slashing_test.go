package e2e_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
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

	// Wait for vote extensions to register EVM addresses, then query them directly
	waitErr := testutil.WaitForBlocks(ctx, 5, validators[0].Node)
	require.NoError(waitErr)

	// Define types for bridge validator queries
	type QueryBridgeValidator struct {
		EthereumAddress string `json:"ethereumAddress"`
		Power           string `json:"power"`
	}
	type QueryGetEvmValidatorsResponse struct {
		BridgeValidatorSet []QueryBridgeValidator `json:"bridge_validator_set"`
	}

	// Get private keys and query the actual registered EVM addresses
	for i, v := range validators {
		exportCmd := []string{
			"sh", "-c", "echo y | layerd keys export validator --unarmored-hex --unsafe --keyring-backend test --home " +
				v.Node.HomeDir(),
		}

		stdout, _, exportErr := v.Node.Exec(ctx, exportCmd, v.Node.Chain.Config().Env)
		require.NoError(exportErr)

		// Parse the output to extract the private key
		lines := strings.Split(string(stdout), "\n")
		var privKeyHex string
		for _, line := range lines {
			if len(line) >= 64 && !strings.Contains(line, "WARNING") {
				privKeyHex = line
				break
			}
		}

		require.NotEmpty(privKeyHex, "Failed to extract private key")

		// Create ECDSA private key from the exported key
		privKeyBytes, decodeErr := hex.DecodeString(privKeyHex)
		require.NoError(decodeErr)

		exportedPrivKey, privErr := crypto.ToECDSA(privKeyBytes)
		require.NoError(privErr)

		// Store the private key for later use in signing malicious attestations
		validators[i].EVMPriv = exportedPrivKey

		fmt.Printf("Validator %d - Private key loaded for signing\n", i)
	}

	// Now query the actual registered EVM addresses
	evmValidatorsRes, _, queryErr := validators[0].Node.ExecQuery(ctx, "bridge", "get-evm-validators")
	require.NoError(queryErr)
	var evmValidators QueryGetEvmValidatorsResponse
	unmarshalErr := json.Unmarshal(evmValidatorsRes, &evmValidators)
	require.NoError(unmarshalErr)

	fmt.Println("Registered EVM validators from bridge:")
	for i, val := range evmValidators.BridgeValidatorSet {
		fmt.Printf("  Validator %d: Address %s, Power %s\n", i, val.EthereumAddress, val.Power)
		// Assign the registered EVM addresses to our validator structs
		if i < len(validators) {
			validators[i].EVMAddr = "0x" + val.EthereumAddress
		}
	}

	// Verify we have the correct number of validators
	require.Equal(len(evmValidators.BridgeValidatorSet), len(validators), "Number of registered EVM validators should match number of test validators")

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

	// Query and compare EVM validators
	evmValidatorsRes2, _, err := validators[0].Node.ExecQuery(ctx, "bridge", "get-evm-validators")
	require.NoError(err)
	var evmValidators2 QueryGetEvmValidatorsResponse
	err = json.Unmarshal(evmValidatorsRes2, &evmValidators2)
	require.NoError(err)

	fmt.Println("Registered EVM validators:")
	for i, val := range evmValidators2.BridgeValidatorSet {
		fmt.Printf("  Validator %d: Address %s, Power %s\n", i, val.EthereumAddress, val.Power)
	}

	fmt.Println("Our assigned EVM addresses:")
	for i, v := range validators {
		fmt.Printf("  Validator %d: Address %s\n", i, v.EVMAddr)
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

	queryId := "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"
	timestamp := uint64(time.Now().UnixMilli())

	// Get the current checkpoint for use in malicious attestation
	checkpointRes, _, err := validators[0].Node.ExecQuery(ctx, "bridge", "get-validator-checkpoint")
	require.NoError(err)
	var checkpointData struct {
		Checkpoint string `json:"validator_checkpoint"`
	}
	err = json.Unmarshal(checkpointRes, &checkpointData)
	require.NoError(err)
	checkpointStr := checkpointData.Checkpoint
	fmt.Println("Current checkpoint:", checkpointStr)

	// Create a malicious attestation from validator[1] for slashing purposes
	// Modify the original attestation data but use same signatures
	maliciousValue := "000000000000000000000000000000000000000000000058528649cf90ee0000"

	// Create attestation data bytes for signing
	queryIdBytes, err := hex.DecodeString(queryId)
	require.NoError(err)
	checkpointBytes, err := hex.DecodeString(checkpointStr)
	require.NoError(err)

	attestTimestamp := uint64(time.Now().UnixMilli())

	// Generate signature from validator[1]
	targetValidator := validators[1]
	fmt.Printf("Target validator for slashing: %s (EVM: %s)\n", targetValidator.AccAddr, targetValidator.EVMAddr)

	type SnapshotData struct {
		QueryID                string `json:"query_id"`
		Value                  string `json:"value"`
		Timestamp              uint64 `json:"timestamp"`
		AggregatePower         uint64 `json:"aggregate_power"`
		PreviousTimestamp      uint64 `json:"previous_timestamp"`
		NextTimestamp          uint64 `json:"next_timestamp"`
		Checkpoint             string `json:"checkpoint"`
		AttestationTimestamp   uint64 `json:"attestation_timestamp"`
		LastConsensusTimestamp uint64 `json:"last_consensus_timestamp"`
	}

	malSnapshotData := SnapshotData{
		QueryID:                queryId,
		Value:                  maliciousValue,
		Timestamp:              timestamp,
		AggregatePower:         100,
		PreviousTimestamp:      timestamp - 1000,
		NextTimestamp:          timestamp + 1000,
		Checkpoint:             checkpointStr,
		AttestationTimestamp:   attestTimestamp,
		LastConsensusTimestamp: timestamp,
	}

	// Create snapshot data for malicious attestation
	snapshotBytes, err := encodeOracleAttestationData(
		queryIdBytes,
		malSnapshotData.Value,
		malSnapshotData.Timestamp,
		malSnapshotData.AggregatePower,
		malSnapshotData.PreviousTimestamp,
		malSnapshotData.NextTimestamp,
		checkpointBytes,
		malSnapshotData.AttestationTimestamp,
		malSnapshotData.LastConsensusTimestamp,
	)
	require.NoError(err)

	// Sign the snapshot
	msgHash := sha256.Sum256(snapshotBytes)
	signature, err := crypto.Sign(msgHash[:], targetValidator.EVMPriv)
	require.NoError(err)

	// Remove the recovery ID (V) - bridge module expects only R || S (64 bytes)
	signature = signature[:64]
	sigHex := hex.EncodeToString(signature)
	fmt.Println("sigHex (64 bytes):", sigHex)

	// Submit attestation evidence
	_, err = validators[0].Node.ExecTx(
		ctx,
		"validator",
		"bridge",
		"submit-attestation-evidence",
		validators[0].AccAddr, // creator address
		malSnapshotData.QueryID,
		malSnapshotData.Value,
		fmt.Sprintf("%d", malSnapshotData.Timestamp),
		fmt.Sprintf("%d", malSnapshotData.AggregatePower),    // aggregate power
		fmt.Sprintf("%d", malSnapshotData.PreviousTimestamp), // previous timestamp
		fmt.Sprintf("%d", malSnapshotData.NextTimestamp),     // next timestamp
		malSnapshotData.Checkpoint,
		fmt.Sprintf("%d", malSnapshotData.AttestationTimestamp),
		fmt.Sprintf("%d", malSnapshotData.LastConsensusTimestamp), // last consensus timestamp
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
	// This must match keeper.EncodeOracleAttestationData exactly

	// domainSeparator is bytes "tellorCurrentAttestation"
	NEW_REPORT_ATTESTATION_DOMAIN_SEPARATOR := []byte("tellorCurrentAttestation")
	// convert domain separator to bytes32
	var domainSepBytes32 [32]byte
	copy(domainSepBytes32[:], NEW_REPORT_ATTESTATION_DOMAIN_SEPARATOR)

	// convert queryId to bytes32
	var queryIdBytes32 [32]byte
	copy(queryIdBytes32[:], queryId)

	// convert value to bytes
	valueBytes, err := hex.DecodeString(value)
	if err != nil {
		return nil, err
	}

	// convert timestamps and power to big.Int
	timestampBig := new(big.Int).SetUint64(timestamp)
	aggregatePowerBig := new(big.Int).SetUint64(aggregatePower)
	previousTimestampBig := new(big.Int).SetUint64(previousTimestamp)
	nextTimestampBig := new(big.Int).SetUint64(nextTimestamp)
	attestationTimestampBig := new(big.Int).SetUint64(attestationTimestamp)
	lastConsensusTimestampBig := new(big.Int).SetUint64(lastConsensusTimestamp)

	// convert checkpoint to bytes32
	var checkpointBytes32 [32]byte
	copy(checkpointBytes32[:], checkpoint)

	// prepare ABI encoding types
	bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return nil, err
	}
	uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	bytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}

	arguments := abi.Arguments{
		{Type: bytes32Type}, // domain separator
		{Type: bytes32Type}, // queryId
		{Type: bytesType},   // value
		{Type: uint256Type}, // timestamp
		{Type: uint256Type}, // aggregatePower
		{Type: uint256Type}, // previousTimestamp
		{Type: uint256Type}, // nextTimestamp
		{Type: bytes32Type}, // checkpoint
		{Type: uint256Type}, // attestationTimestamp
		{Type: uint256Type}, // lastConsensusTimestamp
	}

	encodedData, err := arguments.Pack(
		domainSepBytes32,
		queryIdBytes32,
		valueBytes,
		timestampBig,
		aggregatePowerBig,
		previousTimestampBig,
		nextTimestampBig,
		checkpointBytes32,
		attestationTimestampBig,
		lastConsensusTimestampBig,
	)
	if err != nil {
		return nil, err
	}

	return crypto.Keccak256(encodedData), nil
}
