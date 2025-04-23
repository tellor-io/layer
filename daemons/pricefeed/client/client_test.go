package client

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appflags "github.com/tellor-io/layer/app/flags"
	pricefeed_constants "github.com/tellor-io/layer/daemons/constants"
	daemonflags "github.com/tellor-io/layer/daemons/flags"
	"github.com/tellor-io/layer/daemons/mocks"
	"github.com/tellor-io/layer/daemons/pricefeed/client/price_fetcher"
	handler "github.com/tellor-io/layer/daemons/pricefeed/client/queryhandler"
	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
	pricefeedtypes "github.com/tellor-io/layer/daemons/pricefeed/types"
	daemonserver "github.com/tellor-io/layer/daemons/server"
	servertypes "github.com/tellor-io/layer/daemons/server/types/daemons"
	pricefeed_types "github.com/tellor-io/layer/daemons/server/types/pricefeed"
	"github.com/tellor-io/layer/daemons/testutil/appoptions"
	"github.com/tellor-io/layer/daemons/testutil/client"
	"github.com/tellor-io/layer/daemons/testutil/constants"
	daemontestutils "github.com/tellor-io/layer/daemons/testutil/daemons"
	grpc_util "github.com/tellor-io/layer/daemons/testutil/grpc"
	daemontypes "github.com/tellor-io/layer/daemons/types"
	"google.golang.org/grpc"

	"cosmossdk.io/log"
)

var subTaskRunnerImpl = SubTaskRunnerImpl{}

// FakeSubTaskRunner acts as a dummy struct replacing `SubTaskRunner` that simply advances the
// counter for each task in a threadsafe manner and allows awaiting go-routine completion. This
// struct should only be used for testing.
type FakeSubTaskRunner struct {
	sync.WaitGroup
	sync.Mutex
	UpdaterCallCount       int
	EncoderCallCount       int
	FetcherCallCount       int
	MarketUpdaterCallCount int
}

// StartPriceUpdater replaces `client.StartPriceUpdater` and advances `UpdaterCallCount` by one.
func (f *FakeSubTaskRunner) StartPriceUpdater(
	c *Client,
	ctx context.Context,
	ticker *time.Ticker,
	stop <-chan bool,
	exchangeToMarketPrices types.ExchangeToMarketPrices,
	priceFeedServiceClient servertypes.PriceFeedServiceClient,
	logger log.Logger,
) {
	// No need to lock/unlock since there is only one updater running and no risk of race-condition.
	f.UpdaterCallCount += 1
}

// StartPriceEncoder replaces `client.StartPriceEncoder`, marks the embedded waitgroup done and
// advances `EncoderCallCount` by one. This function will be called from a go-routine and is
// threadsafe.
func (f *FakeSubTaskRunner) StartPriceEncoder(
	exchangeId types.ExchangeId,
	configs types.PricefeedMutableMarketConfigs,
	exchangeToMarketPrices types.ExchangeToMarketPrices,
	logger log.Logger,
	bCh <-chan *price_fetcher.PriceFetcherSubtaskResponse,
) {
	f.Lock()
	defer f.Unlock()

	f.EncoderCallCount += 1
	f.Done()
}

// StartPriceFetcher replaces `client.StartPriceFetcher`, marks the embedded waitgroup done and
// advances `FetcherCallCount` by one. This function will be called from a go-routine and is
// threadsafe.
func (f *FakeSubTaskRunner) StartPriceFetcher(
	ticker *time.Ticker,
	stop <-chan bool,
	configs types.PricefeedMutableMarketConfigs,
	exchangeQueryConfig types.ExchangeQueryConfig,
	exchangeDetails types.ExchangeQueryDetails,
	queryHandler handler.ExchangeQueryHandler,
	logger log.Logger,
	bCh chan<- *price_fetcher.PriceFetcherSubtaskResponse,
) {
	f.Lock()
	defer f.Unlock()

	f.FetcherCallCount += 1
	f.Done()
}

const (
	maxBufferedChannelLength     = 2
	connectionFailsErrorMsg      = "Failed to create connection"
	closeConnectionFailsErrorMsg = "Failed to close connection"
	fiveKilobytes                = 5 * 1024
)

func TestFixedBufferSize(t *testing.T) {
	require.Equal(t, fiveKilobytes, pricefeed_constants.FixedBufferSize)
}

// TODO: Add market param tester config.
func TestStart_InvalidConfig(t *testing.T) {
	tests := map[string]struct {
		// parameters
		mockGrpcClient              *mocks.GrpcClient
		marketParam                 []types.MarketParam
		initialMarketConfig         map[types.MarketId]*types.MutableMarketConfig
		initialExchangeMarketConfig map[types.ExchangeId]*types.MutableExchangeMarketConfig
		exchangeIdToQueryConfig     map[types.ExchangeId]*types.ExchangeQueryConfig
		exchangeIdToExchangeDetails map[types.ExchangeId]types.ExchangeQueryDetails

		// expectations
		expectedError             error
		expectGrpcConnection      bool
		expectCloseTcpConnection  bool
		expectCloseGrpcConnection bool
		// This should equal the length of the `exchangeIdToQueryConfig` passed into
		// `client.Start`.
		expectedNumExchangeTasks int
	}{
		"Invalid: Tcp Connection Fails": {
			mockGrpcClient: grpc_util.GenerateMockGrpcClientWithOptionalTcpConnectionErrors(
				errors.New(connectionFailsErrorMsg),
				nil,
				false,
			),
			expectedError: errors.New(connectionFailsErrorMsg),
		},
		"Invalid: Grpc Connection Fails": {
			mockGrpcClient: grpc_util.GenerateMockGrpcClientWithOptionalGrpcConnectionErrors(
				errors.New(connectionFailsErrorMsg),
				nil,
				false,
			),
			expectedError:            errors.New(connectionFailsErrorMsg),
			expectGrpcConnection:     true,
			expectCloseTcpConnection: true,
		},
		// "Valid: 2 exchanges": {
		// 	mockGrpcClient:              grpc_util.GenerateMockGrpcClientWithOptionalGrpcConnectionErrors(nil, nil, true),
		// 	exchangeIdToQueryConfig:     constants.TestExchangeQueryConfigs,
		// 	exchangeIdToExchangeDetails: constants.TestExchangeIdToExchangeQueryDetails,
		// 	expectGrpcConnection:        true,
		// 	expectCloseTcpConnection:    true,
		// 	expectCloseGrpcConnection:   true,
		// 	expectedNumExchangeTasks:    testExchangeQueryConfigLength,
		// },
		// "Invalid: empty exchange query config": {
		// 	mockGrpcClient:            grpc_util.GenerateMockGrpcClientWithOptionalGrpcConnectionErrors(nil, nil, true),
		// 	exchangeIdToQueryConfig:   map[types.ExchangeId]*types.ExchangeQueryConfig{},
		// 	expectedError:             errors.New("exchangeIds must not be empty"),
		// 	expectGrpcConnection:      true,
		// 	expectCloseTcpConnection:  true,
		// 	expectCloseGrpcConnection: true,
		// },
		// "Invalid: missing exchange query details": {
		// 	mockGrpcClient: grpc_util.GenerateMockGrpcClientWithOptionalGrpcConnectionErrors(nil, nil, true),
		// 	exchangeIdToQueryConfig: map[string]*types.ExchangeQueryConfig{
		// 		validExchangeId: constants.TestExchangeQueryConfigs[validExchangeId],
		// 	},
		// 	expectedError:             fmt.Errorf("no exchange details exists for exchangeId: %v", validExchangeId),
		// 	expectGrpcConnection:      true,
		// 	expectCloseTcpConnection:  true,
		// 	expectCloseGrpcConnection: true,
		// },
		// "Invalid: tcp close connection fails with good inputs": {
		// 	mockGrpcClient: grpc_util.GenerateMockGrpcClientWithOptionalTcpConnectionErrors(
		// 		nil,
		// 		closeConnectionFailsError,
		// 		true,
		// 	),
		// 	exchangeIdToQueryConfig:     constants.TestExchangeQueryConfigs,
		// 	exchangeIdToExchangeDetails: constants.TestExchangeIdToExchangeQueryDetails,
		// 	expectedError:               closeConnectionFailsError,
		// 	expectGrpcConnection:        true,
		// 	expectCloseTcpConnection:    true,
		// 	expectCloseGrpcConnection:   true,
		// 	expectedNumExchangeTasks:    testExchangeQueryConfigLength,
		// },
		// "Invalid: grpc close connection fails with good inputs": {
		// 	mockGrpcClient: grpc_util.GenerateMockGrpcClientWithOptionalGrpcConnectionErrors(
		// 		nil,
		// 		closeConnectionFailsError,
		// 		true,
		// 	),
		// 	exchangeIdToQueryConfig:     constants.TestExchangeQueryConfigs,
		// 	exchangeIdToExchangeDetails: constants.TestExchangeIdToExchangeQueryDetails,
		// 	expectedError:               closeConnectionFailsError,
		// 	expectGrpcConnection:        true,
		// 	expectCloseTcpConnection:    true,
		// 	expectCloseGrpcConnection:   true,
		// 	expectedNumExchangeTasks:    testExchangeQueryConfigLength,
		// },
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			faketaskRunner := FakeSubTaskRunner{
				UpdaterCallCount:       0,
				EncoderCallCount:       0,
				FetcherCallCount:       0,
				MarketUpdaterCallCount: 0,
			}

			// Wait for each encoder and fetcher call to complete.
			faketaskRunner.WaitGroup.Add(tc.expectedNumExchangeTasks * 2)

			// Run Start.
			client := newClient(log.NewNopLogger())
			err := client.start(
				grpc_util.Ctx,
				daemonflags.GetDefaultDaemonFlags(),
				"localhost:9090",
				tc.mockGrpcClient,
				tc.marketParam,
				tc.exchangeIdToQueryConfig,
				tc.exchangeIdToExchangeDetails,
				&faketaskRunner,
			)

			// Expect daemon is not healthy on startup. Daemon becomes healthy after the first successful market
			// update.
			require.ErrorContains(
				t,
				client.HealthCheck(),
				"no successful update has occurred",
			)

			if tc.expectedError == nil {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.expectedError.Error())
			}

			// Wait for encoder and fetcher go-routines to complete and then verify each subtask was
			// called the expected amount.
			faketaskRunner.Wait()
			require.Equal(t, tc.expectedNumExchangeTasks, faketaskRunner.EncoderCallCount)
			require.Equal(t, tc.expectedNumExchangeTasks, faketaskRunner.FetcherCallCount)
			if tc.expectedNumExchangeTasks > 0 {
				require.Equal(t, 1, faketaskRunner.UpdaterCallCount)
			} else {
				require.Equal(t, 0, faketaskRunner.UpdaterCallCount)
			}

			tc.mockGrpcClient.AssertCalled(t, "NewTcpConnection", grpc_util.Ctx, grpc_util.TcpEndpoint)
			if tc.expectGrpcConnection {
				tc.mockGrpcClient.AssertCalled(t, "NewGrpcConnection", grpc_util.Ctx, grpc_util.SocketPath)
			} else {
				tc.mockGrpcClient.AssertNotCalled(t, "NewGrpcConnection", grpc_util.Ctx, grpc_util.SocketPath)
			}

			if tc.expectCloseGrpcConnection {
				tc.mockGrpcClient.AssertCalled(t, "CloseConnection", grpc_util.GrpcConn)
			} else {
				tc.mockGrpcClient.AssertNotCalled(t, "CloseConnection", grpc_util.GrpcConn)
			}

			if tc.expectCloseTcpConnection {
				tc.mockGrpcClient.AssertCalled(t, "CloseConnection", grpc_util.TcpConn)
			} else {
				tc.mockGrpcClient.AssertNotCalled(t, "CloseConnection", grpc_util.TcpConn)
			}
		})
	}
}

// TestStop tests that the Stop interface works as expected. It's difficult to ensure that each go-routine
// is stopped, but this test ensures that the Stop executes successfully with no hangs.
func TestStop(t *testing.T) {
	// Setup daemon and grpc servers.
	daemonFlags := daemonflags.GetDefaultDaemonFlags()
	appFlags := appflags.GetFlagValuesFromOptions(appoptions.GetDefaultTestAppOptions("", nil))

	// Configure and run daemon server.
	daemonServer := daemonserver.NewServer(
		log.NewNopLogger(),
		grpc.NewServer(),
		&daemontypes.FileHandlerImpl{},
		daemonFlags.Shared.SocketAddress,
	)
	daemonServer.WithPriceFeedMarketToExchangePrices(
		pricefeed_types.NewMarketToExchangePrices(5 * time.Second),
	)

	defer daemonServer.Stop()
	go daemonServer.Start()

	// Create a gRPC server running on the default port and attach the mock prices query response.
	grpcServer := grpc.NewServer()
	// pricetypes.RegisterQueryServer(grpcServer, &pricesQueryServer)

	// Start gRPC server with cleanup.
	defer grpcServer.Stop()
	go func() {
		ls, err := net.Listen("tcp", appFlags.GrpcAddress)
		require.NoError(t, err)
		err = grpcServer.Serve(ls)
		require.NoError(t, err)
	}()

	client := StartNewClient(
		grpc_util.Ctx,
		daemonFlags,
		appFlags.GrpcAddress,
		log.NewNopLogger(),
		&daemontypes.GrpcClientImpl{},
		[]types.MarketParam{},
		constants.TestExchangeQueryConfigs,
		constants.TestExchangeIdToExchangeQueryDetails,
		&SubTaskRunnerImpl{},
	)

	// Stop the daemon.
	client.Stop()
}

func TestPriceEncoder_NoWrites(t *testing.T) {
	etmp, bChMap := generateBufferedChannelAndExchangeToMarketPrices(t, constants.Exchange1Exchange2Array)

	runPriceEncoderSequentially(
		t,
		constants.ExchangeId1,
		genMockPricefeedMutableMarketConfigsForExchange(constants.ExchangeId1),
		etmp,
		bChMap[constants.ExchangeId1],
		[]*types.MarketPriceTimestamp{},
	)

	require.Empty(t, etmp.ExchangeMarketPrices[constants.ExchangeId1].MarketToPriceTimestamp)
	require.Empty(t, etmp.ExchangeMarketPrices[constants.ExchangeId2].MarketToPriceTimestamp)
	require.Empty(t, bChMap[constants.ExchangeId1])
	require.Empty(t, bChMap[constants.ExchangeId2])
}

func TestPriceEncoder_DoNotWriteError(t *testing.T) {
	etmp, bChMap := generateBufferedChannelAndExchangeToMarketPrices(t, constants.Exchange1Exchange2Array)

	bCh := bChMap[constants.ExchangeId1]
	bCh <- &price_fetcher.PriceFetcherSubtaskResponse{
		Price: nil,
		Err:   errors.New("Failed to query"),
	}
	close(bCh)

	configs := genMockPricefeedMutableMarketConfigsForExchange(constants.ExchangeId1)

	subTaskRunnerImpl.StartPriceEncoder(constants.ExchangeId1, configs, etmp, log.NewNopLogger(), bCh)

	require.Empty(t, etmp.ExchangeMarketPrices[constants.ExchangeId1].MarketToPriceTimestamp)
	require.Empty(t, etmp.ExchangeMarketPrices[constants.ExchangeId2].MarketToPriceTimestamp)
	require.Empty(t, bChMap[constants.ExchangeId1])
	require.Empty(t, bChMap[constants.ExchangeId2])
}

func TestPriceEncoder_WriteToOneMarket(t *testing.T) {
	etmp, bChMap := generateBufferedChannelAndExchangeToMarketPrices(t, constants.Exchange1Exchange2Array)

	runPriceEncoderSequentially(
		t,
		constants.ExchangeId1,
		genMockPricefeedMutableMarketConfigsForExchange(constants.ExchangeId1),
		etmp,
		bChMap[constants.ExchangeId1],
		[]*types.MarketPriceTimestamp{
			constants.Market9_TimeT_Price1,
		},
	)

	require.Len(t, etmp.ExchangeMarketPrices[constants.ExchangeId1].MarketToPriceTimestamp, 1)
	require.Empty(t, etmp.ExchangeMarketPrices[constants.ExchangeId2].MarketToPriceTimestamp)

	require.Equal(
		t,
		&pricefeedtypes.PriceTimestamp{
			Price:          constants.Price1,
			LastUpdateTime: constants.TimeT,
		},
		etmp.ExchangeMarketPrices[constants.ExchangeId1].MarketToPriceTimestamp[constants.MarketId9],
	)
}

func TestPriceEncoder_WriteToTwoMarkets(t *testing.T) {
	etmp, bChMap := generateBufferedChannelAndExchangeToMarketPrices(t, constants.Exchange1Exchange2Array)

	runPriceEncoderSequentially(
		t,
		constants.ExchangeId1,
		genMockPricefeedMutableMarketConfigsForExchange(constants.ExchangeId1),
		etmp,
		bChMap[constants.ExchangeId1],
		[]*types.MarketPriceTimestamp{
			constants.Market9_TimeT_Price1,
			constants.Market8_TimeTMinusThreshold_Price2,
		},
	)

	require.Len(t, etmp.ExchangeMarketPrices[constants.ExchangeId1].MarketToPriceTimestamp, 2)
	require.Empty(t, etmp.ExchangeMarketPrices[constants.ExchangeId2].MarketToPriceTimestamp)

	require.Equal(
		t,
		&pricefeedtypes.PriceTimestamp{
			Price:          constants.Price1,
			LastUpdateTime: constants.TimeT,
		},
		etmp.ExchangeMarketPrices[constants.ExchangeId1].MarketToPriceTimestamp[constants.MarketId9],
	)
	require.Equal(
		t,
		&pricefeedtypes.PriceTimestamp{
			Price:          constants.Price2,
			LastUpdateTime: constants.TimeTMinusThreshold,
		},
		etmp.ExchangeMarketPrices[constants.ExchangeId1].MarketToPriceTimestamp[constants.MarketId8],
	)
}

func TestPriceEncoder_WriteToOneMarketTwice(t *testing.T) {
	etmp, bChMap := generateBufferedChannelAndExchangeToMarketPrices(t, constants.Exchange1Exchange2Array)

	runPriceEncoderSequentially(
		t,
		constants.ExchangeId1,
		genMockPricefeedMutableMarketConfigsForExchange(constants.ExchangeId1),
		etmp,
		bChMap[constants.ExchangeId1],
		[]*types.MarketPriceTimestamp{
			constants.Market9_TimeTMinusThreshold_Price2,
			constants.Market9_TimeT_Price1,
		},
	)

	require.Len(t, etmp.ExchangeMarketPrices[constants.ExchangeId1].MarketToPriceTimestamp, 1)
	require.Empty(t, etmp.ExchangeMarketPrices[constants.ExchangeId2].MarketToPriceTimestamp)

	require.Equal(
		t,
		&pricefeedtypes.PriceTimestamp{
			Price:          constants.Price1,
			LastUpdateTime: constants.TimeT,
		},
		etmp.ExchangeMarketPrices[constants.ExchangeId1].MarketToPriceTimestamp[constants.MarketId9],
	)
}

func TestPriceEncoder_WriteToTwoExchanges(t *testing.T) {
	etmp, bChMap := generateBufferedChannelAndExchangeToMarketPrices(t, constants.Exchange1Exchange2Array)

	runPriceEncoderSequentially(
		t,
		constants.ExchangeId1,
		genMockPricefeedMutableMarketConfigsForExchange(constants.ExchangeId1),
		etmp,
		bChMap[constants.ExchangeId1],
		[]*types.MarketPriceTimestamp{
			constants.Market9_TimeT_Price1,
		},
	)

	runPriceEncoderSequentially(
		t,
		constants.ExchangeId2,
		genMockPricefeedMutableMarketConfigsForExchange(constants.ExchangeId2),
		etmp,
		bChMap[constants.ExchangeId2],
		[]*types.MarketPriceTimestamp{
			constants.Market8_TimeTMinusThreshold_Price2,
		},
	)

	require.Len(t, etmp.ExchangeMarketPrices[constants.ExchangeId1].MarketToPriceTimestamp, 1)
	require.Len(t, etmp.ExchangeMarketPrices[constants.ExchangeId2].MarketToPriceTimestamp, 1)

	require.Equal(
		t,
		&pricefeedtypes.PriceTimestamp{
			Price:          constants.Price1,
			LastUpdateTime: constants.TimeT,
		},
		etmp.ExchangeMarketPrices[constants.ExchangeId1].MarketToPriceTimestamp[constants.MarketId9],
	)
	require.Equal(
		t,
		&pricefeedtypes.PriceTimestamp{
			Price:          constants.Price2,
			LastUpdateTime: constants.TimeTMinusThreshold,
		},
		etmp.ExchangeMarketPrices[constants.ExchangeId2].MarketToPriceTimestamp[constants.MarketId8],
	)
}

func TestPriceEncoder_WriteToTwoExchangesConcurrentlyWithManyUpdates(t *testing.T) {
	etmp, bChMap := generateBufferedChannelAndExchangeToMarketPrices(t, constants.Exchange1Exchange2Array)

	largeMarketWrite := []*types.MarketPriceTimestamp{
		constants.Market8_TimeTMinusThreshold_Price1,
		constants.Market8_TimeTMinusThreshold_Price2,
		constants.Market8_TimeTMinusThreshold_Price3,
		constants.Market9_TimeTMinusThreshold_Price1,
		constants.Market9_TimeTMinusThreshold_Price2,
		constants.Market9_TimeTMinusThreshold_Price3,
		constants.Market8_TimeT_Price3,
		constants.Market9_TimeT_Price1,
		constants.Market9_TimeT_Price2,
		constants.Market9_TimeT_Price3,
		constants.Market9_TimeTPlusThreshold_Price1,
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		runPriceEncoderConcurrently(
			constants.ExchangeId1,
			genMockPricefeedMutableMarketConfigsForExchange(constants.ExchangeId1),
			etmp,
			bChMap[constants.ExchangeId1],
			largeMarketWrite,
		)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runPriceEncoderConcurrently(
			constants.ExchangeId2,
			genMockPricefeedMutableMarketConfigsForExchange(constants.ExchangeId2),
			etmp,
			bChMap[constants.ExchangeId2],
			largeMarketWrite,
		)
	}()

	wg.Wait()

	require.Len(t, etmp.ExchangeMarketPrices[constants.ExchangeId1].MarketToPriceTimestamp, 2)
	require.Len(t, etmp.ExchangeMarketPrices[constants.ExchangeId2].MarketToPriceTimestamp, 2)

	require.Equal(
		t,
		&pricefeedtypes.PriceTimestamp{
			Price:          constants.Price1,
			LastUpdateTime: constants.TimeTPlusThreshold,
		},
		etmp.ExchangeMarketPrices[constants.ExchangeId1].MarketToPriceTimestamp[constants.MarketId9],
	)
	require.Equal(
		t,
		&pricefeedtypes.PriceTimestamp{
			Price:          constants.Price3,
			LastUpdateTime: constants.TimeT,
		},
		etmp.ExchangeMarketPrices[constants.ExchangeId1].MarketToPriceTimestamp[constants.MarketId8],
	)

	require.Equal(
		t,
		&pricefeedtypes.PriceTimestamp{
			Price:          constants.Price1,
			LastUpdateTime: constants.TimeTPlusThreshold,
		},
		etmp.ExchangeMarketPrices[constants.ExchangeId2].MarketToPriceTimestamp[constants.MarketId9],
	)
	require.Equal(
		t,
		&pricefeedtypes.PriceTimestamp{
			Price:          constants.Price3,
			LastUpdateTime: constants.TimeT,
		},
		etmp.ExchangeMarketPrices[constants.ExchangeId2].MarketToPriceTimestamp[constants.MarketId8],
	)
}

func TestPriceUpdater_Mixed(t *testing.T) {
	tests := map[string]struct {
		// parameters
		exchangeAndMarketPrices []*client.ExchangeIdMarketPriceTimestamp
		priceUpdateError        error

		// expectations
		expectedMarketPriceUpdate []*servertypes.MarketPriceUpdate
	}{
		"Update throws": {
			// Throws error due to mock so that we can simulate fail state.
			exchangeAndMarketPrices: []*client.ExchangeIdMarketPriceTimestamp{
				constants.ExchangeId1_Market9_TimeT_Price1,
			},
			priceUpdateError: errors.New("failed to send price update"),
		},
		"No exchange market prices, does not call `UpdateMarketPrices`": {
			exchangeAndMarketPrices: []*client.ExchangeIdMarketPriceTimestamp{},
			priceUpdateError:        fmt.Errorf("ErrEmptyMarketPriceUpdate"),
		},
		"One market for one exchange": {
			exchangeAndMarketPrices: []*client.ExchangeIdMarketPriceTimestamp{
				constants.ExchangeId1_Market9_TimeT_Price1,
			},
			expectedMarketPriceUpdate: constants.Market9_SingleExchange_AtTimeUpdate,
		},
		"Three markets at timeT": {
			exchangeAndMarketPrices: []*client.ExchangeIdMarketPriceTimestamp{
				constants.ExchangeId1_Market9_TimeT_Price1,
				constants.ExchangeId2_Market9_TimeT_Price2,
				constants.ExchangeId2_Market8_TimeT_Price2,
				constants.ExchangeId3_Market8_TimeT_Price3,
				constants.ExchangeId1_Market7_TimeT_Price1,
				constants.ExchangeId3_Market7_TimeT_Price3,
			},
			expectedMarketPriceUpdate: constants.AtTimeTPriceUpdate,
		},
		"Three markets at mixed time": {
			exchangeAndMarketPrices: []*client.ExchangeIdMarketPriceTimestamp{
				constants.ExchangeId1_Market9_TimeT_Price1,
				constants.ExchangeId2_Market9_TimeT_Price2,
				constants.ExchangeId3_Market9_TimeT_Price3,
				constants.ExchangeId1_Market8_BeforeTimeT_Price3,
				constants.ExchangeId2_Market8_TimeT_Price2,
				constants.ExchangeId3_Market8_TimeT_Price3,
				constants.ExchangeId2_Market7_BeforeTimeT_Price1,
				constants.ExchangeId1_Market7_BeforeTimeT_Price3,
				constants.ExchangeId3_Market7_TimeT_Price3,
			},
			expectedMarketPriceUpdate: constants.MixedTimePriceUpdate,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create `ExchangeIdMarketPriceTimestamp` and populate it with market-price updates.
			etmp, _ := types.NewExchangeToMarketPrices(
				[]types.ExchangeId{
					constants.ExchangeId1,
					constants.ExchangeId2,
					constants.ExchangeId3,
				},
			)
			for _, exchangeAndMarketPrice := range tc.exchangeAndMarketPrices {
				etmp.UpdatePrice(
					exchangeAndMarketPrice.ExchangeId,
					exchangeAndMarketPrice.MarketPriceTimestamp,
				)
			}

			// Create a mock `PriceFeedServiceClient` and run `RunPriceUpdaterTaskLoop`.
			mockPriceFeedClient := generateMockQueryClient()
			mockPriceFeedClient.On("UpdateMarketPrices", grpc_util.Ctx, mock.Anything).
				Return(nil, tc.priceUpdateError)

			err := RunPriceUpdaterTaskLoop(
				grpc_util.Ctx,
				etmp,
				mockPriceFeedClient,
				log.NewNopLogger(),
			)
			require.Equal(
				t,
				tc.priceUpdateError,
				err,
			)

			// We sort the `expectedUpdates` as ordering is not guaranteed.
			// We then verify `UpdateMarketPrices` was called with an update that, when sorted, matches
			// the sorted `expectedUpdates`.
			expectedUpdates := tc.expectedMarketPriceUpdate
			sortMarketPriceUpdateByMarketIdDescending(expectedUpdates)

			if tc.expectedMarketPriceUpdate != nil {
				mockPriceFeedClient.AssertCalled(
					t,
					"UpdateMarketPrices",
					grpc_util.Ctx,
					mock.MatchedBy(func(i interface{}) bool {
						param := i.(*servertypes.UpdateMarketPricesRequest)
						updates := param.MarketPriceUpdates
						sortMarketPriceUpdateByMarketIdDescending(updates)

						for i, update := range updates {
							prices := update.ExchangePrices
							require.ElementsMatch(
								t,
								expectedUpdates[i].ExchangePrices,
								prices,
							)
						}
						return true
					}),
				)
			} else {
				mockPriceFeedClient.AssertNotCalled(t, "UpdateMarketPrices")
			}
		})
	}
}

func TestHealthCheck_Mixed(t *testing.T) {
	tests := map[string]struct {
		updateMarketPricesError error
		expectedError           error
	}{
		"No error - daemon healthy": {
			updateMarketPricesError: nil,
			expectedError:           nil,
		},
		"Error - daemon unhealthy": {
			updateMarketPricesError: fmt.Errorf("failed to update market prices"),
			expectedError: fmt.Errorf(
				"failed to run price updater task loop for price daemon; failed to update market prices",
			),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Setup.
			// Create `ExchangeIdMarketPriceTimestamp` and populate it with market-price updates.
			etmp, err := types.NewExchangeToMarketPrices([]types.ExchangeId{constants.ExchangeId1})
			require.NoError(t, err)
			etmp.UpdatePrice(constants.ExchangeId1, constants.Market9_TimeT_Price1)

			// Create a mock `PriceFeedServiceClient`.
			mockPriceFeedClient := generateMockQueryClient()

			// Mock the `UpdateMarketPrices` call to return an error if specified.
			mockPriceFeedClient.On("UpdateMarketPrices", grpc_util.Ctx, mock.Anything).
				Return(nil, tc.updateMarketPricesError).Once()

			ticker, stop := daemontestutils.SingleTickTickerAndStop()
			client := newClient(log.NewNopLogger())

			// Act.
			// Run the price updater for a single tick. Expect the daemon to toggle health state based on
			// `UpdateMarketPrices` error response.
			subTaskRunnerImpl.StartPriceUpdater(
				client,
				grpc_util.Ctx,
				ticker,
				stop,
				etmp,
				mockPriceFeedClient,
				log.NewNopLogger(),
			)

			// Assert.
			if tc.expectedError == nil {
				require.NoError(t, client.HealthCheck())
			} else {
				require.ErrorContains(t, client.HealthCheck(), tc.expectedError.Error())
			}
		})
	}
}

// // ----------------- Generate Mock Instances ----------------- //

// // generateMockQueryClient generates a mock QueryClient that can be used to support any of the QueryClient
// // interfaces added to the mocks.QueryClient class, including the prices query client and the
// // pricefeed service client.
func generateMockQueryClient() *mocks.QueryClient {
	mockPriceFeedServiceClient := &mocks.QueryClient{}

	return mockPriceFeedServiceClient
}

// ----------------- Helper Functions ----------------- //

func generateBufferedChannelAndExchangeToMarketPrices(
	t *testing.T,
	exchangeIds []types.ExchangeId,
) (
	*types.ExchangeToMarketPricesImpl,
	map[types.ExchangeId]chan *price_fetcher.PriceFetcherSubtaskResponse,
) {
	t.Helper()
	_etmp, err := types.NewExchangeToMarketPrices(exchangeIds)
	etmp := _etmp.(*types.ExchangeToMarketPricesImpl)
	require.NoError(t, err)
	require.NotNil(t, etmp)

	exchangeIdToBufferedChannel := map[types.ExchangeId]chan *price_fetcher.PriceFetcherSubtaskResponse{}
	for _, exchangeId := range exchangeIds {
		bCh := make(chan *price_fetcher.PriceFetcherSubtaskResponse, maxBufferedChannelLength)
		exchangeIdToBufferedChannel[exchangeId] = bCh
	}

	return etmp, exchangeIdToBufferedChannel
}

func runPriceEncoderSequentially(
	t *testing.T,
	exchangeId types.ExchangeId,
	configs types.PricefeedMutableMarketConfigs,
	etmp types.ExchangeToMarketPrices,
	bCh chan *price_fetcher.PriceFetcherSubtaskResponse,
	writes []*types.MarketPriceTimestamp,
) {
	t.Helper()
	// Make sure there are not more write than the `bufferedChannel` can hold.
	require.True(t, len(writes) <= maxBufferedChannelLength)

	for _, write := range writes {
		bCh <- &price_fetcher.PriceFetcherSubtaskResponse{
			Price: write,
			Err:   nil,
		}
	}

	close(bCh)
	subTaskRunnerImpl.StartPriceEncoder(exchangeId, configs, etmp, log.NewNopLogger(), bCh)
}

func runPriceEncoderConcurrently(
	exchangeId types.ExchangeId,
	configs types.PricefeedMutableMarketConfigs,
	etmp types.ExchangeToMarketPrices,
	bCh chan *price_fetcher.PriceFetcherSubtaskResponse,
	writes []*types.MarketPriceTimestamp,
) {
	// Start a `waitGroup` for the `PriceEncoder` which will complete when the `bufferedChannel`
	// is empty and is closed.
	var priceEncoderWg sync.WaitGroup
	priceEncoderWg.Add(1)
	go func() {
		defer priceEncoderWg.Done()
		subTaskRunnerImpl.StartPriceEncoder(exchangeId, configs, etmp, log.NewNopLogger(), bCh)
	}()

	// Start a `waitGroup` for threads that will write to the `bufferedChannel`.
	var writeWg sync.WaitGroup
	for _, write := range writes {
		writeWg.Add(1)
		go func(write *types.MarketPriceTimestamp) {
			defer writeWg.Done()
			bCh <- &price_fetcher.PriceFetcherSubtaskResponse{
				Price: write,
				Err:   nil,
			}
		}(write)
	}

	writeWg.Wait()
	close(bCh)
	priceEncoderWg.Wait()
}

func sortMarketPriceUpdateByMarketIdDescending(
	marketPriceUpdate []*servertypes.MarketPriceUpdate,
) {
	sort.Slice(
		marketPriceUpdate,
		func(i, j int) bool {
			return marketPriceUpdate[i].MarketId > marketPriceUpdate[j].MarketId
		},
	)
}

func genMockPricefeedMutableMarketConfigsForExchange(
	exchangeId types.ExchangeId,
) types.PricefeedMutableMarketConfigs {
	mutableExchangeConfig := &types.MutableExchangeMarketConfig{
		Id: exchangeId,
		MarketToMarketConfig: map[types.MarketId]types.MarketConfig{
			8: {
				Ticker: "MARKET8-USD",
			},
			9: {
				Ticker: "MARKET9-USD",
			},
		},
	}
	mutableMarketConfigs := []*types.MutableMarketConfig{
		{
			Id:           constants.MarketId8,
			Pair:         "MARKET8-USD",
			Exponent:     -9,
			MinExchanges: 1,
		},
		{
			Id:           constants.MarketId9,
			Pair:         "MARKET9-USD",
			Exponent:     -9,
			MinExchanges: 1,
		},
	}
	configs := &mocks.PricefeedMutableMarketConfigs{}
	configs.On("GetExchangeMarketConfigCopy", exchangeId).Return(mutableExchangeConfig, nil)

	// All possible permutations of supported markets.
	configs.On("GetMarketConfigCopies", []types.MarketId{8, 9}).Return(mutableMarketConfigs, nil)
	configs.On("GetMarketConfigCopies", []types.MarketId{8}).Return(mutableMarketConfigs[0:1], nil)
	configs.On("GetMarketConfigCopies", []types.MarketId{9}).Return(mutableMarketConfigs[1:2], nil)
	configs.On("GetMarketConfigCopies", []types.MarketId{}).Return([]*types.MutableMarketConfig{}, nil)

	configs.On("AddPriceFetcher", mock.Anything).Return(nil)
	configs.On("AddPriceEncoder", mock.Anything).Return(nil)
	return configs
}
