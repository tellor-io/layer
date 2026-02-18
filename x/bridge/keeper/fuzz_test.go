package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
)

func FuzzTryRecoverAddressWithBothIDs(f *testing.F) {
	// Valid 64-byte sig + 32-byte hash
	f.Add(make([]byte, 64), make([]byte, 32))
	// Wrong lengths
	f.Add([]byte{}, []byte{})
	f.Add(make([]byte, 63), make([]byte, 32))
	f.Add(make([]byte, 65), make([]byte, 32))
	f.Add(make([]byte, 64), []byte{})
	f.Add(make([]byte, 64), make([]byte, 31))

	k, _, _, _, _, _, _, _ := keepertest.BridgeKeeper(f)

	f.Fuzz(func(t *testing.T, sig, msgHash []byte) {
		_, _ = k.TryRecoverAddressWithBothIDs(sig, msgHash)
	})
}

func FuzzEncodeOracleAttestationData(f *testing.F) {
	f.Add(
		make([]byte, 32), // queryId
		"deadbeef",       // value
		uint64(1000),     // timestamp
		uint64(100),      // aggregatePower
		uint64(999),      // previousTimestamp
		uint64(1001),     // nextTimestamp
		make([]byte, 32), // valsetCheckpoint
		uint64(1000),     // attestationTimestamp
		uint64(900),      // lastConsensusTimestamp
	)
	f.Add(
		[]byte{},  // empty queryId
		"",        // empty value
		uint64(0), // zero timestamp
		uint64(0),
		uint64(0),
		uint64(0),
		[]byte{},
		uint64(0),
		uint64(0),
	)
	f.Add(
		make([]byte, 32),
		"0xdeadbeef",
		^uint64(0), // max uint64
		^uint64(0),
		^uint64(0),
		^uint64(0),
		make([]byte, 32),
		^uint64(0),
		^uint64(0),
	)
	f.Add(
		make([]byte, 32),
		"zzzz", // invalid hex
		uint64(1),
		uint64(1),
		uint64(1),
		uint64(1),
		make([]byte, 32),
		uint64(1),
		uint64(1),
	)

	f.Fuzz(func(t *testing.T, queryId []byte, value string, timestamp, aggregatePower, previousTimestamp, nextTimestamp uint64, valsetCheckpoint []byte, attestationTimestamp, lastConsensusTimestamp uint64) {
		k, _, _, _, _, _, _, _ := keepertest.BridgeKeeper(t)
		_, _ = k.EncodeOracleAttestationData(queryId, value, timestamp, aggregatePower, previousTimestamp, nextTimestamp, valsetCheckpoint, attestationTimestamp, lastConsensusTimestamp)
	})
}

func FuzzGetDepositQueryId(f *testing.F) {
	f.Add(uint64(0))
	f.Add(uint64(1))
	f.Add(uint64(1000))
	f.Add(^uint64(0)) // max

	k, _, _, _, _, _, _, _ := keepertest.BridgeKeeper(f)

	f.Fuzz(func(t *testing.T, depositId uint64) {
		_, _ = k.GetDepositQueryId(depositId)
	})
}

func FuzzEncodeValsetCheckpoint(f *testing.F) {
	f.Add(uint64(100), uint64(1000), make([]byte, 32))
	f.Add(uint64(0), uint64(0), []byte{})
	f.Add(^uint64(0), ^uint64(0), make([]byte, 32))
	f.Add(uint64(1), uint64(1), make([]byte, 64)) // oversized hash

	k, _, _, _, _, _, _, ctx := keepertest.BridgeKeeper(f)
	// Set domain separator needed by EncodeValsetCheckpoint
	require.NoError(f, k.ValsetCheckpointDomainSeparator.Set(ctx, make([]byte, 32)))

	f.Fuzz(func(t *testing.T, powerThreshold, validatorTimestamp uint64, validatorSetHash []byte) {
		_, _ = k.EncodeValsetCheckpoint(ctx, powerThreshold, validatorTimestamp, validatorSetHash)
	})
}
