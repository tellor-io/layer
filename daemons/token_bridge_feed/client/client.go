package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	tokenbridgetypes "github.com/tellor-io/layer/daemons/server/types/token_bridge"
	tokenbridge "github.com/tellor-io/layer/daemons/token_bridge_feed/abi"

	"cosmossdk.io/log"
)

type Client struct {
	// Add necessary fields
	lastReportedDepositId *big.Int
	logger                log.Logger
	tokenDepositsCache    *tokenbridgetypes.DepositReports

	daemonStartup sync.WaitGroup

	runningSubtasksWaitGroup sync.WaitGroup

	tickers []*time.Ticker

	stops []chan bool

	ethClient *ethclient.Client

	bridgeContract *tokenbridge.TokenBridge
}

// Struct to unmarshal the JSON data
type APIResponse struct {
	Status string `json:"status"`
	Data   []struct {
		ExecBlockNumber int `json:"exec_block_number"`
	} `json:"data"`
}

func StartNewClient(ctx context.Context, logger log.Logger, tokenDepositsCache *tokenbridgetypes.DepositReports) *Client {
	logger.Info("Starting tokenbridge daemon")

	client := newClient(logger, tokenDepositsCache)
	client.runningSubtasksWaitGroup.Add(1)
	go func() {
		defer client.runningSubtasksWaitGroup.Done()
		client.start(ctx)
	}()
	return client
}

func newClient(logger log.Logger, tokenDepositsCache *tokenbridgetypes.DepositReports) *Client {
	logger = logger.With(log.ModuleKey, "tokenbridge-daemon")
	client := &Client{
		tickers:            []*time.Ticker{},
		stops:              []chan bool{},
		logger:             logger,
		tokenDepositsCache: tokenDepositsCache,
	}

	// Set the client's daemonStartup state to indicate that the daemon has not finished starting up.
	client.daemonStartup.Add(1)
	return client
}

func (c *Client) start(ctx context.Context) {
	if err := c.InitializeDeposits(); err != nil {
		c.logger.Error("Failed to initialize deposits", "error", err)
		return
	}
	ticker := time.NewTicker(3 * time.Minute)
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
	DepositId   *big.Int
	Sender      common.Address
	Recipient   string
	Amount      *big.Int
	Tip         *big.Int
	BlockHeight *big.Int
}

type DepositReport struct {
	QueryData []byte
	Value     string
}

func (c *Client) QueryAPI(urlStr string) ([]byte, error) {
	c.logger.Info("querying token_bridge_client api")
	parsedUrl, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return nil, err
	}
	resp, err := http.Get(parsedUrl.String())
	if err != nil {
		return nil, fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read API response: %w", err)
	}

	return body, nil
}

func (c *Client) InitializeDeposits() error {
	// Add retry logic for initial connection
	var eclient *ethclient.Client
	var err error
	var ethRpcUrl string
	for retries := 0; retries < 3; retries++ {
		ethRpcUrl, err = c.getEthRpcUrl()
		if err != nil {
			return fmt.Errorf("failed to get ETH RPC url: %w", err)
		}

		eclient, err = ethclient.Dial(ethRpcUrl)
		if err != nil {
			c.logger.Error("Failed to connect to Ethereum client, retrying...", "attempt", retries+1, "error", err)
			time.Sleep(time.Second * 5)
			continue
		}
		break
	}
	if err != nil {
		return fmt.Errorf("failed to connect to the Ethereum client after retries: %w", err)
	}

	c.ethClient = eclient

	contractAddress, err := c.getTokenBridgeContractAddress()
	if err != nil {
		return fmt.Errorf("failed to get token bridge contract address: %w", err)
	}

	bridgeContract, err := tokenbridge.NewTokenBridge(contractAddress, c.ethClient)
	if err != nil {
		return fmt.Errorf("failed to instantiate a TokenBridge contract: %w", err)
	}

	c.bridgeContract = bridgeContract

	latestDepositId, err := c.QueryCurrentDepositId()
	if err != nil {
		return fmt.Errorf("failed to query the latest deposit ID: %w", err)
	}

	c.lastReportedDepositId = latestDepositId

	return nil
}

func (c *Client) QueryTokenBridgeContract() error {
	// Add retry logic for queries
	var latestDepositId *big.Int
	var err error

	for retries := 0; retries < 3; retries++ {
		latestDepositId, err = c.QueryCurrentDepositId()
		if err != nil {
			if retries < 2 {
				c.logger.Error("Failed to query latest deposit ID, reconnecting...", "attempt", retries+1, "error", err)
				// Attempt to reconnect
				if err := c.reconnectEthClient(); err != nil {
					c.logger.Error("Failed to reconnect", "error", err)
					time.Sleep(time.Second * 5)
					continue
				}
			} else {
				return fmt.Errorf("failed to query the latest deposit ID: %w", err)
			}
		} else {
			break
		}
	}

	if c.lastReportedDepositId == nil {
		c.lastReportedDepositId = big.NewInt(0)
	}

	if latestDepositId.Uint64() > c.lastReportedDepositId.Uint64() {
		nextDepositId := big.NewInt(int64(c.lastReportedDepositId.Uint64() + 1))

		depositTicket, err := c.QueryDepositDetails(nextDepositId)
		if err != nil {
			return fmt.Errorf("failed to query deposit details: %w", err)
		}

		// Check if the block height is final
		isFinal, err := c.CheckForFinality(depositTicket.BlockHeight)
		if err != nil {
			return fmt.Errorf("failed to check if block height is final: %w", err)
		}

		if !isFinal {
			c.logger.Info("Block height is not final", "blockHeight", depositTicket.BlockHeight)
			return nil
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

		// Update the token deposits cache
		c.tokenDepositsCache.AddReport(tokenbridgetypes.DepositReport{QueryData: queryData, Value: reportValue})

		// Update the last reported deposit ID
		c.lastReportedDepositId = nextDepositId
		c.logger.Info("Added deposit to pending reports", "depositId", c.lastReportedDepositId)
	}

	return nil
}

func (c *Client) QueryCurrentDepositId() (*big.Int, error) {
	// Query the latest deposit ID from the bridge contract
	latestDepositId, err := c.bridgeContract.DepositId(nil)
	if err != nil {
		return latestDepositId, fmt.Errorf("failed to query latest deposit ID: %w", err)
	}

	return latestDepositId, nil
}

func (c *Client) QueryDepositDetails(depositId *big.Int) (DepositReceipt, error) {
	// Query depositDetails details for a specific depositDetails ID
	depositDetails, err := c.bridgeContract.Deposits(nil, depositId)
	if err != nil {
		return DepositReceipt{}, fmt.Errorf("failed to query deposit details for ID %d: %w", depositId, err)
	}

	depositReceipt := DepositReceipt{
		DepositId:   depositId,
		Sender:      depositDetails.Sender,
		Recipient:   depositDetails.Recipient,
		Amount:      depositDetails.Amount,
		Tip:         depositDetails.Tip,
		BlockHeight: depositDetails.BlockHeight,
	}

	return depositReceipt, nil
}

func (c *Client) CheckForFinality(blockHeight *big.Int) (bool, error) {
	// Check if the block height is final
	url := "https://sepolia.beaconcha.in/api/v1/epoch/finalized/slots"
	body, err := c.QueryAPI(url)
	if err != nil {
		c.logger.Error("Failed to query API", "error", err)
		return false, err
	}

	var apiResponse APIResponse
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return false, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	// Find the highest exec_block_number
	var highestBlockNumber int
	for _, data := range apiResponse.Data {
		if data.ExecBlockNumber > highestBlockNumber {
			highestBlockNumber = data.ExecBlockNumber
		}
	}

	// Check if the input blockHeight is greater than or equal to the highest exec_block_number
	return uint64(highestBlockNumber) >= blockHeight.Uint64(), nil
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

// replicate solidity encoding, abi.encode(address ethSender, string layerRecipient, uint256 amount)
func (c *Client) EncodeReportValue(depositReceipt DepositReceipt) ([]byte, error) {
	// prepare encoding
	AddressType, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, err
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	StringType, err := abi.NewType("string", "", nil)
	if err != nil {
		return nil, err
	}

	reportValueArgs := abi.Arguments{
		{Type: AddressType},
		{Type: StringType},
		{Type: Uint256Type},
		{Type: Uint256Type},
	}

	// encode report value arguments
	reportValueArgsEncoded, err := reportValueArgs.Pack(depositReceipt.Sender, depositReceipt.Recipient, depositReceipt.Amount, depositReceipt.Tip)
	if err != nil {
		return nil, err
	}

	return reportValueArgsEncoded, nil
}

func (c *Client) getEthRpcUrl() (string, error) {
	ethRpcUrl := os.Getenv("ETH_RPC_URL")
	if ethRpcUrl == "" {
		return "", fmt.Errorf("ETH_RPC_URL not set")
	} else {
		fmt.Println("ETH_RPC_URL", ethRpcUrl)
	}
	return ethRpcUrl, nil
}

func (c *Client) getTokenBridgeContractAddress() (common.Address, error) {
	tokenBridgeContractAddress := os.Getenv("TOKEN_BRIDGE_CONTRACT")
	if tokenBridgeContractAddress == "" {
		return common.Address{}, fmt.Errorf("TOKEN_BRIDGE_CONTRACT not set")
	} else {
		fmt.Println("TOKEN_BRIDGE_CONTRACT", tokenBridgeContractAddress)
	}
	return common.HexToAddress(tokenBridgeContractAddress), nil
}

// Add new helper function for reconnection
func (c *Client) reconnectEthClient() error {
	ethRpcUrl, err := c.getEthRpcUrl()
	if err != nil {
		return fmt.Errorf("failed to get ETH RPC url: %w", err)
	}

	eclient, err := ethclient.Dial(ethRpcUrl)
	if err != nil {
		return fmt.Errorf("failed to reconnect to the Ethereum client: %w", err)
	}

	// Close existing client if it exists
	if c.ethClient != nil {
		c.ethClient.Close()
	}

	c.ethClient = eclient

	// Reinstantiate the bridge contract with new client
	contractAddress, err := c.getTokenBridgeContractAddress()
	if err != nil {
		return fmt.Errorf("failed to get token bridge contract address: %w", err)
	}

	bridgeContract, err := tokenbridge.NewTokenBridge(contractAddress, c.ethClient)
	if err != nil {
		return fmt.Errorf("failed to reinstantiate TokenBridge contract: %w", err)
	}

	c.bridgeContract = bridgeContract
	return nil
}
