package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/registry/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestQueryGetDataSpecSpec(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)

	// check Spec() return for unregistered data spec
	unwrappedCtx := sdk.UnwrapSDKContext(ctx)
	specReturn, err := k.GetSpec(unwrappedCtx, "queryType1")
	require.Error(t, err)
	require.Equal(t, specReturn, types.DataSpec{})

	// register a spec and check Spec() returns correct bytes
	spec1 := types.DataSpec{DocumentHash: "hash1", ResponseValueType: "uint256", AggregationMethod: "weighted-median", Registrar: "creator1"}
	specInput := &types.MsgRegisterSpec{
		Registrar: spec1.Registrar,
		QueryType: "queryType1",
		Spec:      spec1,
	}
	registerSpecResult, err := ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.Equal(t, registerSpecResult, &types.MsgRegisterSpecResponse{})

	specReturn, err = k.GetSpec(unwrappedCtx, "queryType1")
	fmt.Println("specReturn2: ", specReturn)
	require.Nil(t, err)
	require.Equal(t, specReturn, spec1)
}

func TestSetDataSpec(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)

	// Define test data
	queryType := "queryType1"
	dataSpec := types.DataSpec{
		DocumentHash:      "hash1",
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
		Registrar:         "creator1",
	}

	// Call the function
	err := k.SetDataSpec(sdk.UnwrapSDKContext(ctx), queryType, dataSpec)
	require.NoError(t, err)

	// Retrieve the data spec
	unwrappedCtx := sdk.UnwrapSDKContext(ctx)
	specReturn, err := k.GetSpec(unwrappedCtx, queryType)
	require.NoError(t, err)
	require.Equal(t, specReturn, dataSpec)

	// test cases where buffer window exceeds max allowed value
	testCases := []struct {
		name        string
		queryType   string
		dataspec    types.DataSpec
		expectError bool
	}{
		{
			name:      "dataspec buffer window < max buffer window, no err",
			queryType: "SPOTPRICE",
			dataspec: types.DataSpec{
				DocumentHash:      "hash1",
				ResponseValueType: "uint256",
				AggregationMethod: "weighted-median",
				Registrar:         "creator1",
				ReportBlockWindow: 500_000, // 20 days
			},
			expectError: false,
		},
		{
			name:      "dataspec buffer window > max buffer window, err",
			queryType: "SPOTPRICE",
			dataspec: types.DataSpec{
				DocumentHash:      "hash2",
				ResponseValueType: "uint256",
				AggregationMethod: "weighted-median",
				Registrar:         "creator1",
				ReportBlockWindow: 1_000_000, // 22 days
			},
			expectError: true,
		},
		{
			name:      "dataspec buffer window = max buffer window, no err",
			queryType: "SPOTPRICE",
			dataspec: types.DataSpec{
				DocumentHash:      "hash3",
				ResponseValueType: "uint256",
				AggregationMethod: "weighted-median",
				Registrar:         "creator1",
				ReportBlockWindow: 700_000, // 21 days
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := k.SetDataSpec(sdk.UnwrapSDKContext(ctx), tc.queryType, tc.dataspec)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				specReturn, err := k.GetSpec(sdk.UnwrapSDKContext(ctx), tc.queryType)
				require.NoError(t, err)
				require.Equal(t, specReturn, tc.dataspec)
			}
		})
	}
}

func TestHasDataSpec(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)

	// Define test data
	queryType := "queryType1"
	dataSpec := types.DataSpec{
		DocumentHash:      "hash1",
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
		Registrar:         "creator1",
	}

	// Call the function
	err := k.SetDataSpec(sdk.UnwrapSDKContext(ctx), queryType, dataSpec)
	require.NoError(t, err)

	// Retrieve the data spec
	unwrappedCtx := sdk.UnwrapSDKContext(ctx)
	specReturn, err := k.HasSpec(unwrappedCtx, queryType)
	require.NoError(t, err)
	require.Equal(t, specReturn, true)
}

func TestMaxReportBufferWindow(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)
	params, err := k.GetParams(sdk.UnwrapSDKContext(ctx))
	require.NoError(t, err)
	require.Equal(t, params.MaxReportBufferWindow, uint64(700_000)) // default is 21 days

	// Test cases
	testCases := []struct {
		name           string
		bufferWindow   time.Duration
		expectedWindow uint64
		expectError    bool
		setup          func()
	}{
		{
			name:           "Set and get 1 hr buffer window",
			bufferWindow:   time.Duration(3600) * time.Second,
			expectedWindow: uint64(2000),
			expectError:    false,
			setup: func() {
				err := k.SetParams(sdk.UnwrapSDKContext(ctx), types.Params{MaxReportBufferWindow: uint64(2000)})
				require.NoError(t, err)
			},
		},
		{
			name:           "Set and get zero buffer window",
			bufferWindow:   time.Duration(0) * time.Second,
			expectedWindow: uint64(0),
			expectError:    false,
			setup: func() {
				err := k.SetParams(sdk.UnwrapSDKContext(ctx), types.Params{MaxReportBufferWindow: uint64(0)})
				require.NoError(t, err)
			},
		},
		{
			name:           "Update existing buffer window to 2 hrs",
			bufferWindow:   time.Duration(7200) * time.Second,
			expectedWindow: uint64(4000),
			expectError:    false,
			setup: func() {
				err := k.SetParams(sdk.UnwrapSDKContext(ctx), types.Params{MaxReportBufferWindow: uint64(4000)})
				require.NoError(t, err)
			},
		},
		{
			name:           "Set to 21 days",
			bufferWindow:   time.Duration(21) * 24 * time.Hour,
			expectedWindow: uint64(900_000),
			expectError:    false,
			setup: func() {
				err := k.SetParams(sdk.UnwrapSDKContext(ctx), types.Params{MaxReportBufferWindow: uint64(900_000)})
				require.NoError(t, err)
			},
		},
		{
			name:           "Set to 63 days",
			bufferWindow:   time.Duration(63) * 24 * time.Hour,
			expectedWindow: uint64(2_000_000),
			expectError:    false,
			setup: func() {
				err := k.SetParams(sdk.UnwrapSDKContext(ctx), types.Params{MaxReportBufferWindow: uint64(2_000_000)})
				require.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set the buffer window
			if tc.setup != nil {
				tc.setup()
			}
			// Get the buffer window
			paramsWindow, err := k.MaxReportBufferWindow(sdk.UnwrapSDKContext(ctx))

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedWindow, paramsWindow)
			}
		})
	}
}
