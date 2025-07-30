package main

import (
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	DefaultRpcURL        = "127.0.0.1:26657"
	MaxReconnectAttempts = 5
	BlockQueryInterval   = 1 * time.Second
)

var (
	// Command line parameters
	rpcURL         string
	configFilePath string
	nodeName       string
	// Block tracking variables
	currentBlockHeight uint64
	blockHeightMutex   sync.RWMutex
	// CSV file variables
	csvFile   *os.File
	csvWriter *csv.Writer
	csvMutex  sync.Mutex
)

type RPCRequest struct {
	Jsonrpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Id      int         `json:"id"`
	Params  interface{} `json:"params"`
}

type BlockResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      int    `json:"id"`
	Result  struct {
		BlockID struct {
			Hash  string `json:"hash"`
			Parts struct {
				Total int    `json:"total"`
				Hash  string `json:"hash"`
			} `json:"parts"`
		} `json:"block_id"`
		Block struct {
			Header struct {
				Version struct {
					Block string `json:"block"`
				} `json:"version"`
				ChainID     string `json:"chain_id"`
				Height      string `json:"height"`
				Time        string `json:"time"`
				LastBlockID struct {
					Hash  string `json:"hash"`
					Parts struct {
						Total int    `json:"total"`
						Hash  string `json:"hash"`
					} `json:"parts"`
				} `json:"last_block_id"`
				LastCommitHash     string `json:"last_commit_hash"`
				DataHash           string `json:"data_hash"`
				ValidatorsHash     string `json:"validators_hash"`
				NextValidatorsHash string `json:"next_validators_hash"`
				ConsensusHash      string `json:"consensus_hash"`
				AppHash            string `json:"app_hash"`
				LastResultsHash    string `json:"last_results_hash"`
				EvidenceHash       string `json:"evidence_hash"`
				ProposerAddress    string `json:"proposer_address"`
			} `json:"header"`
			Data struct {
				Txs []string `json:"txs"`
			} `json:"data"`
			Evidence struct {
				Evidence []interface{} `json:"evidence"`
			} `json:"evidence"`
			LastCommit struct {
				Height  string `json:"height"`
				Round   int    `json:"round"`
				BlockID struct {
					Hash  string `json:"hash"`
					Parts struct {
						Total int    `json:"total"`
						Hash  string `json:"hash"`
					} `json:"parts"`
				} `json:"block_id"`
				Signatures []struct {
					BlockIDFlag      int    `json:"block_id_flag"`
					ValidatorAddress string `json:"validator_address"`
					Timestamp        string `json:"timestamp"`
					Signature        string `json:"signature"`
				} `json:"signatures"`
			} `json:"last_commit"`
		} `json:"block"`
	} `json:"result"`
}

type ConfigType struct {
	AlertName  string `yaml:"alert_name"`
	AlertType  string `yaml:"alert_type"`
	Query      string `yaml:"query"`
	WebhookURL string `yaml:"webhook_url"`
}

type EventConfig struct {
	EventTypes []ConfigType `yaml:"event_types"`
}

// VoteExtensionData represents the decoded structure of vote extension data
type VoteExtensionData struct {
	BlockHeight   uint64 `json:"block_height"`
	OpAndEvmAddrs struct {
		OperatorAddresses []string `json:"operator_addresses"`
		EvmAddresses      []string `json:"evm_addresses"`
	} `json:"op_and_evm_addrs"`
	ValsetSigs struct {
		OperatorAddresses []string `json:"operator_addresses"`
		Timestamps        []string `json:"timestamps"`
		Signatures        []string `json:"signatures"`
	} `json:"valset_sigs"`
	OracleAttestations struct {
		OperatorAddresses []string `json:"operator_addresses"`
		Attestations      []string `json:"attestations"`
		Snapshots         []string `json:"snapshots"`
	} `json:"oracle_attestations"`
	ExtendedCommitInfo struct {
		Votes []struct {
			Validator struct {
				Address string `json:"address"`
				Power   int    `json:"power"`
			} `json:"validator"`
			VoteExtension      string `json:"vote_extension"`
			ExtensionSignature string `json:"extension_signature"`
			BlockIDFlag        int    `json:"block_id_flag"`
		} `json:"votes"`
	} `json:"extended_commit_info"`
}

// initCSVFile initializes the CSV file with headers
func initCSVFile() error {
	var err error

	// Check if file already exists
	fileExists := false
	if _, err := os.Stat("vote_extension_participation.csv"); err == nil {
		fileExists = true
		log.Println("CSV file already exists, will append to existing data")
	}

	// Open file in append mode if it exists, create if it doesn't
	if fileExists {
		csvFile, err = os.OpenFile("vote_extension_participation.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("failed to open existing CSV file: %w", err)
		}
	} else {
		csvFile, err = os.Create("vote_extension_participation.csv")
		if err != nil {
			return fmt.Errorf("failed to create CSV file: %w", err)
		}
	}

	csvWriter = csv.NewWriter(csvFile)

	// Only write headers if this is a new file
	if !fileExists {
		// Write headers
		headers := []string{"height", "timestamp", "vote_ext_participation_rate"}
		if err := csvWriter.Write(headers); err != nil {
			return fmt.Errorf("failed to write CSV headers: %w", err)
		}

		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			return fmt.Errorf("failed to flush CSV headers: %w", err)
		}

		log.Println("CSV file initialized: vote_extension_participation.csv")
	} else {
		log.Println("Appending to existing CSV file: vote_extension_participation.csv")
	}

	return nil
}

// writeToCSV writes a row to the CSV file
func writeToCSV(height, timestamp uint64, participationRate float64) error {
	csvMutex.Lock()
	defer csvMutex.Unlock()

	row := []string{
		strconv.FormatUint(height, 10),
		strconv.FormatUint(timestamp, 10),
		fmt.Sprintf("%.2f", participationRate),
	}

	if err := csvWriter.Write(row); err != nil {
		return fmt.Errorf("failed to write CSV row: %w", err)
	}

	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("failed to flush CSV: %w", err)
	}

	return nil
}

type HTTPClient struct {
	client      *http.Client
	baseURL     string
	protocol    string
	isLocalhost bool
	lastQuery   time.Time
	mu          sync.RWMutex
}

func NewHTTPClient(rpcURL string) *HTTPClient {
	var protocol string
	if strings.HasPrefix(rpcURL, "http://") || strings.HasPrefix(rpcURL, "localhost") {
		protocol = "http"
	} else {
		protocol = "https"
	}

	isLocalhost := strings.Contains(rpcURL, "localhost") || strings.Contains(rpcURL, "127.0.0.1")

	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:     rpcURL,
		protocol:    protocol,
		isLocalhost: isLocalhost,
	}
}

func (h *HTTPClient) makeRPCRequest(method string, params interface{}) ([]byte, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Rate limiting: ensure at least 10ms between requests
	if time.Since(h.lastQuery) < 500*time.Millisecond {
		time.Sleep(500*time.Millisecond - time.Since(h.lastQuery))
	}

	request := RPCRequest{
		Jsonrpc: "2.0",
		Method:  method,
		Id:      1,
		Params:  params,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := h.client.Post(h.baseURL, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	h.lastQuery = time.Now()
	return body, nil
}

func (h *HTTPClient) getLatestBlockHeight() (uint64, error) {
	body, err := h.makeRPCRequest("block", nil)
	if err != nil {
		return 0, err
	}

	var response BlockResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return 0, fmt.Errorf("failed to unmarshal block response: %w", err)
	}

	height, err := strconv.ParseUint(response.Result.Block.Header.Height, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse block height: %w", err)
	}

	return height, nil
}

func (h *HTTPClient) getBlock(height uint64) (*BlockResponse, error) {
	// Get block data
	blockBody, err := h.makeRPCRequest("block", map[string]interface{}{
		"height": fmt.Sprintf("%d", height),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	var blockResponse BlockResponse
	if err := json.Unmarshal(blockBody, &blockResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block response: %w", err)
	}

	return &blockResponse, nil
}

func (h *HTTPClient) healthCheck(ctx context.Context) {
	defer recoverAndAlert("HTTPClient.healthCheck")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check if we can still get the latest block height
			_, err := h.getLatestBlockHeight()
			if err != nil {
				log.Printf("Health check failed: %v", err)
				// Send liveness alert
				message := fmt.Sprintf("**Alert: Node %s is Not Responding**\nFailed to get latest block height from node %s. Please check the node status and logs.", nodeName, nodeName)
				// TODO: Implement alert sending logic
				log.Printf("Would send alert: %s", message)
			}
		}
	}
}

func calculateTotalValidatorSetPower(votes VoteExtensionData) (uint64, error) {
	totalPower := uint64(0)
	for _, vote := range votes.ExtendedCommitInfo.Votes {
		totalPower += uint64(vote.Validator.Power)
	}
	return totalPower, nil
}

func processBlock(blockResponse *BlockResponse, height uint64) error {
	log.Printf("Processing block %d", height)

	// Log basic block information
	if blockResponse.Result.Block.Header.Height != "" {
		log.Printf("Block %d time: %s", height, blockResponse.Result.Block.Header.Time)
		log.Printf("Block %d proposer: %s", height, blockResponse.Result.Block.Header.ProposerAddress)
		log.Printf("Block %d transactions: %d", height, len(blockResponse.Result.Block.Data.Txs))
	}

	// Check if there are any transactions in the block
	if len(blockResponse.Result.Block.Data.Txs) == 0 {
		log.Printf("Block %d has no transactions, skipping vote extension analysis", height)
		return nil
	}

	// Get the first transaction
	firstTx := blockResponse.Result.Block.Data.Txs[0]
	if firstTx == "" {
		log.Printf("Block %d first transaction is empty, skipping vote extension analysis", height)
		return nil
	}

	// Decode the base64 transaction
	decodedBytes, err := base64.StdEncoding.DecodeString(firstTx)
	if err != nil {
		log.Printf("Block %d failed to decode base64 transaction: %v", height, err)
		return fmt.Errorf("failed to decode base64 transaction: %w", err)
	}

	// Parse the JSON data
	var voteExtData VoteExtensionData
	if err := json.Unmarshal(decodedBytes, &voteExtData); err != nil {
		log.Printf("Block %d failed to unmarshal vote extension data: %v", height, err)
		return fmt.Errorf("failed to unmarshal vote extension data: %w", err)
	}

	totalValidatorSetPower, err := calculateTotalValidatorSetPower(voteExtData)
	if err != nil {
		log.Printf("Block %d failed to calculate total validator set power: %v", height, err)
		return fmt.Errorf("failed to calculate total validator set power: %w", err)
	}

	// Calculate vote extension participation rate
	totalVotes := len(blockResponse.Result.Block.LastCommit.Signatures)
	if totalVotes == 0 {
		log.Printf("Block %d has no votes in extended commit info", height)
		return nil
	}

	// Count votes with valid vote extensions (non-empty vote_extension field)
	votesWithExtensions := 0
	powerThatVoted := 0
	for _, vote := range voteExtData.ExtendedCommitInfo.Votes {
		if vote.VoteExtension != "" {
			votesWithExtensions++
			powerThatVoted += vote.Validator.Power
		}
	}

	// Calculate participation rate
	participationRate := float64(powerThatVoted) / float64(totalValidatorSetPower) * 100.0

	// Write to CSV file
	if err := writeToCSV(height, uint64(time.Now().Unix()), participationRate); err != nil {
		log.Printf("Block %d failed to write to CSV: %v", height, err)
		// Don't return error here as we still want to log the results
	}

	// Log the results
	log.Printf("Block %d Vote Extension Analysis:", height)
	log.Printf("  - Total votes: %d", totalVotes)
	log.Printf("  - Votes with extensions: %d", votesWithExtensions)
	log.Printf("  - Participation rate: %.2f%%", participationRate)
	log.Printf("  - Power that voted: %d", powerThatVoted)
	log.Printf("  - Total validator set power: %d", totalValidatorSetPower)

	// Log individual validator information for debugging
	for i, vote := range voteExtData.ExtendedCommitInfo.Votes {
		hasExtension := "No"
		if vote.VoteExtension != "" {
			hasExtension = "Yes"
		}
		log.Printf("  - Validator %d: Address=%s, Power=%d, HasExtension=%s",
			i+1, vote.Validator.Address, vote.Validator.Power, hasExtension)
	}

	// TODO: Add alerting logic for low participation rates
	if participationRate < 80.0 {
		log.Printf("WARNING: Block %d has low vote extension participation rate: %.2f%%", height, participationRate)
		// TODO: Send alert for low participation rate
	}

	return nil
}

func MonitorBlocks(ctx context.Context, wg *sync.WaitGroup) {
	defer func() {
		recoverAndAlert("MonitorBlocks")
		wg.Done()
	}()

	client := NewHTTPClient(rpcURL)

	// Get initial block height
	initialHeight, err := client.getLatestBlockHeight()
	if err != nil {
		log.Printf("Failed to get initial block height: %v", err)
		return
	}

	blockHeightMutex.Lock()
	currentBlockHeight = initialHeight
	blockHeightMutex.Unlock()

	log.Printf("Starting monitoring from block height: %d", initialHeight)

	// Start health check
	go client.healthCheck(ctx)

	// Main monitoring loop
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Get current block height
			blockHeightMutex.RLock()
			height := currentBlockHeight
			blockHeightMutex.RUnlock()

			// Try to get the next block with retry logic
			var blockResponse *BlockResponse
			var err error
			retryCount := 0
			fastRetries := 5
			totalAttempts := 0

			for {
				blockResponse, err = client.getBlock(height + 1)
				if err == nil {
					break
				}
				retryCount++
				log.Printf("Failed to get block %d (attempt %d/%d): %v", height+1, retryCount, fastRetries, err)
				if retryCount < fastRetries {
					time.Sleep(50 * time.Millisecond)
				} else {
					time.Sleep(250 * time.Millisecond)
				}

				if totalAttempts > 15 {
					log.Printf("Failed to get block %d after 15 attempts, sending crash alert and panicking", height+1)

					// Send crash alert before panicking
					message := fmt.Sprintf("**CRITICAL ALERT: Block Retrieval Failure**\nFailed to get block %d after 15 attempts on node %s. The monitor is crashing to prevent data loss.", height+1, nodeName)
					log.Printf("Would send crash alert: %s", message)

					// Panic to crash the application
					panic(fmt.Sprintf("Failed to get block %d after 15 attempts", height+1))
				}
			}

			// Process the block
			if err := processBlock(blockResponse, height+1); err != nil {
				log.Printf("Error processing block %d: %v", height+1, err)
				// Don't increment block height on error, retry the same block
				continue
			}

			// Check if the block was actually valid (not empty)
			if blockResponse.Result.Block.Header.Height == "" &&
				blockResponse.Result.Block.Header.Time == "" &&
				blockResponse.Result.Block.Header.ChainID == "" {
				// Block is empty, don't increment height, retry the same block
				log.Printf("Block %d is empty, retrying...", height+1)
				continue
			}

			// Only increment block height if we successfully processed a valid block
			blockHeightMutex.Lock()
			currentBlockHeight = height + 1
			blockHeightMutex.Unlock()

			log.Printf("Processed block %d", height+1)
		}
	}
}

func recoverAndAlert(goroutineName string) {
	if r := recover(); r != nil {
		message := fmt.Sprintf("**CRITICAL ALERT: Goroutine Crash**\nGoroutine '%s' has crashed with panic: %v\nPlease check the logs and restart the service.", goroutineName, r)
		log.Printf("Would send crash alert: %s", message)
		// Re-panic to maintain the original behavior
		panic(r)
	}
}

func main() {
	// Parse command line flags
	flag.StringVar(&rpcURL, "rpc-url", DefaultRpcURL, "RPC URL (default: 127.0.0.1:26657)")
	flag.StringVar(&configFilePath, "config", "", "Path to config file")
	flag.StringVar(&nodeName, "node", "", "Name of the node being monitored")
	flag.Parse()

	// Validate required parameters
	if nodeName == "" {
		log.Fatal("Usage: go run ./scripts/vote-ext-monitor/vote_ext_participation_rate_monitor.go -rpc-url=<rpc_url> -node=<node_name>")
	}

	// Initialize CSV file
	if err := initCSVFile(); err != nil {
		log.Fatalf("Failed to initialize CSV file: %v", err)
	}
	defer func() {
		if csvFile != nil {
			csvFile.Close()
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Received shutdown signal")
		cancel()
		// Give goroutines a chance to clean up
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go MonitorBlocks(ctx, &wg)

	wg.Wait()
	log.Println("Shutdown complete")
}
