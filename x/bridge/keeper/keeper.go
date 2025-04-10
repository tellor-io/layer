package keeper

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/bridge/types"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService storetypes.KVStoreService

		Schema                       collections.Schema
		Params                       collections.Item[types.Params]
		BridgeValset                 collections.Item[types.BridgeValidatorSet]
		ValidatorCheckpoint          collections.Item[types.ValidatorCheckpoint]
		WithdrawalId                 collections.Item[types.WithdrawalId]
		OperatorToEVMAddressMap      collections.Map[string, types.EVMAddress]
		EVMAddressRegisteredMap      collections.Map[string, types.EVMAddressRegistered]
		BridgeValsetSignaturesMap    collections.Map[uint64, types.BridgeValsetSignatures]
		ValidatorCheckpointParamsMap collections.Map[uint64, types.ValidatorCheckpointParams]
		ValidatorCheckpointIdxMap    collections.Map[uint64, types.CheckpointTimestamp]
		LatestCheckpointIdx          collections.Item[types.CheckpointIdx]
		BridgeValsetByTimestampMap   collections.Map[uint64, types.BridgeValidatorSet]
		ValsetTimestampToIdxMap      collections.Map[uint64, types.CheckpointIdx]
		AttestSnapshotsByReportMap   collections.Map[[]byte, types.AttestationSnapshots]
		AttestSnapshotDataMap        collections.Map[[]byte, types.AttestationSnapshotData]
		SnapshotToAttestationsMap    collections.Map[[]byte, types.OracleAttestations]
		AttestRequestsByHeightMap    collections.Map[uint64, types.AttestationRequests]
		DepositIdClaimedMap          collections.Map[uint64, types.DepositClaimed]
		SnapshotLimit                collections.Item[types.SnapshotLimit] // limit of number of attestation requests per block

		stakingKeeper  types.StakingKeeper
		oracleKeeper   types.OracleKeeper
		bankKeeper     types.BankKeeper
		reporterKeeper types.ReporterKeeper
		disputeKeeper  types.DisputeKeeper
		authority      string
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	stakingKeeper types.StakingKeeper,
	oracleKeeper types.OracleKeeper,
	bankKeeper types.BankKeeper,
	reporterKeeper types.ReporterKeeper,
	disputeKeeper types.DisputeKeeper,
	authority string,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:                          cdc,
		storeService:                 storeService,
		Params:                       collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		BridgeValset:                 collections.NewItem(sb, types.BridgeValsetKey, "bridge_valset", codec.CollValue[types.BridgeValidatorSet](cdc)),
		ValidatorCheckpoint:          collections.NewItem(sb, types.ValidatorCheckpointKey, "validator_checkpoint", codec.CollValue[types.ValidatorCheckpoint](cdc)),
		WithdrawalId:                 collections.NewItem(sb, types.WithdrawalIdKey, "withdrawal_id", codec.CollValue[types.WithdrawalId](cdc)),
		OperatorToEVMAddressMap:      collections.NewMap(sb, types.OperatorToEVMAddressMapKey, "operator_to_evm_address_map", collections.StringKey, codec.CollValue[types.EVMAddress](cdc)),
		EVMAddressRegisteredMap:      collections.NewMap(sb, types.EVMAddressRegisteredMapKey, "evm_address_registered_map", collections.StringKey, codec.CollValue[types.EVMAddressRegistered](cdc)),
		BridgeValsetSignaturesMap:    collections.NewMap(sb, types.BridgeValsetSignaturesMapKey, "bridge_valset_signatures_map", collections.Uint64Key, codec.CollValue[types.BridgeValsetSignatures](cdc)),
		ValidatorCheckpointParamsMap: collections.NewMap(sb, types.ValidatorCheckpointParamsMapKey, "validator_checkpoint_params_map", collections.Uint64Key, codec.CollValue[types.ValidatorCheckpointParams](cdc)),
		ValidatorCheckpointIdxMap:    collections.NewMap(sb, types.ValidatorCheckpointIdxMapKey, "validator_checkpoint_idx_map", collections.Uint64Key, codec.CollValue[types.CheckpointTimestamp](cdc)),
		LatestCheckpointIdx:          collections.NewItem(sb, types.LatestCheckpointIdxKey, "latest_checkpoint_idx", codec.CollValue[types.CheckpointIdx](cdc)),
		BridgeValsetByTimestampMap:   collections.NewMap(sb, types.BridgeValsetByTimestampMapKey, "bridge_valset_by_timestamp_map", collections.Uint64Key, codec.CollValue[types.BridgeValidatorSet](cdc)),
		ValsetTimestampToIdxMap:      collections.NewMap(sb, types.ValsetTimestampToIdxMapKey, "valset_timestamp_to_idx_map", collections.Uint64Key, codec.CollValue[types.CheckpointIdx](cdc)),
		AttestSnapshotsByReportMap:   collections.NewMap(sb, types.AttestSnapshotsByReportMapKey, "attest_snapshots_by_report_map", collections.BytesKey, codec.CollValue[types.AttestationSnapshots](cdc)),
		AttestSnapshotDataMap:        collections.NewMap(sb, types.AttestSnapshotDataMapKey, "attest_snapshot_data_map", collections.BytesKey, codec.CollValue[types.AttestationSnapshotData](cdc)),
		SnapshotToAttestationsMap:    collections.NewMap(sb, types.SnapshotToAttestationsMapKey, "snapshot_to_attestations_map", collections.BytesKey, codec.CollValue[types.OracleAttestations](cdc)),
		AttestRequestsByHeightMap:    collections.NewMap(sb, types.AttestRequestsByHeightMapKey, "attest_requests_by_height_map", collections.Uint64Key, codec.CollValue[types.AttestationRequests](cdc)),
		DepositIdClaimedMap:          collections.NewMap(sb, types.DepositIdClaimedMapKey, "deposit_id_claimed_map", collections.Uint64Key, codec.CollValue[types.DepositClaimed](cdc)),
		SnapshotLimit:                collections.NewItem(sb, types.SnapshotLimitKey, "snapshot_limit", codec.CollValue[types.SnapshotLimit](cdc)), // attestation requests per block limit

		stakingKeeper:  stakingKeeper,
		oracleKeeper:   oracleKeeper,
		bankKeeper:     bankKeeper,
		reporterKeeper: reporterKeeper,
		disputeKeeper:  disputeKeeper,
		authority:      authority,
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) GetCurrentValidatorsEVMCompatible(ctx context.Context) ([]*types.BridgeValidator, error) {
	validators, err := k.stakingKeeper.GetAllValidators(ctx)
	if err != nil {
		return nil, err
	}

	var bridgeValset []*types.BridgeValidator

	for _, validator := range validators {
		evmAddress, err := k.OperatorToEVMAddressMap.Get(ctx, validator.GetOperator())
		if err != nil {
			continue // Skip this validator if the EVM address is not found
		}
		adjustedPower := validator.GetConsensusPower(layertypes.PowerReduction)
		if adjustedPower == 0 {
			continue // Skip this validator if the adjusted power is 0
		}
		bridgeValset = append(bridgeValset, &types.BridgeValidator{
			EthereumAddress: evmAddress.EVMAddress,
			Power:           uint64(adjustedPower),
		})
	}

	if len(bridgeValset) == 0 {
		return nil, errors.New("no validators found")
	}

	// Sort the validators
	sort.Slice(bridgeValset, func(i, j int) bool {
		if bridgeValset[i].Power == bridgeValset[j].Power {
			// If power is equal, sort alphabetically
			return bytes.Compare(bridgeValset[i].EthereumAddress, bridgeValset[j].EthereumAddress) < 0
		}
		// Otherwise, sort by power in descending order
		return bridgeValset[i].Power > bridgeValset[j].Power
	})

	return bridgeValset, nil
}

func (k Keeper) GetCurrentValidatorSetEVMCompatible(ctx context.Context) (*types.BridgeValidatorSet, error) {
	// use GetBridgeValidators to get the current bridge validator set
	bridgeValset, err := k.GetCurrentValidatorsEVMCompatible(ctx)
	if err != nil {
		return nil, err
	}

	return &types.BridgeValidatorSet{BridgeValidatorSet: bridgeValset}, nil
}

// function for loading last saved bridge validator set and comparing it to current set
func (k Keeper) CompareAndSetBridgeValidators(ctx context.Context) (bool, error) {
	// load current validator set in EVM compatible format

	currentValidatorSetEVMCompatible, err := k.GetCurrentValidatorSetEVMCompatible(ctx)
	if err != nil {
		k.Logger(ctx).Info("No current validator set found")
		return false, err
	}

	lastSavedBridgeValidators, err := k.BridgeValset.Get(ctx)
	if err != nil {
		k.Logger(ctx).Info("No saved bridge validator set found")
		err := k.BridgeValset.Set(ctx, *currentValidatorSetEVMCompatible)
		if err != nil {
			k.Logger(ctx).Info("Error setting bridge validator set: ", "error", err)
			return false, err
		}
		error := k.SetBridgeValidatorParams(ctx, currentValidatorSetEVMCompatible)
		if error != nil {
			k.Logger(ctx).Info("Error setting bridge validator params: ", "error", error)
			return false, error
		}
		return false, err
	}
	stale, err := k.LastSavedValidatorSetStale(ctx)
	if err != nil {
		k.Logger(ctx).Info("Error checking if last saved validator set is stale", "error", err)
		return false, err
	}
	if bytes.Equal(k.cdc.MustMarshal(&lastSavedBridgeValidators), k.cdc.MustMarshal(currentValidatorSetEVMCompatible)) && !stale {
		return false, nil
	} else if k.PowerDiff(ctx, lastSavedBridgeValidators, *currentValidatorSetEVMCompatible) < 5e4 && !stale {
		return false, nil
	} else {
		err := k.BridgeValset.Set(ctx, *currentValidatorSetEVMCompatible)
		if err != nil {
			k.Logger(ctx).Info("Error setting bridge validator set: ", "error", err)
			return false, err
		}
		error := k.SetBridgeValidatorParams(ctx, currentValidatorSetEVMCompatible)
		if error != nil {
			k.Logger(ctx).Info("Error setting bridge validator params: ", "error", error)
			return false, error
		}
		return true, nil
	}
}

func (k Keeper) SetBridgeValidatorParams(ctx context.Context, bridgeValidatorSet *types.BridgeValidatorSet) error {
	var totalPower uint64
	for _, validator := range bridgeValidatorSet.BridgeValidatorSet {
		totalPower += validator.GetPower()
	}
	powerThreshold := totalPower * 2 / 3

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	validatorTimestamp := uint64(sdkCtx.BlockTime().UnixMilli())

	// calculate validator set hash
	_, validatorSetHash, err := k.EncodeAndHashValidatorSet(ctx, bridgeValidatorSet)
	if err != nil {
		k.Logger(ctx).Info("Error encoding and hashing validator set: ", "error", err)
		return err
	}

	// calculate validator set checkpoint
	checkpoint, err := k.CalculateValidatorSetCheckpoint(ctx, powerThreshold, validatorTimestamp, validatorSetHash)
	if err != nil {
		k.Logger(ctx).Info("Error calculating validator set checkpoint: ", "error", err)
		return err
	}

	// Set the validator checkpoint
	checkpointType := types.ValidatorCheckpoint{
		Checkpoint: checkpoint,
	}

	error := k.ValidatorCheckpoint.Set(ctx, checkpointType)
	if error != nil {
		k.Logger(ctx).Info("Error setting validator checkpoint: ", "error", error)
		return error
	}

	// Set the bridge valset by timestamp
	err = k.BridgeValsetByTimestampMap.Set(ctx, validatorTimestamp, *bridgeValidatorSet)
	if err != nil {
		k.Logger(ctx).Info("Error setting bridge valset by timestamp: ", "error", err)
		return err
	}

	valsetIdx, err := k.LatestCheckpointIdx.Get(ctx)
	if err != nil {
		k.Logger(ctx).Info("Error getting latest checkpoint index: ", "error", err)
		return err
	}
	var valsetSigs *types.BridgeValsetSignatures
	if valsetIdx.Index == 0 {
		valsetSigs = types.NewBridgeValsetSignatures(len(bridgeValidatorSet.BridgeValidatorSet))
	} else {
		previousValsetTimestamp, err := k.ValidatorCheckpointIdxMap.Get(ctx, valsetIdx.Index-1)
		if err != nil {
			k.Logger(ctx).Info("Error getting previous valset timestamp: ", "error", err)
			return err
		}
		previousValset, err := k.BridgeValsetByTimestampMap.Get(ctx, previousValsetTimestamp.Timestamp)
		if err != nil {
			k.Logger(ctx).Info("Error getting previous valset: ", "error", err)
			return err
		}

		valsetSigs = types.NewBridgeValsetSignatures(len(previousValset.BridgeValidatorSet))
	}
	err = k.BridgeValsetSignaturesMap.Set(ctx, validatorTimestamp, *valsetSigs)
	if err != nil {
		k.Logger(ctx).Info("Error setting bridge valset signatures: ", "error", err)
		return err
	}
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"new_bridge_validator_set",
			sdk.NewAttribute("timestamp", strconv.FormatUint(validatorTimestamp, 10)),
		),
	})
	return nil
}

func (k Keeper) CalculateValidatorSetCheckpoint(
	ctx context.Context,
	powerThreshold uint64,
	validatorTimestamp uint64,
	validatorSetHash []byte,
) ([]byte, error) {
	// Define the domain separator for the validator set hash, fixed size 32 bytes
	VALIDATOR_SET_HASH_DOMAIN_SEPARATOR := []byte("checkpoint")
	var domainSeparatorFixSize [32]byte
	copy(domainSeparatorFixSize[:], VALIDATOR_SET_HASH_DOMAIN_SEPARATOR)

	// Convert validatorSetHash to a fixed size 32 bytes
	var validatorSetHashFixSize [32]byte
	copy(validatorSetHashFixSize[:], validatorSetHash)

	// Convert powerThreshold and validatorTimestamp to *big.Int for ABI encoding
	powerThresholdBigInt := new(big.Int).SetUint64(powerThreshold)
	validatorTimestampBigInt := new(big.Int).SetUint64(validatorTimestamp)

	Bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		k.Logger(ctx).Warn("Error creating new bytes32 ABI type", "error", err)
		return nil, err
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		k.Logger(ctx).Warn("Error creating new uint256 ABI type", "error", err)
		return nil, err
	}

	// Prepare the types for encoding
	arguments := abi.Arguments{
		{Type: Bytes32Type},
		{Type: Uint256Type},
		{Type: Uint256Type},
		{Type: Bytes32Type},
	}

	// Encode the arguments
	encodedCheckpointData, err := arguments.Pack(
		domainSeparatorFixSize,
		powerThresholdBigInt,
		validatorTimestampBigInt,
		validatorSetHashFixSize,
	)
	if err != nil {
		k.Logger(ctx).Warn("Error encoding arguments", "error", err)
		return nil, err
	}

	checkpoint := crypto.Keccak256(encodedCheckpointData)

	// save checkpoint params
	checkpointParams := types.ValidatorCheckpointParams{
		Checkpoint:     checkpoint,
		ValsetHash:     validatorSetHash,
		Timestamp:      validatorTimestamp,
		PowerThreshold: powerThreshold,
	}
	err = k.ValidatorCheckpointParamsMap.Set(ctx, validatorTimestamp, checkpointParams)
	if err != nil {
		k.Logger(ctx).Info("Error setting validator checkpoint params: ", "error", err)
		return nil, err
	}

	// load checkpoint index. if not found, set to 0
	checkpointIdx := types.CheckpointIdx{}
	lastCheckpointIdx, err := k.LatestCheckpointIdx.Get(ctx)
	if err != nil {
		k.Logger(ctx).Info("Error getting latest checkpoint index: ", "error", err)
		checkpointIdx.Index = 0
	} else {
		checkpointIdx.Index = lastCheckpointIdx.Index + 1
	}

	// save checkpoint index
	err = k.ValidatorCheckpointIdxMap.Set(ctx, checkpointIdx.Index, types.CheckpointTimestamp{Timestamp: validatorTimestamp})
	if err != nil {
		k.Logger(ctx).Info("Error setting validator checkpoint index: ", "error", err)
		return nil, err
	}

	// save latest checkpoint index
	err = k.LatestCheckpointIdx.Set(ctx, checkpointIdx)
	if err != nil {
		k.Logger(ctx).Info("Error setting latest checkpoint index: ", "error", err)
		return nil, err
	}
	err = k.ValsetTimestampToIdxMap.Set(ctx, validatorTimestamp, checkpointIdx)
	if err != nil {
		k.Logger(ctx).Info("Error setting valset timestamp to index: ", "error", err)
		return nil, err
	}
	return checkpoint, nil
}

// check whether last saved validator set is too old (more than 2 weeks). we set this to 2 weeks
// to give bridge relayer enough time to update the validator set in bridge contracts before expiration
func (k Keeper) LastSavedValidatorSetStale(ctx context.Context) (bool, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// get current block timestamp
	blockTime := sdkCtx.BlockTime().Add(1 * time.Second)
	valsetTimestamp, err := k.GetValidatorSetTimestampBefore(ctx, uint64(blockTime.UnixMilli()))
	if err != nil {
		k.Logger(ctx).Info("Error getting validator set timestamp before", "error", err)
		return false, err
	}
	twoWeeksAgo := blockTime.Add(-2 * time.Hour * 24 * 7)
	if valsetTimestamp < uint64(twoWeeksAgo.UnixMilli()) {
		return true, nil
	}
	return false, nil
}

func (k Keeper) GetValidatorCheckpointFromStorage(ctx context.Context) (*types.ValidatorCheckpoint, error) {
	checkpoint, err := k.ValidatorCheckpoint.Get(ctx)
	if err != nil {
		k.Logger(ctx).Error("Failed to get validator checkpoint", "error", err)
		return nil, err
	}
	return &checkpoint, nil
}

func (k Keeper) GetValidatorTimestampByIdxFromStorage(ctx context.Context, checkpointIdx uint64) (types.CheckpointTimestamp, error) {
	return k.ValidatorCheckpointIdxMap.Get(ctx, checkpointIdx)
}

func (k Keeper) GetValidatorSetIndexByTimestamp(ctx context.Context, timestamp uint64) (uint64, error) {
	checkpointIdx, err := k.ValsetTimestampToIdxMap.Get(ctx, timestamp)
	if err != nil {
		k.Logger(ctx).Error("Failed to get validator set index by timestamp", "error", err)
		return 0, err
	}
	return checkpointIdx.Index, nil
}

func (k Keeper) GetCurrentValidatorSetTimestamp(ctx context.Context) (uint64, error) {
	checkpointIdx, err := k.LatestCheckpointIdx.Get(ctx)
	if err != nil {
		k.Logger(ctx).Error("Failed to get latest checkpoint index", "error", err)
		return 0, err
	}
	checkpointTimestamp, err := k.ValidatorCheckpointIdxMap.Get(ctx, checkpointIdx.Index)
	if err != nil {
		k.Logger(ctx).Error("Failed to get checkpoint timestamp", "error", err)
		return 0, err
	}
	return checkpointTimestamp.Timestamp, nil
}

func (k Keeper) GetValidatorSetSignaturesFromStorage(ctx context.Context, timestamp uint64) (*types.BridgeValsetSignatures, error) {
	valsetSigs, err := k.BridgeValsetSignaturesMap.Get(ctx, timestamp)
	if err != nil {
		k.Logger(ctx).Error("Failed to get bridge valset signatures", "error", err)
		return nil, err
	}
	return &valsetSigs, nil
}

func (k Keeper) EncodeAndHashValidatorSet(ctx context.Context, validatorSet *types.BridgeValidatorSet) (encodedBridgeValidatorSet, bridgeValidatorSetHash []byte, err error) {
	// Define Go equivalent of the Solidity Validator struct
	type Validator struct {
		Addr  common.Address
		Power *big.Int
	}

	// Convert validatorSet to a slice of the Validator struct defined above
	var validators []Validator
	for _, v := range validatorSet.BridgeValidatorSet {
		addr := common.BytesToAddress(v.EthereumAddress)
		power := big.NewInt(0).SetUint64(v.Power)
		validators = append(validators, Validator{Addr: addr, Power: power})
	}

	// Solidity dynamic array encoding starts with the offset to the data
	// followed by the length of the array itself. Since we're directly encoding the array next,
	// the data starts immediately after these two fields, which is at 64 bytes offset.
	offsetToData := make([]byte, 32)
	binary.BigEndian.PutUint64(offsetToData[24:], uint64(32)) // 64 bytes offset to the start of the data

	// Encode the length of the array
	arrayLength := len(validators)
	lengthEncoded := make([]byte, 32)
	binary.BigEndian.PutUint64(lengthEncoded[24:], uint64(arrayLength))

	AddressType, err := abi.NewType("address", "", nil)
	if err != nil {
		k.Logger(ctx).Warn("Error creating new address ABI type", "error", err)
		return nil, nil, err
	}
	UintType, err := abi.NewType("uint256", "", nil)
	if err != nil {
		k.Logger(ctx).Warn("Error creating new uint256 ABI type", "error", err)
		return nil, nil, err
	}

	// Encode each Validator struct
	var encodedVals []byte
	for _, val := range validators {
		encodedVal, err := abi.Arguments{
			{Type: AddressType},
			{Type: UintType},
		}.Pack(val.Addr, val.Power)
		if err != nil {
			return nil, nil, err
		}
		encodedVals = append(encodedVals, encodedVal...)
	}

	// Concatenate the offset, length, and encoded validators
	finalEncoded := append(offsetToData, lengthEncoded...)
	finalEncoded = append(finalEncoded, encodedVals...)

	// Hash the encoded bytes
	valSetHash := crypto.Keccak256(finalEncoded)

	return finalEncoded, valSetHash, nil
}

func (k Keeper) PowerDiff(ctx context.Context, b, c types.BridgeValidatorSet) int64 {
	const precision = 1e6

	powers := map[string]int64{}
	var totalPower int64
	for _, bv := range b.BridgeValidatorSet {
		power := int64(bv.GetPower())
		powers[string(bv.EthereumAddress)] = power
		totalPower += power
	}

	for _, bv := range c.BridgeValidatorSet {
		if val, ok := powers[string(bv.EthereumAddress)]; ok {
			powers[string(bv.EthereumAddress)] = val - int64(bv.GetPower())
		} else {
			powers[string(bv.EthereumAddress)] = -int64(bv.GetPower())
		}
	}

	var delta int64
	for _, v := range powers {
		delta += absInt64(v)
	}

	if totalPower == 0 {
		return 0
	}

	relativeDiff := (delta * precision) / totalPower
	return relativeDiff
}

func absInt64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func (k Keeper) EVMAddressFromSignatures(ctx context.Context, sigA, sigB []byte, operatorAddress string) (common.Address, error) {
	msgA := fmt.Sprintf("TellorLayer: Initial bridge signature A for operator %s", operatorAddress)
	msgB := fmt.Sprintf("TellorLayer: Initial bridge signature B for operator %s", operatorAddress)

	// convert messages to bytes
	msgBytesA := []byte(msgA)
	msgBytesB := []byte(msgB)

	// hash messages
	msgHashBytes32A := sha256.Sum256(msgBytesA)
	msgHashBytesA := msgHashBytes32A[:]

	msgHashBytes32B := sha256.Sum256(msgBytesB)
	msgHashBytesB := msgHashBytes32B[:]

	// hash the hash, since the keyring signer automatically hashes the message
	msgDoubleHashBytes32A := sha256.Sum256(msgHashBytesA)
	msgDoubleHashBytesA := msgDoubleHashBytes32A[:]

	msgDoubleHashBytes32B := sha256.Sum256(msgHashBytesB)
	msgDoubleHashBytesB := msgDoubleHashBytes32B[:]

	addressesA, err := k.TryRecoverAddressWithBothIDs(sigA, msgDoubleHashBytesA)
	if err != nil {
		k.Logger(ctx).Warn("Error trying to recover address with both IDs", "error", err)
		return common.Address{}, err
	}
	addressesB, err := k.TryRecoverAddressWithBothIDs(sigB, msgDoubleHashBytesB)
	if err != nil {
		k.Logger(ctx).Warn("Error trying to recover address with both IDs", "error", err)
		return common.Address{}, err
	}

	// check if addresses match
	if bytes.Equal(addressesA[0].Bytes(), addressesB[0].Bytes()) || bytes.Equal(addressesA[0].Bytes(), addressesB[1].Bytes()) {
		return addressesA[0], nil
	} else if bytes.Equal(addressesA[1].Bytes(), addressesB[0].Bytes()) || bytes.Equal(addressesA[1].Bytes(), addressesB[1].Bytes()) {
		return addressesA[1], nil
	} else {
		k.Logger(ctx).Warn("EVM addresses do not match")
		return common.Address{}, fmt.Errorf("EVM addresses do not match")
	}
}

func (k Keeper) TryRecoverAddressWithBothIDs(sig, msgHash []byte) ([]common.Address, error) {
	var addrs []common.Address
	for _, id := range []byte{0, 1} {
		sigWithID := append(sig[:64], id)
		pubKey, err := crypto.SigToPub(msgHash, sigWithID)
		if err != nil {
			return []common.Address{}, err
		}
		recoveredAddr := crypto.PubkeyToAddress(*pubKey)
		addrs = append(addrs, recoveredAddr)
	}
	return addrs, nil
}

func (k Keeper) SetEVMAddressByOperator(ctx context.Context, operatorAddr string, evmAddr []byte) error {
	evmAddrType := types.EVMAddress{
		EVMAddress: evmAddr,
	}

	// check if the EVM address is already registered
	evmAddressString := common.Bytes2Hex(evmAddr)
	isRegistered, err := k.EVMAddressRegisteredMap.Has(ctx, evmAddressString)
	if err != nil {
		k.Logger(ctx).Info("Error checking if EVM address is registered", "error", err)
		return err
	}
	if isRegistered {
		k.Logger(ctx).Info("EVM address already registered, skipping",
			"evm_address", evmAddressString,
			"operator_address", operatorAddr,
		)
		// if addr is already registered, we return nil to allow for graceful handling of duplicate registration
		// attempts (e.g. from validator restarts)
		return nil
	}

	err = k.OperatorToEVMAddressMap.Set(ctx, operatorAddr, evmAddrType)
	if err != nil {
		k.Logger(ctx).Info("Error setting EVM address by operator", "error", err)
		return err
	}

	// mark the EVM address as registered
	err = k.EVMAddressRegisteredMap.Set(ctx, evmAddressString, types.EVMAddressRegistered{Registered: true})
	if err != nil {
		k.Logger(ctx).Info("Error marking EVM address as registered", "error", err)
		return err
	}

	// emit event
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.ModuleName,
			sdk.NewAttribute("operator_address", operatorAddr),
			sdk.NewAttribute("evm_address", evmAddressString),
		),
	)
	return nil
}

func (k Keeper) SetBridgeValsetSignature(ctx context.Context, operatorAddress string, timestamp uint64, signature string) error {
	// get the bridge valset signatures array by timestamp
	valsetSigs, err := k.BridgeValsetSignaturesMap.Get(ctx, timestamp)
	if err != nil {
		k.Logger(ctx).Info("Error getting bridge valset signatures", "error", err)
		return err
	}
	// get the evm address associated with the operator address
	ethAddress, err := k.OperatorToEVMAddressMap.Get(ctx, operatorAddress)
	if err != nil {
		k.Logger(ctx).Info("Error getting EVM address from operator address", "error", err)
		return err
	}
	// get the valset index by timestamp
	valsetIdx, err := k.ValsetTimestampToIdxMap.Get(ctx, timestamp)
	if err != nil {
		k.Logger(ctx).Info("Error getting valset index by timestamp", "error", err)
		return err
	}
	if valsetIdx.Index == 0 {
		// first valset, no sigs needed
		return nil
	}
	previousIndex := valsetIdx.Index - 1
	previousValsetTimestamp, err := k.ValidatorCheckpointIdxMap.Get(ctx, previousIndex)
	if err != nil {
		k.Logger(ctx).Info("Error getting previous valset timestamp", "error", err)
		return err
	}
	previousValset, err := k.BridgeValsetByTimestampMap.Get(ctx, previousValsetTimestamp.Timestamp)
	if err != nil {
		k.Logger(ctx).Info("Error getting previous valset", "error", err)
		return err
	}
	// decode the signature hex
	signatureBytes, err := hex.DecodeString(signature)
	if err != nil {
		k.Logger(ctx).Info("Error decoding signature hex", "error", err)
		return err
	}
	// set the signature in the valset signatures array by finding the index of the operator address
	for i, val := range previousValset.BridgeValidatorSet {
		if bytes.Equal(val.EthereumAddress, ethAddress.EVMAddress) {
			valsetSigs.SetSignature(i, signatureBytes)
		}
	}
	// set the valset signatures array by timestamp
	err = k.BridgeValsetSignaturesMap.Set(ctx, timestamp, valsetSigs)
	if err != nil {
		k.Logger(ctx).Info("Error setting bridge valset signatures", "error", err)
		return err
	}
	return nil
}

func (k Keeper) SetOracleAttestation(ctx context.Context, operatorAddress string, snapshot, sig []byte) error {
	// get the evm address associated with the operator address
	ethAddress, err := k.OperatorToEVMAddressMap.Get(ctx, operatorAddress)
	if err != nil {
		k.Logger(ctx).Info("Error getting EVM address from operator address", "error", err)
		return err
	}
	// get the last saved bridge validator set
	lastSavedBridgeValidators, err := k.BridgeValset.Get(ctx)
	if err != nil {
		k.Logger(ctx).Info("Error getting last saved bridge validators", "error", err)
		return err
	}
	// set the signature in the oracle attestation map by finding the index of the operator address
	for i, val := range lastSavedBridgeValidators.BridgeValidatorSet {
		if bytes.Equal(val.EthereumAddress, ethAddress.EVMAddress) {
			snapshotToSigsMap, err := k.SnapshotToAttestationsMap.Get(ctx, snapshot)
			if err != nil {
				k.Logger(ctx).Info("Error getting snapshot to attestations map", "error", err)
				return err
			}
			snapshotToSigsMap.SetAttestation(i, sig)
			err = k.SnapshotToAttestationsMap.Set(ctx, snapshot, snapshotToSigsMap)
			if err != nil {
				k.Logger(ctx).Info("Error setting snapshot to attestations map", "error", err)
				return err
			}
		}
	}
	return nil
}

func (k Keeper) GetEVMAddressByOperator(ctx context.Context, operatorAddress string) ([]byte, error) {
	ethAddress, err := k.OperatorToEVMAddressMap.Get(ctx, operatorAddress)
	if err != nil {
		k.Logger(ctx).Info("Error getting EVM address from operator address", "error", err)
		return nil, err
	}
	return ethAddress.EVMAddress, nil
}

func (k Keeper) GetBridgeValsetByTimestamp(ctx context.Context, timestamp uint64) (*types.BridgeValidatorSet, error) {
	bridgeValset, err := k.BridgeValsetByTimestampMap.Get(ctx, timestamp)
	if err != nil {
		k.Logger(ctx).Info("Error getting bridge valset by timestamp", "error", err)
		return nil, err
	}
	return &bridgeValset, nil
}

func (k Keeper) GetLatestCheckpointIndex(ctx context.Context) (uint64, error) {
	checkpointIdx, err := k.LatestCheckpointIdx.Get(ctx)
	if err != nil {
		k.Logger(ctx).Info("Error getting latest checkpoint index", "error", err)
		return 0, err
	}
	return checkpointIdx.Index, nil
}

func (k Keeper) GetValidatorDidSignCheckpoint(ctx context.Context, operatorAddr string, checkpointTimestamp uint64) (didSign bool, prevValsetIndex int64, err error) {
	// get the valset index by timestamp
	valsetIdx, err := k.ValsetTimestampToIdxMap.Get(ctx, checkpointTimestamp)
	if err != nil {
		k.Logger(ctx).Info("Error getting valset index by timestamp", "error", err)
		return false, -1, err
	}
	if valsetIdx.Index == 0 {
		// first valset, no sigs needed
		return false, -1, nil
	}
	// get previous valset
	previousIndex := valsetIdx.Index - 1
	previousValsetTimestamp, err := k.ValidatorCheckpointIdxMap.Get(ctx, previousIndex)
	if err != nil {
		k.Logger(ctx).Info("Error getting previous valset timestamp", "error", err)
		return false, -1, err
	}
	previousValset, err := k.BridgeValsetByTimestampMap.Get(ctx, previousValsetTimestamp.Timestamp)
	if err != nil {
		k.Logger(ctx).Info("Error getting previous valset", "error", err)
		return false, -1, err
	}
	// get the evm address associated with the operator address
	ethAddress, err := k.OperatorToEVMAddressMap.Get(ctx, operatorAddr)
	if err != nil {
		k.Logger(ctx).Info("Error getting EVM address from operator address", "error", err)
		return false, -1, err
	}
	// get the valset signatures array by timestamp
	valsetSigs, err := k.BridgeValsetSignaturesMap.Get(ctx, checkpointTimestamp)
	if err != nil {
		k.Logger(ctx).Info("Error getting bridge valset signatures", "error", err)
		return false, -1, err
	}
	// get the index of the evm address in the previous valset
	for i, val := range previousValset.BridgeValidatorSet {
		if bytes.Equal(val.EthereumAddress, ethAddress.EVMAddress) {
			// check if the signature exists
			if len(valsetSigs.Signatures[i]) != 0 {
				return true, int64(i), nil
			} else {
				return false, int64(i), nil
			}
		}
	}
	return false, -1, nil
}

func (k Keeper) CreateNewReportSnapshots(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()
	snapshotlimit, err := k.SnapshotLimit.Get(ctx)
	if err != nil {
		k.Logger(ctx).Info("Error getting snapshot limit", "error", err)
		return err
	}
	reports := k.oracleKeeper.GetAggregatedReportsByHeight(ctx, uint64(blockHeight))
	for _, report := range reports {
		queryId := report.QueryId
		timeNow := sdkCtx.BlockTime().Add(time.Second)
		reportTime, err := k.oracleKeeper.GetTimestampBefore(ctx, queryId, timeNow)
		if err != nil {
			return nil
		}
		err = k.CreateSnapshot(ctx, queryId, reportTime, false)
		if err != nil {
			return err
		}
		if snapshotlimit.Limit == 0 {
			break
		}
		snapshotlimit.Limit--
	}
	return nil
}

// Called with each new agg report and with new request for optimistic attestations
func (k Keeper) CreateSnapshot(ctx context.Context, queryId []byte, timestamp time.Time, isExternalRequest bool) error {
	aggReport, err := k.oracleKeeper.GetAggregateByTimestamp(ctx, queryId, uint64(timestamp.UnixMilli()))
	if err != nil {
		k.Logger(ctx).Info("Error getting aggregate report by timestamp", "error", err)
		return err
	}
	// check whether report is flagged as disputed. If so, do not create a snapshot.
	if isExternalRequest && aggReport.Flagged {
		return errors.New("report is flagged as dispute evidence")
	}
	// get the current validator checkpoint
	validatorCheckpoint, err := k.GetValidatorCheckpointFromStorage(ctx)
	if err != nil {
		k.Logger(ctx).Info("Error getting validator checkpoint from storage", "error", err)
		return err
	}

	tsBefore, err := k.oracleKeeper.GetTimestampBefore(ctx, queryId, timestamp)
	if err != nil {
		tsBefore = time.Unix(0, 0)
	}

	tsAfter, err := k.oracleKeeper.GetTimestampAfter(ctx, queryId, timestamp)
	if err != nil {
		tsAfter = time.Unix(0, 0)
	}

	// use current block time for attestationTimestamp
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	attestationTimestamp := sdkCtx.BlockTime()

	lastConsensusTimestamp, err := k.GetLastConsensusTimestamp(ctx, queryId, timestamp, aggReport.AggregatePower, isExternalRequest)
	if err != nil {
		k.Logger(ctx).Info("Error getting last consensus timestamp", "error", err)
		return err
	}

	snapshotBytes, err := k.EncodeOracleAttestationData(
		queryId,
		aggReport.AggregateValue,
		uint64(timestamp.UnixMilli()),
		aggReport.AggregatePower,
		uint64(tsBefore.UnixMilli()),
		uint64(tsAfter.UnixMilli()),
		validatorCheckpoint.Checkpoint,
		uint64(attestationTimestamp.UnixMilli()),
		lastConsensusTimestamp,
	)
	if err != nil {
		k.Logger(ctx).Info("Error encoding oracle attestation data", "error", err)
		return err
	}

	// set snapshot by report
	key := crypto.Keccak256([]byte(hex.EncodeToString(queryId) + fmt.Sprint(timestamp.UnixMilli())))
	// check if map for this key exists, otherwise create a new map
	exists, err := k.AttestSnapshotsByReportMap.Has(ctx, key)
	if err != nil {
		k.Logger(ctx).Info("Error checking if attestation snapshots by report map exists", "error", err)
		return err
	}
	if !exists {
		attestationSnapshots := types.NewAttestationSnapshots()
		err = k.AttestSnapshotsByReportMap.Set(ctx, key, *attestationSnapshots)
		if err != nil {
			k.Logger(ctx).Info("Error setting attestation snapshots by report", "error", err)
			return err
		}
	}
	attestationSnapshots, err := k.AttestSnapshotsByReportMap.Get(ctx, key)
	if err != nil {
		k.Logger(ctx).Info("Error getting attestation snapshots by report", "error", err)
		return err
	}
	// set the snapshot by report
	attestationSnapshots.SetSnapshot(snapshotBytes)
	err = k.AttestSnapshotsByReportMap.Set(ctx, key, attestationSnapshots)
	if err != nil {
		k.Logger(ctx).Info("Error setting attestation snapshots by report", "error", err)
		return err
	}

	// set snapshot to snapshot data map
	snapshotData := types.AttestationSnapshotData{
		ValidatorCheckpoint:    validatorCheckpoint.Checkpoint,
		AttestationTimestamp:   uint64(attestationTimestamp.UnixMilli()),
		PrevReportTimestamp:    uint64(tsBefore.UnixMilli()),
		NextReportTimestamp:    uint64(tsAfter.UnixMilli()),
		QueryId:                queryId,
		Timestamp:              uint64(timestamp.UnixMilli()),
		LastConsensusTimestamp: lastConsensusTimestamp,
	}
	err = k.AttestSnapshotDataMap.Set(ctx, snapshotBytes, snapshotData)
	if err != nil {
		k.Logger(ctx).Info("Error setting attestation snapshot data", "error", err)
		return err
	}

	// initialize snapshot to attestations map
	lastSavedBridgeValidators, err := k.BridgeValset.Get(ctx)
	if err != nil {
		k.Logger(ctx).Info("Error getting last saved bridge validators", "error", err)
		return err
	}
	oracleAttestations := types.NewOracleAttestations(len(lastSavedBridgeValidators.BridgeValidatorSet))
	// set the map
	err = k.SnapshotToAttestationsMap.Set(ctx, snapshotBytes, *oracleAttestations)
	if err != nil {
		k.Logger(ctx).Info("Error setting snapshot to attestations map", "error", err)
		return err
	}

	// add to attestation requests
	blockHeight := uint64(sdkCtx.BlockHeight())
	exists, err = k.AttestRequestsByHeightMap.Has(ctx, blockHeight)
	if err != nil {
		k.Logger(ctx).Info("Error checking if attestation requests by height map exists", "error", err)
		return err
	}
	if !exists {
		attestRequests := types.AttestationRequests{}
		err = k.AttestRequestsByHeightMap.Set(ctx, blockHeight, attestRequests)
		if err != nil {
			k.Logger(ctx).Info("Error setting attestation requests by height", "error", err)
			return err
		}
	}
	attestRequests, err := k.AttestRequestsByHeightMap.Get(ctx, blockHeight)
	if err != nil {
		k.Logger(ctx).Info("Error getting attestation requests by height", "error", err)
		return err
	}
	snapshotLimit, err := k.SnapshotLimit.Get(ctx)
	if err != nil {
		k.Logger(ctx).Info("Error getting snapshot limit", "error", err)
		return err
	}
	if isExternalRequest && len(attestRequests.Requests) > int(snapshotLimit.Limit) {
		return errors.New("too many external requests")
	}
	request := types.AttestationRequest{
		Snapshot: snapshotBytes,
	}
	attestRequests.AddRequest(&request)
	return k.AttestRequestsByHeightMap.Set(ctx, blockHeight, attestRequests)
}

func (k Keeper) GetLastConsensusTimestamp(ctx context.Context, queryId []byte, timestamp time.Time, aggregatePower uint64, isExternalRequest bool) (uint64, error) {
	var pastReportTimestamp time.Time
	// handle new report
	if !isExternalRequest {
		currentValsetTimestamp, err := k.GetCurrentValidatorSetTimestamp(ctx)
		if err != nil {
			return 0, err
		}
		valsetParams, err := k.ValidatorCheckpointParamsMap.Get(ctx, currentValsetTimestamp)
		if err != nil {
			return 0, err
		}
		// handle consensus report
		if aggregatePower >= valsetParams.PowerThreshold {
			return uint64(timestamp.UnixMilli()), nil
			// handle non-consensus report
		} else {
			pastReportTimestamp, err = k.oracleKeeper.GetTimestampBefore(ctx, queryId, timestamp)
			if err != nil {
				if strings.Contains(err.Error(), "no data before timestamp") {
					return 0, nil
				}
				return 0, err
			}
		}
		// handle external request
	} else {
		var err error
		_, pastReportTimestamp, err = k.oracleKeeper.GetCurrentAggregateReport(ctx, queryId)
		if err != nil {
			return 0, err
		}
	}
	snapshotKey := crypto.Keccak256([]byte(hex.EncodeToString(queryId) + fmt.Sprint(pastReportTimestamp.UnixMilli())))
	snapshots, err := k.AttestSnapshotsByReportMap.Get(ctx, snapshotKey)
	if err != nil {
		return 0, err
	}
	if len(snapshots.Snapshots) == 0 {
		return 0, nil
	}
	snapshot := snapshots.Snapshots[0]
	snapshotData, err := k.AttestSnapshotDataMap.Get(ctx, snapshot)
	if err != nil {
		return 0, err
	}
	return snapshotData.LastConsensusTimestamp, nil
}

func (k Keeper) EncodeOracleAttestationData(
	queryId []byte,
	value string,
	timestamp uint64,
	aggregatePower uint64,
	previousTimestamp uint64,
	nextTimestamp uint64,
	valsetCheckpoint []byte,
	attestationTimestamp uint64,
	lastConsensusTimestamp uint64,
) ([]byte, error) {
	// domainSeparator is bytes "tellorCurrentAttestation"
	NEW_REPORT_ATTESTATION_DOMAIN_SEPARATOR := []byte("tellorCurrentAttestation")
	// Convert domain separator to bytes32
	var domainSepBytes32 [32]byte
	copy(domainSepBytes32[:], NEW_REPORT_ATTESTATION_DOMAIN_SEPARATOR)

	// Convert queryId to bytes32
	var queryIdBytes32 [32]byte
	copy(queryIdBytes32[:], queryId)

	// Convert value to bytes
	valueBytes, err := hex.DecodeString(value)
	if err != nil {
		return nil, err
	}

	// Convert timestamp to uint64
	timestampUint64 := new(big.Int)
	timestampUint64.SetUint64(timestamp)

	// Convert aggregatePower to uint64
	aggregatePowerUint64 := new(big.Int)
	aggregatePowerUint64.SetUint64(aggregatePower)

	// Convert previousTimestamp to uint64
	previousTimestampUint64 := new(big.Int)
	previousTimestampUint64.SetUint64(previousTimestamp)

	// Convert nextTimestamp to uint64
	nextTimestampUint64 := new(big.Int)
	nextTimestampUint64.SetUint64(nextTimestamp)

	// Convert valsetCheckpoint to bytes32
	var valsetCheckpointBytes32 [32]byte
	copy(valsetCheckpointBytes32[:], valsetCheckpoint)

	// Convert attestationTimestamp to uint64
	attestationTimestampUint64 := new(big.Int)
	attestationTimestampUint64.SetUint64(attestationTimestamp)

	// Convert lastConsensusTimestamp to uint64
	lastConsensusTimestampUint64 := new(big.Int)
	lastConsensusTimestampUint64.SetUint64(lastConsensusTimestamp)

	// Prepare Encoding
	Bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return nil, err
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	BytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}

	arguments := abi.Arguments{
		{Type: Bytes32Type},
		{Type: Bytes32Type},
		{Type: BytesType},
		{Type: Uint256Type},
		{Type: Uint256Type},
		{Type: Uint256Type},
		{Type: Uint256Type},
		{Type: Bytes32Type},
		{Type: Uint256Type},
		{Type: Uint256Type},
	}

	// Encode the data
	encodedData, err := arguments.Pack(
		domainSepBytes32,
		queryIdBytes32,
		valueBytes,
		timestampUint64,
		aggregatePowerUint64,
		previousTimestampUint64,
		nextTimestampUint64,
		valsetCheckpointBytes32,
		attestationTimestampUint64,
		lastConsensusTimestampUint64,
	)
	if err != nil {
		return nil, err
	}

	oracleAttestationHash := crypto.Keccak256(encodedData)
	return oracleAttestationHash, nil
}

func (k Keeper) GetAttestationRequestsByHeight(ctx context.Context, height uint64) (*types.AttestationRequests, error) {
	attestRequests, err := k.AttestRequestsByHeightMap.Get(ctx, height)
	if err != nil {
		return nil, err
	}
	return &attestRequests, nil
}

func (k Keeper) GetValidatorCheckpointParamsFromStorage(ctx context.Context, timestamp uint64) (types.ValidatorCheckpointParams, error) {
	return k.ValidatorCheckpointParamsMap.Get(ctx, timestamp)
}

func (k Keeper) GetValidatorSetTimestampBefore(ctx context.Context, targetTimestamp uint64) (uint64, error) {
	var mostRecentTimestamp uint64
	rng := new(collections.Range[uint64]).EndExclusive(targetTimestamp).Descending()
	err := k.ValidatorCheckpointParamsMap.Walk(ctx, rng, func(key uint64, value types.ValidatorCheckpointParams) (bool, error) {
		mostRecentTimestamp = key
		return true, nil // Stop after finding the first (most recent) timestamp
	})
	if err != nil {
		return 0, fmt.Errorf("error walking through ValidatorCheckpointParamsMap: %w", err)
	}
	if mostRecentTimestamp == 0 {
		return 0, fmt.Errorf("no validator set timestamp found before %d", targetTimestamp)
	}
	return mostRecentTimestamp, nil
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}
