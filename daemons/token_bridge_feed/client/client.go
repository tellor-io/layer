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
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	tokenbridgetypes "github.com/tellor-io/layer/daemons/server/types/token_bridge"
	tokenbridgetipstypes "github.com/tellor-io/layer/daemons/server/types/token_bridge_tips"
	tokenbridge "github.com/tellor-io/layer/daemons/token_bridge_feed/abi"

	"cosmossdk.io/log"
)

type Client struct {
	// Add necessary fields
	lastReportedDepositId *big.Int
	logger                log.Logger
	tokenDepositsCache    *tokenbridgetypes.DepositReports
	tokenBridgeTipsCache  *tokenbridgetipstypes.DepositTips
	daemonStartup         sync.WaitGroup

	runningSubtasksWaitGroup sync.WaitGroup

	tickers []*time.Ticker

	stops []chan bool

	ethClients      []*ethclient.Client
	bridgeContracts []*tokenbridge.TokenBridge
}

// Struct to unmarshal the JSON data
type APIResponse struct {
	Status string `json:"status"`
	Data   []struct {
		ExecBlockNumber int `json:"exec_block_number"`
	} `json:"data"`
}

func StartNewClient(ctx context.Context, logger log.Logger, tokenDepositsCache *tokenbridgetypes.DepositReports, tokenBridgeTipsCache *tokenbridgetipstypes.DepositTips) *Client {
	logger.Info("Starting tokenbridge daemon")

	client := newClient(logger, tokenDepositsCache, tokenBridgeTipsCache)
	client.runningSubtasksWaitGroup.Add(1)
	go func() {
		defer client.runningSubtasksWaitGroup.Done()
		client.start(ctx)
	}()
	return client
}

func newClient(logger log.Logger, tokenDepositsCache *tokenbridgetypes.DepositReports, tokenBridgeTipsCache *tokenbridgetipstypes.DepositTips) *Client {
	logger = logger.With(log.ModuleKey, "tokenbridge-daemon")
	client := &Client{
		tickers:              []*time.Ticker{},
		stops:                []chan bool{},
		logger:               logger,
		tokenDepositsCache:   tokenDepositsCache,
		tokenBridgeTipsCache: tokenBridgeTipsCache,
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
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Process regular deposits
			if err := c.QueryTokenBridgeContract(); err != nil {
				c.logger.Error("Failed to query and process deposits", "error", err)
			}

			// Process tips
			if err := c.ProcessPendingTips(); err != nil {
				c.logger.Error("Failed to process pending tips", "error", err)
			}
		}
	}
}

// Add new method to process tips
func (c *Client) ProcessPendingTips() error {
	oldestTipQueryData, err := c.tokenBridgeTipsCache.GetOldestTip()
	if err != nil {
		return nil
	}

	// Decode the query data to extract depositId
	queryType, depositId, err := c.DecodeQueryData(oldestTipQueryData.QueryData)
	if err != nil {
		c.logger.Error("Failed to decode tip query data", "error", err)
		c.tokenBridgeTipsCache.RemoveOldestTip()
		return nil
	}

	// Verify this is a TRBBridge query
	if queryType != "TRBBridge" {
		c.logger.Error("Invalid query type for tip", "queryType", queryType)
		c.tokenBridgeTipsCache.RemoveOldestTip()
		return nil
	}

	// Query deposit details
	depositTicket, err := c.QueryDepositDetails(depositId)
	if err != nil {
		c.logger.Error("Failed to query deposit details for tip", "error", err)
		return nil
	}

	// Check whether the deposit exists
	if depositTicket.Amount.Cmp(big.NewInt(0)) == 0 {
		c.logger.Info("Deposit does not exist", "depositId", depositId)
		c.tokenBridgeTipsCache.RemoveOldestTip()
		return nil
	}

	// Check finality
	isFinal, err := c.CheckForFinality(depositTicket.BlockHeight)
	if err != nil || !isFinal {
		c.logger.Info("Tip deposit not yet final", "depositId", depositId)
		return nil
	}

	reportValue, err := c.EncodeReportValue(depositTicket)
	if err != nil {
		c.logger.Error("Failed to encode report value for tip", "error", err)
		return nil
	}

	// Add to deposits cache
	c.tokenDepositsCache.AddReport(tokenbridgetypes.DepositReport{
		QueryData: oldestTipQueryData.QueryData,
		Value:     reportValue,
	})

	// Remove from tips cache
	c.tokenBridgeTipsCache.RemoveOldestTip()
	c.logger.Info("Processed tip and added to deposits", "depositId", depositId)

	return nil
}

// Add helper method to decode query data
func (c *Client) DecodeQueryData(queryData []byte) (string, *big.Int, error) {
	// Prepare types for decoding
	StringType, err := abi.NewType("string", "", nil)
	if err != nil {
		return "", nil, err
	}
	BytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return "", nil, err
	}

	// Decode outer layer
	args := abi.Arguments{{Type: StringType}, {Type: BytesType}}
	decoded, err := args.Unpack(queryData)
	if err != nil {
		return "", nil, err
	}

	queryType := decoded[0].(string)
	innerData := decoded[1].([]byte)

	// Decode inner layer
	BoolType, err := abi.NewType("bool", "", nil)
	if err != nil {
		return "", nil, err
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return "", nil, err
	}

	innerArgs := abi.Arguments{{Type: BoolType}, {Type: Uint256Type}}
	innerDecoded, err := innerArgs.Unpack(innerData)
	if err != nil {
		return "", nil, err
	}

	isDeposit := innerDecoded[0].(bool)
	if !isDeposit {
		return "", nil, fmt.Errorf("tip is not a deposit")
	}
	depositId := innerDecoded[1].(*big.Int)
	return queryType, depositId, nil
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
	urls, err := c.getEthRpcUrls()
	if err != nil {
		return fmt.Errorf("failed to get ETH RPC urls: %w", err)
	}

	// Initialize slices
	c.ethClients = make([]*ethclient.Client, len(urls))
	c.bridgeContracts = make([]*tokenbridge.TokenBridge, len(urls))

	// Connect to each RPC endpoint
	for i, url := range urls {
		for retries := 0; retries < 3; retries++ {
			client, err := ethclient.Dial(url)
			if err != nil {
				c.logger.Error("Failed to connect to Ethereum client, retrying...",
					"url", url, "attempt", retries+1, "error", err)
				time.Sleep(time.Second * 5)
				continue
			}
			c.ethClients[i] = client
			break
		}
		if c.ethClients[i] == nil {
			return fmt.Errorf("failed to connect to RPC endpoint: %s", url)
		}

		contractAddress, err := c.getTokenBridgeContractAddress()
		if err != nil {
			return fmt.Errorf("failed to get token bridge contract address: %w", err)
		}

		c.bridgeContracts[i], err = tokenbridge.NewTokenBridge(contractAddress, c.ethClients[i])
		if err != nil {
			return fmt.Errorf("failed to instantiate TokenBridge contract for endpoint %s: %w", url, err)
		}
	}

	latestDepositId, err := c.QueryCurrentDepositId()
	if err != nil {
		return fmt.Errorf("failed to query the latest deposit ID: %w", err)
	}

	c.lastReportedDepositId = latestDepositId

	return nil
}

func (c *Client) QueryTokenBridgeContract() error {
	var latestDepositId *big.Int
	var err error

	for retries := 0; retries < 3; retries++ {
		latestDepositId, err = c.QueryCurrentDepositId()
		if err != nil {
			if retries < 2 {
				c.logger.Error("Failed to query latest deposit ID, reconnecting...",
					"attempt", retries+1, "error", err)
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
	urls, err := c.getEthRpcUrls()
	if err != nil {
		return fmt.Errorf("failed to get ETH RPC urls: %w", err)
	}

	for i, url := range urls {
		// Close existing client if it exists
		if c.ethClients[i] != nil {
			c.ethClients[i].Close()
		}

		client, err := ethclient.Dial(url)
		if err != nil {
			c.logger.Error("Failed to reconnect to Ethereum client",
				"url", url, "error", err)
			continue
		}
		c.ethClients[i] = client

		contractAddress, err := c.getTokenBridgeContractAddress()
		if err != nil {
			return fmt.Errorf("failed to get token bridge contract address: %w", err)
		}

		c.bridgeContracts[i], err = tokenbridge.NewTokenBridge(contractAddress, client)
		if err != nil {
			return fmt.Errorf("failed to reinstantiate TokenBridge contract: %w", err)
		}
	}

	return nil
}

func (c *Client) getEthRpcUrls() ([]string, error) {
	ethRpcUrls := os.Getenv("ETH_RPC_URL")
	if ethRpcUrls == "" {
		return nil, fmt.Errorf("ETH_RPC_URL not set")
	}

	// Split by comma
	urls := strings.Split(ethRpcUrls, ",")

	// Trim whitespace
	for i, url := range urls {
		urls[i] = strings.TrimSpace(url)
	}

	return urls, nil
}

func (c *Client) QueryCurrentDepositId() (*big.Int, error) {
	var results []*big.Int

	// Query all RPCs
	for i, contract := range c.bridgeContracts {
		depositId, err := contract.DepositId(nil)
		if err != nil {
			c.logger.Error("Failed to query deposit ID from RPC",
				"index", i, "error", err)
			continue
		}
		results = append(results, depositId)
	}

	if len(results) < len(c.bridgeContracts)/2+1 {
		return nil, fmt.Errorf("failed to get majority consensus: insufficient responses")
	}

	// Find the majority value
	counts := make(map[string]int)
	for _, result := range results {
		counts[result.String()]++
	}

	var majorityValue *big.Int
	majorityCount := 0
	for valueStr, count := range counts {
		if count > majorityCount {
			majorityCount = count
			value, _ := new(big.Int).SetString(valueStr, 10)
			majorityValue = value
		}
	}

	// Ensure we have a majority
	if majorityCount <= len(c.bridgeContracts)/2 {
		return nil, fmt.Errorf("no majority consensus reached")
	}

	return majorityValue, nil
}

func (c *Client) QueryDepositDetails(depositId *big.Int) (DepositReceipt, error) {
	var results []DepositReceipt

	// Query all RPCs
	for i, contract := range c.bridgeContracts {
		deposit, err := contract.Deposits(nil, depositId)
		if err != nil {
			c.logger.Error("Failed to query deposit details from RPC",
				"index", i, "error", err)
			continue
		}

		results = append(results, DepositReceipt{
			DepositId:   depositId,
			Sender:      deposit.Sender,
			Recipient:   deposit.Recipient,
			Amount:      deposit.Amount,
			Tip:         deposit.Tip,
			BlockHeight: deposit.BlockHeight,
		})
	}

	if len(results) < len(c.bridgeContracts)/2+1 {
		return DepositReceipt{}, fmt.Errorf("failed to get majority consensus: insufficient responses")
	}

	// Find the majority value by comparing serialized results
	counts := make(map[string]int)
	receiptMap := make(map[string]DepositReceipt)

	for _, result := range results {
		serialized, _ := json.Marshal(result)
		key := string(serialized)
		counts[key]++
		receiptMap[key] = result
	}

	var majorityReceipt DepositReceipt
	majorityCount := 0
	for key, count := range counts {
		if count > majorityCount {
			majorityCount = count
			majorityReceipt = receiptMap[key]
		}
	}

	// Ensure we have a majority
	if majorityCount <= len(c.bridgeContracts)/2 {
		return DepositReceipt{}, fmt.Errorf("no majority consensus reached")
	}

	return majorityReceipt, nil
}
