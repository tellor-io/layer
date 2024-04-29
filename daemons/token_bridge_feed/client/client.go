package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"cosmossdk.io/log"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	tokenbridge "github.com/tellor-io/layer/daemons/token_bridge_feed/abi"
)

type Client struct {
	// Add necessary fields
	lastReportedDepositId *big.Int
	pendingReports        []DepositReport
	logger                log.Logger

	daemonStartup sync.WaitGroup

	runningSubtasksWaitGroup sync.WaitGroup

	tickers []*time.Ticker

	stops []chan bool

	stopDaemon sync.Once

	ethClient *ethclient.Client

	bridgeContract *tokenbridge.TokenBridge
}

func StartNewClient(ctx context.Context, logger log.Logger) *Client {
	logger.Info("Starting tokenbridge daemon")

	client := newClient(logger)
	client.runningSubtasksWaitGroup.Add(1)
	go func() {
		defer client.runningSubtasksWaitGroup.Done()
		client.start(ctx)
	}()
	return client
}

func newClient(logger log.Logger) *Client {
	logger = logger.With(log.ModuleKey, "tokenbridge-daemon")
	client := &Client{
		tickers: []*time.Ticker{},
		stops:   []chan bool{},
		logger:  logger,
	}

	// Set the client's daemonStartup state to indicate that the daemon has not finished starting up.
	client.daemonStartup.Add(1)
	return client
}

func (c *Client) start(ctx context.Context) {

	c.InitializeDeposits()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := c.QueryTokenBridgeContract()
			if err != nil {
				c.logger.Error("Failed to query and process deposits", "error", err)
			}
		}
	}
}

type DepositReceipt struct {
	DepositId *big.Int
	Sender    common.Address
	Recipient string
	Amount    *big.Int
}

type DepositReport struct {
	QueryData []byte
	Value     string
}

func (c *Client) QueryAPI(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make API request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read API response: %v", err)
	}

	return body, nil
}

// func (c *Client) QueryBridgeDeposits() error {
// 	client, err := ethclient.Dial("ws://127.0.0.1:7545")
// 	if err != nil {
// 		return fmt.Errorf("failed to connect to the Ethereum client: %v", err)
// 	}

// 	contractAddress := common.HexToAddress("0x47F2853f1f85E2c3E3194cC3152769E3c9900a4e")

// 	tbContract, err := tokenbridge.NewTokenBridge(contractAddress, client)
// 	if err != nil {
// 		return fmt.Errorf("failed to instantiate a TokenBridge contract: %v", err)
// 	}

// 	latestDepositId, err := c.QueryCurrentDepositId(tbContract)
// 	if err != nil {
// 		return fmt.Errorf("failed to query the latest deposit ID: %v", err)
// 	}

// 	if c.lastReportedDepositId == nil {
// 		c.lastReportedDepositId = big.NewInt(0)
// 	}

// 	if latestDepositId.Uint64() > c.lastReportedDepositId.Uint64() {
// 		c.lastReportedDepositId = big.NewInt(int64(c.lastReportedDepositId.Uint64() + 1))

// 		depositTicket, err := c.QueryDepositDetails(tbContract, c.lastReportedDepositId)
// 		if err != nil {
// 			return fmt.Errorf("failed to query deposit details: %v", err)
// 		}

// 		// assemble and add to pending reports
// 		queryData, err := c.EncodeQueryData(depositTicket)
// 		if err != nil {
// 			c.logger.Error("Failed to encode query data", "error", err)
// 		}
// 		reportValue, err := c.EncodeReportValue(depositTicket)
// 		if err != nil {
// 			c.logger.Error("Failed to encode report value", "error", err)
// 		}
// 		c.pendingReports = append(c.pendingReports, DepositReport{queryData, reportValue})
// 	}

// 	return nil
// }

func (c *Client) InitializeDeposits() error {
	c.logger.Info("Initializing token bridge client")
	eclient, err := ethclient.Dial("ws://127.0.0.1:7545")
	if err != nil {
		return fmt.Errorf("failed to connect to the Ethereum client: %v", err)
	}

	c.ethClient = eclient

	contractAddress := common.HexToAddress("0x47F2853f1f85E2c3E3194cC3152769E3c9900a4e")

	bridgeContract, err := tokenbridge.NewTokenBridge(contractAddress, c.ethClient)
	if err != nil {
		return fmt.Errorf("failed to instantiate a TokenBridge contract: %v", err)
	}

	c.bridgeContract = bridgeContract

	latestDepositId, err := c.QueryCurrentDepositId()
	if err != nil {
		return fmt.Errorf("failed to query the latest deposit ID: %v", err)
	}

	c.lastReportedDepositId = latestDepositId

	c.logger.Info("Last reported deposit ID", "depositId", c.lastReportedDepositId)
	c.logger.Info("Initialized token bridge client")

	return nil
}

func (c *Client) QueryTokenBridgeContract() error {
	latestDepositId, err := c.QueryCurrentDepositId()
	if err != nil {
		return fmt.Errorf("failed to query the latest deposit ID: %v", err)
	}

	if c.lastReportedDepositId == nil {
		c.lastReportedDepositId = big.NewInt(0)
	}

	if latestDepositId.Uint64() > c.lastReportedDepositId.Uint64() {
		c.lastReportedDepositId = big.NewInt(int64(c.lastReportedDepositId.Uint64() + 1))

		depositTicket, err := c.QueryDepositDetails(c.lastReportedDepositId)
		if err != nil {
			return fmt.Errorf("failed to query deposit details: %v", err)
		}

		// assemble and add to pending reports
		queryData, err := c.EncodeQueryData(depositTicket)
		if err != nil {
			c.logger.Error("Failed to encode query data", "error", err)
		}
		reportValue, err := c.EncodeReportValue(depositTicket)
		if err != nil {
			c.logger.Error("Failed to encode report value", "error", err)
		}
		c.pendingReports = append(c.pendingReports, DepositReport{queryData, reportValue})

		c.logger.Info("Added deposit to pending reports", "depositId", c.lastReportedDepositId)
	}

	return nil
}

func (c *Client) QueryCurrentDepositId() (*big.Int, error) {
	// Query the latest deposit ID from the bridge contract
	latestDepositId, err := c.bridgeContract.DepositId(nil)
	if err != nil {
		return latestDepositId, fmt.Errorf("failed to query latest deposit ID: %v", err)
	}

	return latestDepositId, nil
}

func (c *Client) QueryDepositDetails(depositId *big.Int) (DepositReceipt, error) {
	// Query depositDetails details for a specific depositDetails ID
	depositDetails, err := c.bridgeContract.Deposits(nil, depositId)
	if err != nil {
		return DepositReceipt{}, fmt.Errorf("failed to query deposit details for ID %d: %v", depositId, err)
	}

	depositReceipt := DepositReceipt{
		DepositId: depositId,
		Sender:    depositDetails.Sender,
		Recipient: depositDetails.Recipient,
		Amount:    depositDetails.Amount,
	}

	return depositReceipt, nil
}

func (c *Client) EncodeQueryData(depositReceipt DepositReceipt) ([]byte, error) {
	// encode query data
	queryTypeString := "TRBBridge"
	toLayerBool := true
	// prepare encoding
	StringType, err := abi.NewType("string", "", nil)
	if err != nil {
		return nil, err
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	BoolType, err := abi.NewType("bool", "", nil)
	if err != nil {
		return nil, err
	}
	BytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}

	// encode query data arguments first
	queryDataArgs := abi.Arguments{
		{Type: BoolType},
		{Type: Uint256Type},
	}
	queryDataArgsEncoded, err := queryDataArgs.Pack(toLayerBool, depositReceipt.DepositId)
	if err != nil {
		return nil, err
	}

	// encode query data
	finalArgs := abi.Arguments{
		{Type: StringType},
		{Type: BytesType},
	}
	queryDataEncoded, err := finalArgs.Pack(queryTypeString, queryDataArgsEncoded)
	if err != nil {
		return nil, err
	}
	return queryDataEncoded, nil
}

func (c *Client) EncodeReportValue(depositReceipt DepositReceipt) (string, error) {
	// replicate solidity encoding, abi.encode(address ethSender, string layerRecipient, uint256 amount)

	// prepare encoding
	AddressType, err := abi.NewType("address", "", nil)
	if err != nil {
		return "", err
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return "", err
	}
	StringType, err := abi.NewType("string", "", nil)
	if err != nil {
		return "", err
	}

	reportValueArgs := abi.Arguments{
		{Type: AddressType},
		{Type: StringType},
		{Type: Uint256Type},
	}

	// encode report value arguments
	reportValueArgsEncoded, err := reportValueArgs.Pack(depositReceipt.Sender, depositReceipt.Recipient, depositReceipt.Amount)
	if err != nil {
		return "", err
	}

	reportValString := hex.EncodeToString(reportValueArgsEncoded)

	return reportValString, nil
}

func (c *Client) GetPendingBridgeDeposit() (DepositReport, error) {
	if len(c.pendingReports) == 0 {
		return DepositReport{}, fmt.Errorf("no pending bridge deposits")
	}

	report := c.pendingReports[0]
	c.pendingReports = c.pendingReports[1:]
	return report, nil
}
