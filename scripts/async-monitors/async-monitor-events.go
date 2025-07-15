package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/tellor-io/layer/utils"
	"gopkg.in/yaml.v3"
)

const (
	AlertCooldownPeriod  = 2 * time.Hour
	AlertWindowPeriod    = 10 * time.Minute
	MaxAlertsInWindow    = 10
	PowerThreshold       = 2.0 / 3.0
	DefaultRpcURL        = "127.0.0.1:26657"
	MaxReconnectAttempts = 5
	BlockQueryInterval   = 1 * time.Second
)

var (
	configMutex                  sync.RWMutex
	Current_Total_Reporter_Power uint64
	reporterPowerMutex           sync.RWMutex
	// Rate limiting variables
	AGGREGATE_REPORT_NAME    = "aggregate-report"
	aggregateAlertCount      int
	aggregateAlertTimestamps []time.Time
	aggregateAlertMutex      sync.RWMutex
	aggregateAlertCooldown   time.Time
	// Map to store event types we're interested in
	eventTypeMap map[string]ConfigType
	// Command line parameters
	rpcURL              string
	configFilePath      string
	nodeName            string
	blockTimeThreshold  time.Duration
	previousBlockTime   time.Time
	blockTimeMutex      sync.RWMutex
	lastBlockHeight     uint64
	isTimestampAnalyzer bool
	currentBlockHeight  uint64
	blockHeightMutex    sync.RWMutex
)

type Params struct {
	Query string `json:"query"`
}

type ConfigType struct {
	AlertName  string `yaml:"alert_name"`
	AlertType  string `yaml:"alert_type"`
	Query      string `yaml:"query"`
	WebhookURL string `yaml:"webhook_url"`
}

type RPCRequest struct {
	Jsonrpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Id      int         `json:"id"`
	Params  interface{} `json:"params"`
}

type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Index bool   `json:"index"`
}

type Event struct {
	Type       string      `json:"type"`
	Attributes []Attribute `json:"attributes"`
}

type TxResult struct {
	Code      int         `json:"code"`
	Data      interface{} `json:"data"`
	Log       string      `json:"log"`
	Info      string      `json:"info"`
	GasWanted string      `json:"gas_wanted"`
	GasUsed   string      `json:"gas_used"`
	Events    []Event     `json:"events"`
	Codespace string      `json:"codespace"`
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

type BlockResultsResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      int    `json:"id"`
	Result  struct {
		Height                string        `json:"height"`
		TxsResults            []TxResult    `json:"txs_results"`
		FinalizeBlockEvents   []Event       `json:"finalize_block_events"`
		ValidatorUpdates      []interface{} `json:"validator_updates"`
		ConsensusParamUpdates struct {
			Block struct {
				MaxBytes string `json:"max_bytes"`
				MaxGas   string `json:"max_gas"`
			} `json:"block"`
			Evidence struct {
				MaxAgeNumBlocks string `json:"max_age_num_blocks"`
				MaxAgeDuration  string `json:"max_age_duration"`
				MaxBytes        string `json:"max_bytes"`
			} `json:"evidence"`
			Validator struct {
				PubKeyTypes []string `json:"pub_key_types"`
			} `json:"validator"`
			Version struct{} `json:"version"`
			Abci    struct {
				VoteExtensionsEnableHeight string `json:"vote_extensions_enable_height"`
			} `json:"abci"`
		} `json:"consensus_param_updates"`
	} `json:"result"`
}

type EventConfig struct {
	EventTypes []ConfigType `yaml:"event_types"`
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
	// Check if URL already has a protocol
	if strings.HasPrefix(rpcURL, "http://") || strings.HasPrefix(rpcURL, "https://") {
		return &HTTPClient{
			client: &http.Client{
				Timeout: 30 * time.Second,
			},
			baseURL:     rpcURL,
			protocol:    "https", // Default to https for external URLs
			isLocalhost: false,
		}
	}

	protocol := "http"
	isLocalhost := strings.Contains(rpcURL, "localhost") || strings.Contains(rpcURL, "127.0.0.1")
	if !isLocalhost {
		protocol = "https"
	}

	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:     fmt.Sprintf("%s://%s", protocol, rpcURL),
		protocol:    protocol,
		isLocalhost: isLocalhost,
	}
}

func (h *HTTPClient) makeRPCRequest(method string, params interface{}) ([]byte, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Rate limiting: ensure at least 500ms between requests (instead of 1 second)
	if time.Since(h.lastQuery) < 250*time.Millisecond {
		time.Sleep(250*time.Millisecond - time.Since(h.lastQuery))
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

	var url string
	if h.isLocalhost {
		url = h.baseURL
	} else {
		url = h.baseURL + "/rpc"
	}

	resp, err := h.client.Post(url, "application/json", strings.NewReader(string(jsonData)))
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

func (h *HTTPClient) getBlock(height uint64) (*BlockResponse, *BlockResultsResponse, error) {
	// Get block data
	blockBody, err := h.makeRPCRequest("block", map[string]interface{}{
		"height": fmt.Sprintf("%d", height),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get block: %w", err)
	}

	var blockResponse BlockResponse
	if err := json.Unmarshal(blockBody, &blockResponse); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal block response: %w", err)
	}

	// Get block results
	resultsBody, err := h.makeRPCRequest("block_results", map[string]interface{}{
		"height": fmt.Sprintf("%d", height),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get block results: %w", err)
	}

	var resultsResponse BlockResultsResponse
	if err := json.Unmarshal(resultsBody, &resultsResponse); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal block results response: %w", err)
	}

	return &blockResponse, &resultsResponse, nil
}

func loadConfig() error {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	var newConfig EventConfig
	if err := yaml.Unmarshal(data, &newConfig); err != nil {
		return fmt.Errorf("error parsing config file: %w", err)
	}

	// Initialize the event type map
	newEventTypeMap := make(map[string]ConfigType)
	for _, et := range newConfig.EventTypes {
		newEventTypeMap[et.AlertType] = et
		fmt.Printf("Loaded event type: %s (%s)\n", et.AlertName, et.AlertType)
	}

	configMutex.Lock()
	eventTypeMap = newEventTypeMap
	configMutex.Unlock()
	return nil
}

func startConfigWatcher(ctx context.Context) {
	defer recoverAndAlert("startConfigWatcher")
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := loadConfig(); err != nil {
				fmt.Printf("Error reloading config: %v\n", err)
			}
		}
	}
}

func handleAggregateReport(event Event, eventType ConfigType) {
	aggregateAlertMutex.Lock()
	defer aggregateAlertMutex.Unlock()

	// Check if we're in cooldown
	if time.Now().Before(aggregateAlertCooldown) {
		fmt.Printf("In cooldown, skipping aggregate report\n")
		return
	}

	// Clean up old timestamps (older than 10 minutes)
	now := time.Now()
	tenMinutesAgo := now.Add(-10 * time.Minute)
	var validTimestamps []time.Time
	for _, ts := range aggregateAlertTimestamps {
		if ts.After(tenMinutesAgo) {
			validTimestamps = append(validTimestamps, ts)
		}
	}
	aggregateAlertTimestamps = validTimestamps
	aggregateAlertCount = len(validTimestamps)

	reporterPowerMutex.RLock()
	currentPower := Current_Total_Reporter_Power
	reporterPowerMutex.RUnlock()

	for j := 0; j < len(event.Attributes); j++ {
		if event.Attributes[j].Key == AGGREGATE_REPORT_NAME {
			if aggregatePower, err := strconv.ParseUint(event.Attributes[j].Value, 10, 64); err == nil {
				if float64(aggregatePower) < float64(currentPower)*2/3 {
					// Check if we've hit the alert limit
					if aggregateAlertCount >= 10 {
						// Send final alert and start cooldown
						message := fmt.Sprintf("**Rate Limit Reached: %s**\nToo many alerts in the last 10 minutes. Alerts will be paused for 2 hours. Please check on reporters and see what is going on\n", eventType.AlertName)
						for _, attr := range event.Attributes {
							message += fmt.Sprintf("%s: %s\n", attr.Key, attr.Value)
						}

						discordNotifier := utils.NewDiscordNotifier(eventType.WebhookURL)
						if err := discordNotifier.SendAlert(message); err != nil {
							fmt.Printf("Error sending final Discord alert: %v\n", err)
						} else {
							fmt.Printf("Sent final Discord alert and starting cooldown\n")
						}

						// Set cooldown for 2 hours
						aggregateAlertCooldown = now.Add(2 * time.Hour)
						aggregateAlertCount = 0
						aggregateAlertTimestamps = nil
						return
					}

					// Normal alert
					message := fmt.Sprintf("**Event Alert: %s**\n", eventType.AlertName)
					for _, attr := range event.Attributes {
						message += fmt.Sprintf("%s: %s\n", attr.Key, attr.Value)
					}

					discordNotifier := utils.NewDiscordNotifier(eventType.WebhookURL)
					if err := discordNotifier.SendAlert(message); err != nil {
						fmt.Printf("Error sending Discord alert for event %s: %v\n", event.Type, err)
					} else {
						fmt.Printf("Sent Discord alert for event: %s\n", event.Type)
						// Add timestamp for this alert
						aggregateAlertTimestamps = append(aggregateAlertTimestamps, now)
						aggregateAlertCount++
					}
				}
			} else {
				fmt.Printf("Error parsing aggregate power: %v\n", err)
			}
		}
	}
}

func MonitorBlockEvents(ctx context.Context, wg *sync.WaitGroup) {
	defer func() {
		recoverAndAlert("MonitorBlockEvents")
		wg.Done()
	}()

	if err := loadConfig(); err != nil {
		log.Printf("Error loading initial config: %v", err)
		return
	}

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

	// Start config watcher
	go startConfigWatcher(ctx)

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
			var resultsResponse *BlockResultsResponse
			var err error
			retryCount := 0
			fastRetries := 5
			totalAttempts := 0

			for {
				blockResponse, resultsResponse, err = client.getBlock(height + 1)
				if err == nil {
					break
				}
				retryCount++
				log.Printf("Failed to get block %d (attempt %d/%d): %v", height+1, retryCount, fastRetries, err)
				if retryCount < fastRetries {
					time.Sleep(100 * time.Millisecond)
				} else {
					time.Sleep(500 * time.Millisecond)
				}

				if totalAttempts > 15 {
					log.Printf("Failed to get block %d after 15 attempts, sending crash alert and panicking", height+1)

					// Send crash alert before panicking
					if eventType, exists := eventTypeMap["crash-alert"]; exists {
						message := fmt.Sprintf("**CRITICAL ALERT: Block Retrieval Failure**\nFailed to get block %d after 15 attempts on node %s. The monitor is crashing to prevent data loss.", height+1, nodeName)
						discordNotifier := utils.NewDiscordNotifier(eventType.WebhookURL)
						if err := discordNotifier.SendAlert(message); err != nil {
							log.Printf("Error sending crash Discord alert: %v", err)
						} else {
							log.Printf("Sent crash Discord alert for block retrieval failure")
						}
					}

					// Panic to crash the application
					panic(fmt.Sprintf("Failed to get block %d after 15 attempts", height+1))
				}
			}

			// Process the block
			if err := processBlock(blockResponse, resultsResponse, height+1); err != nil {
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
				if _, exists := eventTypeMap["liveness-alert"]; exists {
					discordNotifier := utils.NewDiscordNotifier(eventTypeMap["liveness-alert"].WebhookURL)
					if err := discordNotifier.SendAlert(message); err != nil {
						log.Printf("Error sending timeout Discord alert: %v", err)
					} else {
						log.Printf("Sent timeout Discord alert for node %s", nodeName)
					}
				}
			}
		}
	}
}

func processBlock(blockResponse *BlockResponse, resultsResponse *BlockResultsResponse, height uint64) error {
	// Check block time threshold if it's configured
	if blockTimeThreshold > 0 {
		// Parse current block time
		blockTimeStr := blockResponse.Result.Block.Header.Time

		// Skip block time analysis if the time field is empty
		if blockTimeStr == "" {
			fmt.Printf("Warning: Block %d has empty time field, skipping block time analysis\n", height)
		} else {
			currentBlockTime, err := time.Parse(time.RFC3339Nano, blockTimeStr)
			if err != nil {
				return fmt.Errorf("unable to parse block time: %w", err)
			}

			// Always log the extracted time field when block time threshold is set
			fmt.Printf("Block %d time: %s\n", height, blockTimeStr)

			blockTimeMutex.RLock()
			prevTime := previousBlockTime
			blockTimeMutex.RUnlock()

			// Skip first block after startup
			if !prevTime.IsZero() {
				timeDiff := currentBlockTime.Sub(prevTime)
				blockDiff := height - lastBlockHeight
				if blockDiff > 0 {
					normalizedTimeDiff := time.Duration(float64(timeDiff) / float64(blockDiff))
					fmt.Println("Normalized time per block: ", normalizedTimeDiff.String())
					if normalizedTimeDiff > blockTimeThreshold {
						message := fmt.Sprintf("**Alert: Abnormally Long Block Time**\nNode: %s\nTime between blocks: %v\nNormalized time per block: %v\nThreshold: %v\nPrevious block time: %v\nCurrent block time: %v",
							nodeName, timeDiff, normalizedTimeDiff, blockTimeThreshold, prevTime, currentBlockTime)

						if eventType, exists := eventTypeMap["block-time-alert"]; exists {
							discordNotifier := utils.NewDiscordNotifier(eventType.WebhookURL)
							if err := discordNotifier.SendAlert(message); err != nil {
								fmt.Printf("Error sending block time Discord alert: %v\n", err)
							} else {
								fmt.Printf("Sent block time Discord alert\n")
							}
						}
					}
				}
			}

			// Update previous block time
			blockTimeMutex.Lock()
			previousBlockTime = currentBlockTime
			lastBlockHeight = height
			blockTimeMutex.Unlock()
		}
	}

	// Process events from finalize block events
	if len(resultsResponse.Result.FinalizeBlockEvents) > 0 {
		// Count configured events for logging
		configuredEventCount := 0
		for _, event := range resultsResponse.Result.FinalizeBlockEvents {
			if _, exists := eventTypeMap[event.Type]; exists {
				configuredEventCount++
				// Skip logging aggregate_report events
				if event.Type != AGGREGATE_REPORT_NAME {
					fmt.Printf("Found configured event: %s\n", event.Type)
				}
			}
		}

		if configuredEventCount > 0 {
			fmt.Printf("Found %d configured events in block %d\n", configuredEventCount, height)
		}

		go processBlockEvents(resultsResponse.Result.FinalizeBlockEvents, fmt.Sprintf("%d", height))
	}

	// Process transaction events
	if len(resultsResponse.Result.TxsResults) > 0 {
		// Count configured events in transactions for logging
		txConfiguredEventCount := 0
		for i, txResult := range resultsResponse.Result.TxsResults {
			for _, event := range txResult.Events {
				if _, exists := eventTypeMap[event.Type]; exists {
					txConfiguredEventCount++
					// Skip logging aggregate_report events
					if event.Type != AGGREGATE_REPORT_NAME {
						fmt.Printf("Found configured event in tx %d: %s\n", i, event.Type)
					}
				}
			}
		}

		if txConfiguredEventCount > 0 {
			fmt.Printf("Found %d configured events in transactions for block %d\n", txConfiguredEventCount, height)
		}

		go processTransactionEvents(resultsResponse.Result.TxsResults, fmt.Sprintf("%d", height))
	}

	return nil
}

func processBlockEvents(events []Event, height string) {
	for _, event := range events {
		if eventType, exists := eventTypeMap[event.Type]; exists {
			fmt.Printf("Event at %s: %s\n", height, event.Type)
			handleEvent(event, eventType)
		}
	}
}

func processTransactionEvents(txResults []TxResult, height string) {
	for _, txResult := range txResults {
		for _, event := range txResult.Events {
			if eventType, exists := eventTypeMap[event.Type]; exists {
				fmt.Printf("Event at %s: %s\n", height, event.Type)
				handleEvent(event, eventType)
			}
		}
	}
}

// Add a helper function to write timestamps to a CSV file
func writeTimestampToCSV(timestamp string) error {
	// Check if file exists
	_, err := os.Stat("bridge_validator_timestamps.csv")
	fileExists := err == nil

	// Create the file if it doesn't exist, or append to it if it does
	file, err := os.OpenFile("bridge_validator_timestamps.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// If file was just created, write the header
	if !fileExists {
		if _, err := file.WriteString("validator_set_update_timestamps\n"); err != nil {
			return fmt.Errorf("error writing header to file: %w", err)
		}
	}

	// Write the timestamp to the CSV file
	if _, err := file.WriteString(timestamp + "\n"); err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	return nil
}

// Add a helper function to handle events
func handleEvent(event Event, eventType ConfigType) {
	if event.Type == AGGREGATE_REPORT_NAME {
		handleAggregateReport(event, eventType)
	} else {
		message := fmt.Sprintf("**Event Alert: %s**\n", eventType.AlertName)
		for _, attr := range event.Attributes {
			message += fmt.Sprintf("%s: %s\n", attr.Key, attr.Value)
		}

		discordNotifier := utils.NewDiscordNotifier(eventType.WebhookURL)
		if err := discordNotifier.SendAlert(message); err != nil {
			fmt.Printf("Error sending Discord alert for event %s: %v\n", event.Type, err)
		} else {
			fmt.Printf("Sent Discord alert for event: %s\n", event.Type)
		}
	}

	if event.Type == "new_bridge_validator_set" {
		for _, attr := range event.Attributes {
			if attr.Key == "timestamp" {
				if err := writeTimestampToCSV(attr.Value); err != nil {
					fmt.Printf("Error writing timestamp to CSV: %v\n", err)
				} else {
					fmt.Printf("Successfully wrote timestamp %s to CSV\n", attr.Value)
				}
				break
			}
		}
	}
}

type ValidatorResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		BlockHeight string `json:"block_height"`
		Validators  []struct {
			Address string `json:"address"`
			PubKey  struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"pub_key"`
			VotingPower      string `json:"voting_power"`
			ProposerPriority string `json:"proposer_priority"`
		} `json:"validators"`
		Count string `json:"count"`
		Total string `json:"total"`
	} `json:"result"`
}

func updateTotalReporterPower(ctx context.Context) {
	defer recoverAndAlert("updateTotalReporterPower")
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	client := NewHTTPClient(rpcURL)
	backoffDuration := 1 * time.Second
	maxBackoff := 5 * time.Minute

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			body, err := client.makeRPCRequest("validators", nil)
			if err != nil {
				fmt.Printf("Error querying validators: %v\n", err)
				// Implement exponential backoff
				time.Sleep(backoffDuration)
				backoffDuration *= 2
				if backoffDuration > maxBackoff {
					backoffDuration = maxBackoff
				}
				continue
			}

			var validatorResp ValidatorResponse
			if err := json.Unmarshal(body, &validatorResp); err != nil {
				fmt.Printf("Error decoding validator response: %v\n", err)
				time.Sleep(backoffDuration)
				backoffDuration *= 2
				if backoffDuration > maxBackoff {
					backoffDuration = maxBackoff
				}
				continue
			}

			var totalPower int64
			for _, validator := range validatorResp.Result.Validators {
				power, err := strconv.ParseInt(validator.VotingPower, 10, 64)
				if err != nil {
					fmt.Printf("Error parsing voting power: %v\n", err)
					continue
				}
				totalPower += power
			}
			fmt.Printf("Total power: %d\n", totalPower)

			reporterPowerMutex.Lock()
			Current_Total_Reporter_Power = uint64(totalPower)
			reporterPowerMutex.Unlock()

			fmt.Printf("Updated total reporter power: %d\n", totalPower)

			// Reset backoff on successful update
			backoffDuration = 1 * time.Second
		}
	}
}

func recoverAndAlert(goroutineName string) {
	if r := recover(); r != nil {
		var eventInfo ConfigType
		if info, exists := eventTypeMap["crash-alert"]; exists {
			eventInfo = info
		} else {
			log.Printf("No crash event type configured, skipping alert for goroutine: %s", goroutineName)
			panic(r)
		}
		message := fmt.Sprintf("**CRITICAL ALERT: Goroutine Crash**\nGoroutine '%s' has crashed with panic: %v\nPlease check the logs and restart the service.", goroutineName, r)
		discordNotifier := utils.NewDiscordNotifier(eventInfo.WebhookURL)
		if err := discordNotifier.SendAlert(message); err != nil {
			log.Printf("Error sending crash Discord alert: %v", err)
		} else {
			log.Printf("Sent crash Discord alert for goroutine: %s", goroutineName)
		}
		// Re-panic to maintain the original behavior
		panic(r)
	}
}

// Add a helper function to analyze validator set update timestamps
func analyzeValidatorSetUpdates(ctx context.Context) {
	defer recoverAndAlert("analyzeValidatorSetUpdates")

	// Run analysis once a day at 9 AM
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Calculate time until next 9 AM
	now := time.Now()
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())
	if now.After(nextRun) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	// Wait until 9 AM
	time.Sleep(nextRun.Sub(now))

	// Run initial analysis
	runAnalysis()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runAnalysis()
		}
	}
}

func runAnalysis() {
	// Check if CSV file exists
	_, err := os.Stat("bridge_validator_timestamps.csv")
	if err != nil {
		log.Printf("CSV file not found, skipping analysis: %v", err)
		return
	}

	// Read CSV file
	file, err := os.Open("bridge_validator_timestamps.csv")
	if err != nil {
		log.Printf("Error opening CSV file: %v", err)
		return
	}
	defer file.Close()

	// Read all lines
	scanner := bufio.NewScanner(file)
	var timestamps []time.Time

	// Skip header
	if scanner.Scan() {
		header := scanner.Text()
		if header != "validator_set_update_timestamps" {
			log.Printf("Unexpected CSV header: %s", header)
			return
		}
	}

	// Parse timestamps
	for scanner.Scan() {
		timestampStr := strings.TrimSpace(scanner.Text())
		if timestampStr == "" {
			continue
		}

		// Try different timestamp formats
		var timestamp time.Time
		var parseErr error

		// Try RFC3339 format first
		timestamp, parseErr = time.Parse(time.RFC3339, timestampStr)
		if parseErr != nil {
			// Try Unix timestamp
			if unixTime, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
				timestamp = time.Unix(unixTime, 0)
			} else {
				log.Printf("Error parsing timestamp %s: %v", timestampStr, parseErr)
				continue
			}
		}
		sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)
		if timestamp.After(sevenDaysAgo) {
			timestamps = append(timestamps, timestamp)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading CSV file: %v", err)
		return
	}

	if len(timestamps) < 2 {
		log.Printf("Not enough timestamps for analysis (need at least 2, got %d)", len(timestamps))
		return
	}

	// Sort timestamps
	sort.Slice(timestamps, func(i, j int) bool {
		return timestamps[i].Before(timestamps[j])
	})

	if len(timestamps) < 2 {
		log.Printf("Not enough recent timestamps for 7-day analysis (need at least 2, got %d)", len(timestamps))
		return
	}

	// Calculate time differences
	var timeDiffs []time.Duration
	for i := 1; i < len(timestamps); i++ {
		diff := timestamps[i].Sub(timestamps[i-1])
		timeDiffs = append(timeDiffs, diff)
	}

	// Calculate average
	var totalDuration time.Duration
	for _, diff := range timeDiffs {
		totalDuration += diff
	}
	averageDuration := totalDuration / time.Duration(len(timeDiffs))

	// Calculate median
	slices.Sort(timeDiffs)
	medianDuration := timeDiffs[len(timeDiffs)/2]
	if len(timeDiffs)%2 == 0 {
		medianDuration = (timeDiffs[len(timeDiffs)/2-1] + timeDiffs[len(timeDiffs)/2]) / 2
	}

	// Get latest timestamp
	latestTimestamp := timestamps[len(timestamps)-1]

	// Format the message
	message := fmt.Sprintf("**Daily Validator Set Update Analysis**\n\n"+
		"**Latest Update:** %s\n"+
		"**Analysis Period:** Last 7 days\n"+
		"**Total Updates:** %d\n"+
		"**Average Frequency:** %s\n"+
		"**Median Frequency:** %s\n"+
		"**Node:** %s",
		latestTimestamp.Format("2006-01-02 15:04:05 UTC"),
		len(timestamps),
		formatDuration(averageDuration),
		formatDuration(medianDuration),
		nodeName)

	// Send Discord alert
	configMutex.RLock()
	eventType, exists := eventTypeMap["valset_update_analysis"]
	configMutex.RUnlock()

	if !exists {
		log.Printf("No valset_update_analysis event type configured, skipping alert")
		return
	}

	discordNotifier := utils.NewDiscordNotifier(eventType.WebhookURL)
	if err := discordNotifier.SendAlert(message); err != nil {
		log.Printf("Error sending validator set analysis Discord alert: %v", err)
	} else {
		log.Printf("Sent daily validator set analysis Discord alert")
	}
}

// Helper function to format duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0f seconds", d.Seconds())
	} else if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		days := int(d.Hours()) / 24
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%dd %dh", days, hours)
	}
}

func main() {
	// Parse command line flags
	flag.StringVar(&rpcURL, "rpc-url", DefaultRpcURL, "RPC URL (default: 127.0.0.1:26657)")
	flag.StringVar(&configFilePath, "config", "", "Path to config file")
	flag.StringVar(&nodeName, "node", "", "Name of the node being monitored")
	flag.BoolVar(&isTimestampAnalyzer, "timestamp-analyzer", false, "Enable analyzer of validator set update timestamps")
	flag.DurationVar(&blockTimeThreshold, "block-time-threshold", 0, "Block time threshold (e.g. 5m, 1h). If not set, block time monitoring is disabled.")
	flag.Parse()

	// Validate required parameters
	if configFilePath == "" || nodeName == "" {
		log.Fatal("Usage: go run ./scripts/async-monitors/async-monitor-events.go -rpc-url=<rpc_url> -config=<config_file_path> -node=<node_name>")
	}

	// Initialize Current_Total_Reporter_Power with a default value
	reporterPowerMutex.Lock()
	Current_Total_Reporter_Power = 100 // Default value until first update
	reporterPowerMutex.Unlock()

	lastBlockHeight = 0

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
	wg.Add(2)

	go MonitorBlockEvents(ctx, &wg)
	go updateTotalReporterPower(ctx)

	// Start validator set update analyzer if enabled
	if isTimestampAnalyzer {
		wg.Add(1)
		go analyzeValidatorSetUpdates(ctx)
	}

	wg.Wait()
	log.Println("Shutdown complete")
}
