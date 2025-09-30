package e2e_test

import (
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
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// TestValsetSignatureSlashing tests the valset signature slashing mechanism
func TestValsetSignatureSlashing(t *testing.T) {
	require := require.New(t)

	t.Helper()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	cosmos.SetSDKConfig("tellor")

	// Use standard configuration with custom genesis modifications
	config := e2e.DefaultSetupConfig()
	config.ModifyGenesis = []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.dispute.params.team_address", sdk.MustAccAddressFromBech32("tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf").Bytes()),
		cosmos.NewGenesisKV("consensus.params.abci.vote_extensions_enable_height", "1"),
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "15s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "loya"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices.0.amount", "0.000025000000000000"),
	}

	chain, _, ctx := e2e.SetupChainWithCustomConfig(t, config)

	require.NoError(chain.RecoverKey(ctx, "team", teamMnemonic))
	require.NoError(chain.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: "tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf",
		Amount:  math.NewInt(1000000000000),
		Denom:   "loya",
	}))

	// Get validators using the helper
	validatorsInfo, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)

	// Setup validator info with EVM-specific fields
	type Validators struct {
		e2e.ValidatorInfo
		EVMPriv *ecdsa.PrivateKey
		EVMAddr string
	}

	validators := make([]Validators, len(validatorsInfo))
	for i, v := range validatorsInfo {
		fmt.Println("val", i, " Account Address: ", v.AccAddr)
		fmt.Println("val", i, " Validator Address: ", v.ValAddr)

		validators[i] = Validators{
			ValidatorInfo: v,
		}
	}

	// wait for vote extensions to register EVM addresses, then query them directly
	waitErr := testutil.WaitForBlocks(ctx, 5, validators[0].Node)
	require.NoError(waitErr)

	// define types for bridge validator queries
	type QueryBridgeValidator struct {
		EthereumAddress string `json:"ethereumAddress"`
		Power           string `json:"power"`
	}
	type QueryGetEvmValidatorsResponse struct {
		BridgeValidatorSet []QueryBridgeValidator `json:"bridge_validator_set"`
	}

	// get private keys and query the actual registered EVM addresses
	for i, v := range validators {
		exportCmd := []string{
			"sh", "-c", "echo y | layerd keys export validator --unarmored-hex --unsafe --keyring-backend test --home " +
				v.Node.HomeDir(),
		}

		stdout, _, exportErr := v.Node.Exec(ctx, exportCmd, v.Node.Chain.Config().Env)
		require.NoError(exportErr)

		// parse the output to extract the private key
		lines := strings.Split(string(stdout), "\n")
		var privKeyHex string
		for _, line := range lines {
			if len(line) >= 64 && !strings.Contains(line, "WARNING") {
				privKeyHex = line
				break
			}
		}

		require.NotEmpty(privKeyHex, "Failed to extract private key")

		// create ECDSA private key from the exported key
		privKeyBytes, decodeErr := hex.DecodeString(privKeyHex)
		require.NoError(decodeErr)

		exportedPrivKey, privErr := crypto.ToECDSA(privKeyBytes)
		require.NoError(privErr)

		// store the private key for later use in signing malicious valsets
		validators[i].EVMPriv = exportedPrivKey

		fmt.Printf("Validator %d - Private key loaded for signing\n", i)
	}

	// now query the actual registered EVM addresses
	evmValidatorsRes, _, queryErr := e2e.QueryWithTimeout(ctx, validators[0].Node, "bridge", "get-evm-validators")
	require.NoError(queryErr)
	var evmValidators QueryGetEvmValidatorsResponse
	unmarshalErr := json.Unmarshal(evmValidatorsRes, &evmValidators)
	require.NoError(unmarshalErr)

	fmt.Println("Registered EVM validators from bridge:")
	for i, val := range evmValidators.BridgeValidatorSet {
		fmt.Printf("  Validator %d: Address %s, Power %s\n", i, val.EthereumAddress, val.Power)
		// assign the registered EVM addresses to our validator structs
		if i < len(validators) {
			validators[i].EVMAddr = "0x" + val.EthereumAddress
		}
	}

	// verify we have the correct number of validators
	require.Equal(len(evmValidators.BridgeValidatorSet), len(validators), "Number of registered EVM validators should match number of test validators")

	// confirm that 2 validators are bonded
	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 2)

	// record initial validator tokens
	initialTokens := make([]math.Int, len(validators))
	for i, v := range validators {
		valInfo, err := chain.StakingQueryValidator(ctx, v.ValAddr)
		require.NoError(err)
		initialTokens[i] = valInfo.Tokens
		fmt.Println("Validator", i, "initial tokens:", initialTokens[i])
	}

	// wait for validator set updates to be processed
	err = testutil.WaitForBlocks(ctx, 3, validators[0].Node)
	require.NoError(err)

	valsetTimestamp := uint64(time.Now().UnixMilli())
	fmt.Printf("Using current time as valset timestamp: %d\n", valsetTimestamp)

	// create a fake validator set hash (modified from the actual one)
	fakeValsetHash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	powerThreshold := uint64(10000000) // total voting power

	// create malicious valset checkpoint from validator[1] for slashing purposes
	targetValidator := validators[1]
	fmt.Printf("Target validator for slashing: %s (EVM: %s)\n", targetValidator.AccAddr, targetValidator.EVMAddr)

	// encode the malicious valset checkpoint from the correct chain id
	chainId := "layer"
	maliciousCheckpoint, err := encodeValsetCheckpoint(powerThreshold, valsetTimestamp, fakeValsetHash, chainId)
	require.NoError(err)

	// sign the malicious checkpoint
	msgHash := sha256.Sum256(maliciousCheckpoint)
	signature, err := crypto.Sign(msgHash[:], targetValidator.EVMPriv)
	require.NoError(err)

	// remove the recovery ID (V) - bridge module expects only R || S (64 bytes)
	signature = signature[:64]
	sigHex := hex.EncodeToString(signature)
	fmt.Println("sigHex (64 bytes):", sigHex)

	// submit valset signature evidence
	_, err = validators[0].Node.ExecTx(
		ctx,
		"validator",
		"bridge",
		"submit-valset-signature-evidence",
		validators[0].AccAddr, // creator address
		fmt.Sprintf("%d", valsetTimestamp),
		fakeValsetHash,
		fmt.Sprintf("%d", powerThreshold),
		sigHex,
		"--keyring-dir", validators[0].Node.HomeDir(),
	)
	require.NoError(err)
	fmt.Println("Submitted valset signature evidence against validator", targetValidator.ValAddr)

	// wait for evidence to be processed
	err = testutil.WaitForBlocks(ctx, 3, validators[0].Node)
	require.NoError(err)

	// check if the validator was slashed
	slashedValInfo, err := chain.StakingQueryValidator(ctx, targetValidator.ValAddr)
	require.NoError(err)

	// verify the validator was slashed
	fmt.Println("Validator initial tokens:", initialTokens[1])
	fmt.Println("Validator final tokens:", slashedValInfo.Tokens)
	require.True(slashedValInfo.Tokens.LT(initialTokens[1]), "Validator should have been slashed")

	// check if the validator was jailed
	require.True(slashedValInfo.Jailed, "Validator should have been jailed")
}

// TestValsetSignatureSlashingWithDifferentChainId tests that a validator cannot be slashed if the chain id is different
func TestValsetSignatureSlashingWithDifferentChainId(t *testing.T) {
	require := require.New(t)

	t.Helper()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	cosmos.SetSDKConfig("tellor")

	// Use standard configuration with custom genesis modifications
	config := e2e.DefaultSetupConfig()
	config.NumFullNodes = 1
	config.ModifyGenesis = []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.dispute.params.team_address", sdk.MustAccAddressFromBech32("tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf").Bytes()),
		cosmos.NewGenesisKV("consensus.params.abci.vote_extensions_enable_height", "1"),
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "15s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "loya"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices.0.amount", "0.000025000000000000"),
	}

	chain, _, ctx := e2e.SetupChainWithCustomConfig(t, config)

	require.NoError(chain.RecoverKey(ctx, "team", teamMnemonic))
	require.NoError(chain.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: "tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf",
		Amount:  math.NewInt(1000000000000),
		Denom:   "loya",
	}))

	// Get validators using the helper
	validatorsInfo, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)

	// Setup validator info with EVM-specific fields
	type Validators struct {
		e2e.ValidatorInfo
		EVMPriv *ecdsa.PrivateKey
		EVMAddr string
	}

	validators := make([]Validators, len(validatorsInfo))
	for i, v := range validatorsInfo {
		fmt.Println("val", i, " Account Address: ", v.AccAddr)
		fmt.Println("val", i, " Validator Address: ", v.ValAddr)

		validators[i] = Validators{
			ValidatorInfo: v,
		}
	}

	// wait for vote extensions to register EVM addresses, then query them directly
	waitErr := testutil.WaitForBlocks(ctx, 5, validators[0].Node)
	require.NoError(waitErr)

	// define types for bridge validator queries
	type QueryBridgeValidator struct {
		EthereumAddress string `json:"ethereumAddress"`
		Power           string `json:"power"`
	}
	type QueryGetEvmValidatorsResponse struct {
		BridgeValidatorSet []QueryBridgeValidator `json:"bridge_validator_set"`
	}

	// get private keys and query the actual registered EVM addresses
	for i, v := range validators {
		exportCmd := []string{
			"sh", "-c", "echo y | layerd keys export validator --unarmored-hex --unsafe --keyring-backend test --home " +
				v.Node.HomeDir(),
		}

		stdout, _, exportErr := v.Node.Exec(ctx, exportCmd, v.Node.Chain.Config().Env)
		require.NoError(exportErr)

		// parse the output to extract the private key
		lines := strings.Split(string(stdout), "\n")
		var privKeyHex string
		for _, line := range lines {
			if len(line) >= 64 && !strings.Contains(line, "WARNING") {
				privKeyHex = line
				break
			}
		}

		require.NotEmpty(privKeyHex, "Failed to extract private key")

		// create ECDSA private key from the exported key
		privKeyBytes, decodeErr := hex.DecodeString(privKeyHex)
		require.NoError(decodeErr)

		exportedPrivKey, privErr := crypto.ToECDSA(privKeyBytes)
		require.NoError(privErr)

		// store the private key for later use in signing malicious valsets
		validators[i].EVMPriv = exportedPrivKey

		fmt.Printf("Validator %d - Private key loaded for signing\n", i)
	}

	// now query the actual registered EVM addresses
	evmValidatorsRes, _, queryErr := e2e.QueryWithTimeout(ctx, validators[0].Node, "bridge", "get-evm-validators")
	require.NoError(queryErr)
	var evmValidators QueryGetEvmValidatorsResponse
	unmarshalErr := json.Unmarshal(evmValidatorsRes, &evmValidators)
	require.NoError(unmarshalErr)

	fmt.Println("Registered EVM validators from bridge:")
	for i, val := range evmValidators.BridgeValidatorSet {
		fmt.Printf("  Validator %d: Address %s, Power %s\n", i, val.EthereumAddress, val.Power)
		// assign the registered EVM addresses to our validator structs
		if i < len(validators) {
			validators[i].EVMAddr = "0x" + val.EthereumAddress
		}
	}

	// verify we have the correct number of validators
	require.Equal(len(evmValidators.BridgeValidatorSet), len(validators), "Number of registered EVM validators should match number of test validators")

	// confirm that 2 validators are bonded
	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 2)

	// record initial validator tokens
	initialTokens := make([]math.Int, len(validators))
	for i, v := range validators {
		valInfo, err := chain.StakingQueryValidator(ctx, v.ValAddr)
		require.NoError(err)
		initialTokens[i] = valInfo.Tokens
		fmt.Println("Validator", i, "initial tokens:", initialTokens[i])
	}

	// wait for validator set updates to be processed
	err = testutil.WaitForBlocks(ctx, 3, validators[0].Node)
	require.NoError(err)

	valsetTimestamp := uint64(time.Now().UnixMilli())
	fmt.Printf("Using current time as valset timestamp: %d\n", valsetTimestamp)

	// create a fake validator set hash (modified from the actual one)
	fakeValsetHash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	powerThreshold := uint64(10000000) // total voting power

	// create malicious valset checkpoint from validator[1] for slashing purposes
	targetValidator := validators[1]
	fmt.Printf("Target validator for slashing: %s (EVM: %s)\n", targetValidator.AccAddr, targetValidator.EVMAddr)

	// encode the malicious valset checkpoint from the correct chain id
	chainId := "layertest-4"
	maliciousCheckpoint, err := encodeValsetCheckpoint(powerThreshold, valsetTimestamp, fakeValsetHash, chainId)
	require.NoError(err)

	// sign the malicious checkpoint
	msgHash := sha256.Sum256(maliciousCheckpoint)
	signature, err := crypto.Sign(msgHash[:], targetValidator.EVMPriv)
	require.NoError(err)

	// remove the recovery ID (V) - bridge module expects only R || S (64 bytes)
	signature = signature[:64]
	sigHex := hex.EncodeToString(signature)
	fmt.Println("sigHex (64 bytes):", sigHex)

	// submit valset signature evidence
	_, err = validators[0].Node.ExecTx(
		ctx,
		"validator",
		"bridge",
		"submit-valset-signature-evidence",
		validators[0].AccAddr, // creator address
		fmt.Sprintf("%d", valsetTimestamp),
		fakeValsetHash,
		fmt.Sprintf("%d", powerThreshold),
		sigHex,
		"--keyring-dir", validators[0].Node.HomeDir(),
	)
	require.ErrorContains(err, "operator address not found")
	fmt.Println("Submitted valset signature evidence against validator that errored out as expected", targetValidator.ValAddr)

	// wait for evidence to be processed
	err = testutil.WaitForBlocks(ctx, 3, validators[0].Node)
	require.NoError(err)

	// check if the validator was slashed
	slashedValInfo, err := chain.StakingQueryValidator(ctx, targetValidator.ValAddr)
	require.NoError(err)

	// verify the validator was slashed
	fmt.Println("Validator initial tokens:", initialTokens[1])
	fmt.Println("Validator final tokens:", slashedValInfo.Tokens)
	require.True(slashedValInfo.Tokens.Equal(initialTokens[1]), "Validator should have been slashed")

	// check if the validator was jailed
	require.False(slashedValInfo.Jailed, "Validator should have been jailed")
}

// encodeValsetCheckpoint replicates the keeper's EncodeValsetCheckpoint function
func encodeValsetCheckpoint(powerThreshold, validatorTimestamp uint64, validatorSetHash, chainId string) ([]byte, error) {
	// Create domain separator by ABI encoding "checkpoint" and chain ID
	// This matches the Solidity implementation: keccak256(abi.encode("checkpoint", chainId))
	stringType, err := abi.NewType("string", "", nil)
	if err != nil {
		return nil, err
	}

	// ABI encode "checkpoint" and chain ID (both as strings)
	domainSeparatorArgs := abi.Arguments{
		{Type: stringType},
		{Type: stringType},
	}
	domainSeparatorEncoded, err := domainSeparatorArgs.Pack("checkpoint", chainId)
	if err != nil {
		return nil, err
	}
	domainSeparator := crypto.Keccak256(domainSeparatorEncoded)

	// Convert domain separator to fixed size 32 bytes
	var domainSeparatorFixSize [32]byte
	copy(domainSeparatorFixSize[:], domainSeparator)

	// convert validatorSetHash to bytes
	validatorSetHashBytes, err := hex.DecodeString(validatorSetHash)
	if err != nil {
		return nil, err
	}

	// convert validatorSetHash to a fixed size 32 bytes
	var validatorSetHashFixSize [32]byte
	copy(validatorSetHashFixSize[:], validatorSetHashBytes)

	// convert powerThreshold and validatorTimestamp to *big.Int for ABI encoding
	powerThresholdBigInt := new(big.Int).SetUint64(powerThreshold)
	validatorTimestampBigInt := new(big.Int).SetUint64(validatorTimestamp)

	bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return nil, err
	}
	uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}

	// prepare the types for encoding
	arguments := abi.Arguments{
		{Type: bytes32Type},
		{Type: uint256Type},
		{Type: uint256Type},
		{Type: bytes32Type},
	}

	// encode the arguments
	encodedCheckpointData, err := arguments.Pack(
		domainSeparatorFixSize,
		powerThresholdBigInt,
		validatorTimestampBigInt,
		validatorSetHashFixSize,
	)
	if err != nil {
		return nil, err
	}

	checkpoint := crypto.Keccak256(encodedCheckpointData)
	return checkpoint, nil
}
