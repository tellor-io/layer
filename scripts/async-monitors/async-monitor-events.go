package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
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
	AlertCooldownPeriod      = 2 * time.Hour
	AlertWindowPeriod        = 10 * time.Minute
	MaxAlertsInWindow        = 10
	PowerThreshold           = 2.0 / 3.0
	DefaultRpcURL            = "127.0.0.1:26657"
	DefaultSwaggerURL        = "http://127.0.0.1:1317"
	MaxReconnectAttempts     = 5
	BlockQueryInterval       = 1 * time.Second
	MinAmountThresholdTRB    = 500
	MinAmountThresholdBase   = 500_000_000 // 500 TRB in base units (500 * 1_000_000)
	PowerPercentageThreshold = 0.31        // 31%
)

var (
	// MinAmountThresholdBigInt is 500 TRB in base units as a big.Int for safe comparisons
	MinAmountThresholdBigInt = big.NewInt(MinAmountThresholdBase)
)

var (
	configMutex                  sync.RWMutex
	Current_Total_Reporter_Power uint64
	reporterPowerMutex           sync.RWMutex
	// Rate limiting variables
	AGGREGATE_REPORT_EVENT_TYPE = "aggregate_report"
	AGGREGATE_POWER_ATTR_KEY    = "aggregate_power"
	DELEGATE_EVENT_TYPE         = "delegate"
	TIP_WITHDRAWN_EVENT_TYPE    = "tip_withdrawn"
	aggregateAlertCount         int
	aggregateAlertTimestamps    []time.Time
	aggregateAlertMutex         sync.RWMutex
	aggregateAlertCooldown      time.Time
	// Map to store event types we're interested in
	eventTypeMap map[string]ConfigType
	// Supported query IDs map for asset pair lookups
	supportedQueryIDsMap *SupportedQueryIDsMap
	queryIDsMutex        sync.RWMutex
	// HTTP client for RPC requests (shared across goroutines)
	globalHTTPClient *HTTPClient
	httpClientMutex  sync.RWMutex
	// Command line parameters
	rpcURL                   string
	swaggerURL               string
	configFilePath           string
	supportedQueryIDsMapPath string
	nodeName                 string
	blockTimeThreshold       time.Duration
	previousBlockTime        time.Time
	blockTimeMutex           sync.RWMutex
	lastBlockHeight          uint64
	isTimestampAnalyzer      bool
	currentBlockHeight       uint64
	blockHeightMutex         sync.RWMutex
	// Track validators that have been alerted for exceeding power threshold
	alertedValidators      map[string]bool
	alertedValidatorsMutex sync.RWMutex
)

type Params struct {
	Query string `json:"query"`
}

type ConfigType struct {
	AlertName   string    `yaml:"alert_name"`
	AlertType   string    `yaml:"alert_type"`
	Query       string    `yaml:"query"`
	WebhookURL  string    `yaml:"webhook_url"`
	LastAlerted time.Time `yaml:"last_alerted"`
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
		AppHash string `json:"app_hash"`
	} `json:"result"`
}

type EventConfig struct {
	EventTypes []ConfigType `yaml:"event_types"`
}

// SupportedQueryIDsMap represents the structure of the supported_query_ids_map.json file
type SupportedQueryIDsMap struct {
	QueryIDToAssetPairMap   map[string]string `json:"queryIdToAssetPairMap"`
	QueryDataToAssetPairMap map[string]string `json:"queryDataToAssetPairMap"`
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

	resp, err := h.client.Post(h.baseURL, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w, Request: %v", err, request)
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

func (h *HTTPClient) getValidators(height uint64) (*ValidatorResponse, error) {
	var params interface{}
	if height > 0 {
		params = map[string]interface{}{
			"height": fmt.Sprintf("%d", height),
		}
	} else {
		params = nil // nil means latest validators
	}

	body, err := h.makeRPCRequest("validators", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get validators: %w", err)
	}

	var resp ValidatorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal validators response: %w", err)
	}

	return &resp, nil
}

func CheckForValidatorsWithMoreThan31PercentPower(validatorData ValidatorResponse) error {
	// First, calculate total power
	var totalPower int64
	validatorPowers := make(map[string]int64) // Map validator address to power for quick lookup

	for _, validator := range validatorData.Result.Validators {
		power, err := strconv.ParseInt(validator.VotingPower, 10, 64)
		if err != nil {
			log.Printf("Error parsing voting power for validator %s: %v\n", validator.Address, err)
			continue
		}
		totalPower += power
		validatorPowers[validator.Address] = power
	}

	if totalPower == 0 {
		return fmt.Errorf("total power is zero, cannot calculate percentages")
	}

	// Early return if the first (highest power) validator doesn't exceed threshold
	// Validators are already sorted by the RPC response
	if len(validatorData.Result.Validators) == 0 {
		return nil
	}

	firstAddr := validatorData.Result.Validators[0].Address
	firstPower, ok := validatorPowers[firstAddr]
	if !ok {
		return fmt.Errorf("failed to get voting power for first validator %s", firstAddr)
	}
	firstValidatorPowerPercent := float64(firstPower) / float64(totalPower)
	if firstValidatorPowerPercent <= PowerPercentageThreshold {
		// No validators exceed threshold, but check if any previously alerted validators dropped below
		configMutex.RLock()
		eventType, hasWebhook := eventTypeMap["validator-power-alert"]
		configMutex.RUnlock()

		alertedValidatorsMutex.Lock()
		for address := range alertedValidators {
			// Check if this validator is still in the current set
			currentPower, found := validatorPowers[address]
			if !found {
				// Validator no longer in set, remove from tracking
				delete(alertedValidators, address)
			} else {
				currentPowerPercent := float64(currentPower) / float64(totalPower)
				if currentPowerPercent <= PowerPercentageThreshold {
					// Validator dropped below threshold - send alert
					message := "**Alert: Validator Dropped Below Power Threshold**\n"
					message += fmt.Sprintf("Validator Address: %s\n", address)
					message += fmt.Sprintf("Current Voting Power: %d\n", currentPower)
					message += fmt.Sprintf("Total Power: %d\n", totalPower)
					message += fmt.Sprintf("Current Power Percentage: %.2f%%\n", currentPowerPercent*100)
					message += fmt.Sprintf("Threshold: %.2f%%\n", PowerPercentageThreshold*100)

					if hasWebhook {
						discordNotifier := utils.NewDiscordNotifier(eventType.WebhookURL)
						if err := discordNotifier.SendAlert(message); err != nil {
							log.Printf("Error sending Discord alert for validator %s dropping below threshold: %v\n", address, err)
						} else {
							log.Printf("Sent Discord alert for validator dropping below threshold: %s (%.2f%%)\n", address, currentPowerPercent*100)
						}
					} else {
						log.Printf("Validator %s dropped below threshold (%.2f%% <= %.2f%%) but no webhook configured\n", address, currentPowerPercent*100, PowerPercentageThreshold*100)
					}

					// Remove from tracking
					delete(alertedValidators, address)
				}
			}
		}
		alertedValidatorsMutex.Unlock()
		return nil
	}

	// Get webhook URL for alerts
	configMutex.RLock()
	eventType, hasWebhook := eventTypeMap["validator-power-alert"]
	configMutex.RUnlock()

	if !hasWebhook {
		log.Printf("No webhook configured for validator-power-alert, skipping alerts\n")
	}

	// Track current validators that exceed threshold
	currentExceedingValidators := make(map[string]bool)

	// Check each validator (they're already sorted, so we can stop early if needed)
	for _, validator := range validatorData.Result.Validators {
		power, ok := validatorPowers[validator.Address]
		if !ok {
			// Should be rare (only if we failed to parse this validator's voting power earlier)
			log.Printf("Skipping validator %s: missing parsed voting power\n", validator.Address)
			continue
		}
		powerPercent := float64(power) / float64(totalPower)

		if powerPercent > PowerPercentageThreshold {
			currentExceedingValidators[validator.Address] = true

			// Check if this validator was previously alerted
			alertedValidatorsMutex.RLock()
			wasAlerted := alertedValidators[validator.Address]
			alertedValidatorsMutex.RUnlock()

			if !wasAlerted {
				// New validator exceeding threshold - send alert
				message := "**Alert: Validator Exceeding Power Threshold**\n"
				message += fmt.Sprintf("Validator Address: %s\n", validator.Address)
				message += fmt.Sprintf("Voting Power: %d\n", power)
				message += fmt.Sprintf("Total Power: %d\n", totalPower)
				message += fmt.Sprintf("Power Percentage: %.2f%%\n", powerPercent*100)
				message += fmt.Sprintf("Threshold: %.2f%%\n", PowerPercentageThreshold*100)

				if hasWebhook {
					discordNotifier := utils.NewDiscordNotifier(eventType.WebhookURL)
					if err := discordNotifier.SendAlert(message); err != nil {
						log.Printf("Error sending Discord alert for validator %s: %v\n", validator.Address, err)
					} else {
						log.Printf("Sent Discord alert for validator exceeding threshold: %s (%.2f%%)\n", validator.Address, powerPercent*100)
						// Mark as alerted
						alertedValidatorsMutex.Lock()
						if alertedValidators == nil {
							alertedValidators = make(map[string]bool)
						}
						alertedValidators[validator.Address] = true
						alertedValidatorsMutex.Unlock()
					}
				} else {
					log.Printf("Validator %s exceeds threshold (%.2f%% > %.2f%%) but no webhook configured\n", validator.Address, powerPercent*100, PowerPercentageThreshold*100)
					// Still track it even without webhook
					alertedValidatorsMutex.Lock()
					if alertedValidators == nil {
						alertedValidators = make(map[string]bool)
					}
					alertedValidators[validator.Address] = true
					alertedValidatorsMutex.Unlock()
				}
			}
		} else {
			// This validator is below threshold, so all subsequent ones will be too (already sorted)
			break
		}
	}

	// Check for validators that dropped below threshold
	alertedValidatorsMutex.Lock()
	for address := range alertedValidators {
		if !currentExceedingValidators[address] {
			// Get the validator's current power from our map
			currentPower, found := validatorPowers[address]
			if !found {
				// Validator no longer in set, remove from tracking
				delete(alertedValidators, address)
				continue
			}
			currentPowerPercent := float64(currentPower) / float64(totalPower)

			// Send alert for dropping below threshold
			message := "**Alert: Validator Dropped Below Power Threshold**\n"
			message += fmt.Sprintf("Validator Address: %s\n", address)
			message += fmt.Sprintf("Current Voting Power: %d\n", currentPower)
			message += fmt.Sprintf("Total Power: %d\n", totalPower)
			message += fmt.Sprintf("Current Power Percentage: %.2f%%\n", currentPowerPercent*100)
			message += fmt.Sprintf("Threshold: %.2f%%\n", PowerPercentageThreshold*100)

			if hasWebhook {
				discordNotifier := utils.NewDiscordNotifier(eventType.WebhookURL)
				if err := discordNotifier.SendAlert(message); err != nil {
					log.Printf("Error sending Discord alert for validator %s dropping below threshold: %v\n", address, err)
				} else {
					log.Printf("Sent Discord alert for validator dropping below threshold: %s (%.2f%%)\n", address, currentPowerPercent*100)
				}
			} else {
				log.Printf("Validator %s dropped below threshold (%.2f%% <= %.2f%%) but no webhook configured\n", address, currentPowerPercent*100, PowerPercentageThreshold*100)
			}

			// Remove from tracking
			delete(alertedValidators, address)
		}
	}
	alertedValidatorsMutex.Unlock()

	return nil
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
		// Preserve existing LastAlerted value if it exists
		if existingConfig, exists := eventTypeMap[et.AlertType]; exists {
			et.LastAlerted = existingConfig.LastAlerted
		}
		newEventTypeMap[et.AlertType] = et
		log.Printf("Loaded event type: %s (%s)\n", et.AlertName, et.AlertType)
	}

	configMutex.Lock()
	eventTypeMap = newEventTypeMap
	configMutex.Unlock()
	return nil
}

// loadSupportedQueryIDsMap loads the supported query IDs map from the JSON file
func loadSupportedQueryIDsMap() error {
	if supportedQueryIDsMapPath == "" {
		return fmt.Errorf("supported query IDs map file path not provided")
	}

	data, err := os.ReadFile(supportedQueryIDsMapPath)
	if err != nil {
		return fmt.Errorf("error reading supported query IDs map file at %s: %w", supportedQueryIDsMapPath, err)
	}

	var newQueryIDsMap SupportedQueryIDsMap
	if err := json.Unmarshal(data, &newQueryIDsMap); err != nil {
		return fmt.Errorf("error parsing supported query IDs map file at %s: %w", supportedQueryIDsMapPath, err)
	}

	queryIDsMutex.Lock()
	supportedQueryIDsMap = &newQueryIDsMap
	queryIDsMutex.Unlock()

	log.Printf("Loaded supported query IDs map from %s with %d query ID mappings and %d query data mappings\n",
		supportedQueryIDsMapPath, len(newQueryIDsMap.QueryIDToAssetPairMap), len(newQueryIDsMap.QueryDataToAssetPairMap))
	return nil
}

// getAssetPairFromQueryID returns the asset pair for a given query ID, or empty string if not found
func getAssetPairFromQueryID(queryID string) string {
	queryIDsMutex.RLock()
	defer queryIDsMutex.RUnlock()

	if supportedQueryIDsMap == nil {
		log.Println("supportedQueryIDsMap is nil")
		return ""
	}

	if assetPair, exists := supportedQueryIDsMap.QueryIDToAssetPairMap[queryID]; exists {
		return assetPair
	}

	// If not found, return empty string
	log.Println("assetPair not found")
	return ""
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
				log.Printf("Error reloading config: %v\n", err)
			}
		}
	}
}

func handleAggregateReport(event Event, eventType ConfigType) {
	aggregateAlertMutex.Lock()
	defer aggregateAlertMutex.Unlock()

	// Check if we're in cooldown
	if time.Now().Before(aggregateAlertCooldown) {
		log.Printf("In cooldown, skipping aggregate report\n")
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
		if event.Attributes[j].Key == AGGREGATE_POWER_ATTR_KEY {
			if aggregatePower, err := strconv.ParseUint(event.Attributes[j].Value, 10, 64); err == nil {
				if float64(aggregatePower) < float64(currentPower)*2/3 {
					log.Printf("Aggregate power is less than 2/3 of current power: %d\n", currentPower)
					// Check if we've hit the alert limit
					if aggregateAlertCount >= 10 {
						// Send final alert and start cooldown
						message := fmt.Sprintf("**Rate Limit Reached: %s**\nToo many alerts in the last 10 minutes. Alerts will be paused for 2 hours. Please check on reporters and see what is going on\n", eventType.AlertName)
						for _, attr := range event.Attributes {
							message += fmt.Sprintf("%s: %s\n", attr.Key, attr.Value)
							if attr.Key == "query_id" {
								assetPair := getAssetPairFromQueryID(attr.Value)
								if assetPair != "" {
									message += fmt.Sprintf("Asset Pair: %s\n", assetPair)
								}
							}
						}

						discordNotifier := utils.NewDiscordNotifier(eventType.WebhookURL)
						if err := discordNotifier.SendAlert(message); err != nil {
							log.Printf("Error sending final Discord alert: %v\n", err)
						} else {
							log.Printf("Sent final Discord alert and starting cooldown\n")
							eventConfig := eventTypeMap[eventType.AlertType]
							eventConfig.LastAlerted = now
							eventTypeMap[eventConfig.AlertType] = eventConfig
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
						log.Printf("Error sending Discord alert for event %s: %v\n", event.Type, err)
					} else {
						log.Printf("Sent Discord alert for event: %s\n", event.Type)
						// Add timestamp for this alert
						aggregateAlertTimestamps = append(aggregateAlertTimestamps, now)
						aggregateAlertCount++
						eventConfig := eventTypeMap[eventType.AlertType]
						eventConfig.LastAlerted = now
						eventTypeMap[eventConfig.AlertType] = eventConfig
					}
				}
			} else {
				log.Printf("Error parsing aggregate power: %v\n", err)
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

	// Load supported query IDs map
	if err := loadSupportedQueryIDsMap(); err != nil {
		log.Printf("Error loading supported query IDs map: %v", err)
		return
	}

	client := NewHTTPClient(rpcURL)

	// Store global HTTP client for use in other goroutines
	httpClientMutex.Lock()
	globalHTTPClient = client
	httpClientMutex.Unlock()

	// Get initial block height
	initialHeight, err := client.getLatestBlockHeight()
	if err != nil {
		log.Printf("Failed to get initial block height: %v", err)
		panic(err)
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
			log.Printf("Warning: Block %d has empty time field, skipping block time analysis\n", height)
		} else {
			currentBlockTime, err := time.Parse(time.RFC3339Nano, blockTimeStr)
			if err != nil {
				return fmt.Errorf("unable to parse block time: %w", err)
			}

			// Always log the extracted time field when block time threshold is set
			log.Printf("Block %d time: %s\n", height, blockTimeStr)

			blockTimeMutex.RLock()
			prevTime := previousBlockTime
			blockTimeMutex.RUnlock()

			// Skip first block after startup
			if !prevTime.IsZero() {
				timeDiff := currentBlockTime.Sub(prevTime)
				blockDiff := height - lastBlockHeight
				if blockDiff > 0 {
					normalizedTimeDiff := time.Duration(float64(timeDiff) / float64(blockDiff))
					log.Println("Normalized time per block: ", normalizedTimeDiff.String())
					if normalizedTimeDiff > blockTimeThreshold {
						message := fmt.Sprintf("**Alert: Abnormally Long Block Time**\nNode: %s\nTime between blocks: %v\nNormalized time per block: %v\nThreshold: %v\nPrevious block time: %v\nCurrent block time: %v",
							nodeName, timeDiff, normalizedTimeDiff, blockTimeThreshold, prevTime, currentBlockTime)

						if eventType, exists := eventTypeMap["block-time-alert"]; exists {
							discordNotifier := utils.NewDiscordNotifier(eventType.WebhookURL)
							if err := discordNotifier.SendAlert(message); err != nil {
								log.Printf("Error sending block time Discord alert: %v\n", err)
							} else {
								log.Printf("Sent block time Discord alert\n")
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
				if event.Type != AGGREGATE_REPORT_EVENT_TYPE {
					log.Printf("Found configured event: %s\n", event.Type)
				}
			}
		}

		if configuredEventCount > 0 {
			log.Printf("Found %d configured events in block %d\n", configuredEventCount, height)
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
					if event.Type != AGGREGATE_REPORT_EVENT_TYPE {
						log.Printf("Found configured event in tx %d: %s\n", i, event.Type)
					}
				}
			}
		}

		if txConfiguredEventCount > 0 {
			log.Printf("Found %d configured events in transactions for block %d\n", txConfiguredEventCount, height)
		}

		go processTransactionEvents(resultsResponse.Result.TxsResults, fmt.Sprintf("%d", height))
	}

	return nil
}

func processBlockEvents(events []Event, height string) {
	for _, event := range events {
		if eventType, exists := eventTypeMap[event.Type]; exists {
			log.Printf("Event at %s: %s\n", height, event.Type)
			handleEvent(event, eventType)
		}
	}
}

func processTransactionEvents(txResults []TxResult, height string) {
	for _, txResult := range txResults {
		for _, event := range txResult.Events {
			if eventType, exists := eventTypeMap[event.Type]; exists {
				log.Printf("Event at %s: %s\n", height, event.Type)
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

// SwaggerValidator represents a single validator from the Swagger API response
type SwaggerValidator struct {
	OperatorAddress string `json:"operator_address"`
	Tokens          string `json:"tokens"`
	Status          string `json:"status"`
}

// SwaggerValidatorsResponse represents the response from the Swagger API validators endpoint
type SwaggerValidatorsResponse struct {
	Validators []SwaggerValidator `json:"validators"`
	Pagination interface{}        `json:"pagination,omitempty"`
}

// ValidatorPowerInfo contains validator power and total bonded power information
type ValidatorPowerInfo struct {
	ValidatorPower *big.Int
	TotalPower     *big.Int
}

// getValidatorPowerForAddress gets the voting power (tokens) for a specific validator operator address
// and calculates total bonded power using the Swagger API endpoint /cosmos/staking/v1beta1/validators
// Returns both the validator's power and the total bonded power
func getValidatorPowerForAddress(operatorAddress string) (*ValidatorPowerInfo, error) {
	if swaggerURL == "" {
		return nil, fmt.Errorf("swagger URL not configured")
	}

	apiURL := fmt.Sprintf("%s/cosmos/staking/v1beta1/validators", swaggerURL)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to query validators from Swagger API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Swagger API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var validatorsResp SwaggerValidatorsResponse
	if err := json.Unmarshal(body, &validatorsResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal validators response: %w", err)
	}

	// Find the matching validator and calculate total bonded power
	var validatorPower *big.Int
	totalPower := big.NewInt(0)
	var foundValidator bool

	for _, validator := range validatorsResp.Validators {
		// Only count bonded validators for total power
		if validator.Status == "BOND_STATUS_BONDED" {
			tokens, ok := new(big.Int).SetString(validator.Tokens, 10)
			if !ok {
				log.Printf("Warning: failed to parse tokens for validator %s: invalid format\n", validator.OperatorAddress)
				continue
			}
			totalPower.Add(totalPower, tokens)

			// Check if this is the validator we're looking for
			if validator.OperatorAddress == operatorAddress {
				validatorPower = new(big.Int).Set(tokens)
				foundValidator = true
			}
		}
	}

	if !foundValidator {
		return nil, fmt.Errorf("validator not found with operator address: %s", operatorAddress)
	}

	if totalPower.Sign() == 0 {
		return nil, fmt.Errorf("total bonded power is zero")
	}

	return &ValidatorPowerInfo{
		ValidatorPower: validatorPower,
		TotalPower:     totalPower,
	}, nil
}

// parseAmountFromString parses an amount string (e.g., "500000000loya" or "500000000") and returns the numeric value as big.Int
func parseAmountFromString(amountStr string) (*big.Int, error) {
	// Remove denom if present (e.g., "500000000loya" -> "500000000")
	amountStr = strings.TrimSuffix(amountStr, "loya")
	amountStr = strings.TrimSpace(amountStr)

	amount, ok := new(big.Int).SetString(amountStr, 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse amount: invalid format")
	}
	return amount, nil
}

// handleDelegateOrTipWithdrawn handles delegate and tip_withdrawn events with special conditions
func handleDelegateOrTipWithdrawn(event Event, eventType ConfigType) {
	var validatorAddress string
	var delegatorOrSelectorAddress string
	var amount *big.Int
	var foundAmount bool
	var foundValidatorAddress bool

	// Extract attributes based on event type
	// delegate event: "validator", "delegator", "amount", "new_shares"
	// tip_withdrawn event: "validator", "selector", "amount", "shares"
	for _, attr := range event.Attributes {
		if attr.Key == "amount" {
			var err error
			amount, err = parseAmountFromString(attr.Value)
			if err != nil {
				log.Printf("Error parsing amount for %s event: %v\n", event.Type, err)
				return
			}
			foundAmount = true
		}
		// Validator address (present in both event types)
		if attr.Key == "validator" {
			validatorAddress = attr.Value
			foundValidatorAddress = true
		}
		// Delegator address (for delegate events)
		if attr.Key == "delegator" {
			delegatorOrSelectorAddress = attr.Value
		}
		// Selector address (for tip_withdrawn events)
		if attr.Key == "selector" {
			delegatorOrSelectorAddress = attr.Value
		}
	}

	if !foundAmount {
		log.Printf("No amount found in %s event\n", event.Type)
		return
	}

	if !foundValidatorAddress {
		log.Printf("No validator address found in %s event\n", event.Type)
		return
	}

	// Check condition 1: Amount > 500 TRB
	amountExceedsThreshold := amount.Cmp(MinAmountThresholdBigInt) >= 0

	// Check condition 2: Power percentage > 31%
	var powerExceedsThreshold bool
	var powerPercentage float64
	var validatorPower *big.Int
	var totalPower *big.Int

	// Get validator power and total power from Swagger API
	powerInfo, err := getValidatorPowerForAddress(validatorAddress)
	if err != nil {
		log.Printf("Error getting validator power for address %s: %v\n", validatorAddress, err)
		// If we can't get validator power, only check amount threshold
		if !amountExceedsThreshold {
			log.Printf("Amount %s does not exceed threshold %s, skipping alert\n", amount.String(), MinAmountThresholdBigInt.String())
			return
		}
	} else {
		validatorPower = powerInfo.ValidatorPower
		totalPower = powerInfo.TotalPower

		// Calculate power percentage using big.Int for precision
		// percentage = (validatorPower * 10000) / totalPower (multiply by 10000 to get percentage with 2 decimal places)
		// Then compare: (validatorPower * 10000) / totalPower > (31 * 10000) / 100
		// Which simplifies to: (validatorPower * 100) / totalPower > 31
		// To avoid floating point, we compare: validatorPower * 100 > totalPower * 31
		validatorPowerTimes100 := new(big.Int).Mul(validatorPower, big.NewInt(100))
		totalPowerTimes31 := new(big.Int).Mul(totalPower, big.NewInt(31))
		powerExceedsThreshold = validatorPowerTimes100.Cmp(totalPowerTimes31) > 0

		// Calculate power percentage for logging (using float64 for display)
		validatorPowerFloat := new(big.Float).SetInt(validatorPower)
		totalPowerFloat := new(big.Float).SetInt(totalPower)
		powerPercentageFloat := new(big.Float).Quo(validatorPowerFloat, totalPowerFloat)
		powerPercentage, _ = powerPercentageFloat.Float64()
	}

	// Only send alert if either condition is met
	if !amountExceedsThreshold && !powerExceedsThreshold {
		log.Printf("Neither condition met for %s event: amount=%s (threshold=%s), power=%.2f%% (threshold=%.2f%%)\n",
			event.Type, amount.String(), MinAmountThresholdBigInt.String(), powerPercentage*100, PowerPercentageThreshold*100)
		return
	}

	// Build alert message
	message := fmt.Sprintf("**Event Alert: %s**\n", eventType.AlertName)
	for _, attr := range event.Attributes {
		message += fmt.Sprintf("%s: %s\n", attr.Key, attr.Value)
	}

	// Add condition details
	message += "\n**Alert Conditions Met:**\n"
	if amountExceedsThreshold {
		// Convert big.Int to float64 for TRB calculation
		amountFloat := new(big.Float).SetInt(amount)
		trbFloat := new(big.Float).Quo(amountFloat, big.NewFloat(1_000_000.0))
		trbValue, _ := trbFloat.Float64()
		message += fmt.Sprintf("✓ Amount (%s base units = %.2f TRB) exceeds threshold (%s base units = %d TRB)\n",
			amount.String(), trbValue, MinAmountThresholdBigInt.String(), MinAmountThresholdTRB)
	}
	if powerExceedsThreshold {
		message += fmt.Sprintf("✓ Validator power percentage (%.2f%%) exceeds threshold (%.2f%%)\n",
			powerPercentage*100, PowerPercentageThreshold*100)
		message += fmt.Sprintf("  Validator Address: %s\n", validatorAddress)
		if delegatorOrSelectorAddress != "" {
			if event.Type == DELEGATE_EVENT_TYPE {
				message += fmt.Sprintf("  Delegator Address: %s\n", delegatorOrSelectorAddress)
			} else {
				message += fmt.Sprintf("  Selector Address: %s\n", delegatorOrSelectorAddress)
			}
		}
		message += fmt.Sprintf("  Validator Power: %s\n", validatorPower.String())
		message += fmt.Sprintf("  Total Power: %s\n", totalPower.String())
	}

	discordNotifier := utils.NewDiscordNotifier(eventType.WebhookURL)
	if err := discordNotifier.SendAlert(message); err != nil {
		log.Printf("Error sending Discord alert for event %s: %v\n", event.Type, err)
	} else {
		log.Printf("Sent Discord alert for event: %s\n", event.Type)
		configMutex.Lock()
		eventConfig := eventTypeMap[eventType.AlertType]
		eventConfig.LastAlerted = time.Now()
		eventTypeMap[eventType.AlertType] = eventConfig
		configMutex.Unlock()
	}
}

// Add a helper function to handle events
func handleEvent(event Event, eventType ConfigType) {
	if event.Type == AGGREGATE_REPORT_EVENT_TYPE {
		handleAggregateReport(event, eventType)
	} else if event.Type == DELEGATE_EVENT_TYPE || event.Type == TIP_WITHDRAWN_EVENT_TYPE {
		handleDelegateOrTipWithdrawn(event, eventType)
	} else {
		message := fmt.Sprintf("**Event Alert: %s**\n", eventType.AlertName)
		for _, attr := range event.Attributes {
			message += fmt.Sprintf("%s: %s\n", attr.Key, attr.Value)
			if attr.Key == "query_id" {
				// Try to get the asset pair for this query ID
				assetPair := getAssetPairFromQueryID(attr.Value)
				if assetPair != "" {
					message += fmt.Sprintf("Asset Pair: %s\n", assetPair)
				} else {
					message += "Asset Pair: Unknown\n"
				}
			}
		}

		discordNotifier := utils.NewDiscordNotifier(eventType.WebhookURL)
		if err := discordNotifier.SendAlert(message); err != nil {
			log.Printf("error is being thrown in handleEvent")
			log.Printf("Error sending Discord alert for event %s: %v\n", event.Type, err)
		} else {
			log.Printf("Sent Discord alert for event: %s\n", event.Type)
			eventConfig := eventTypeMap[eventType.AlertType]
			eventConfig.LastAlerted = time.Now()
			eventTypeMap[eventConfig.AlertType] = eventConfig
		}
	}

	if event.Type == "new_bridge_validator_set" {
		for _, attr := range event.Attributes {
			if attr.Key == "timestamp" {
				if err := writeTimestampToCSV(attr.Value); err != nil {
					log.Printf("Error writing timestamp to CSV: %v\n", err)
				} else {
					log.Printf("Successfully wrote timestamp %s to CSV\n", attr.Value)
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
			validatorResp, err := client.getValidators(0) // 0 means latest validators
			if err != nil {
				log.Printf("Error querying validators: %v\n", err)
				// Implement exponential backoff
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
					log.Printf("Error parsing voting power: %v\n", err)
					continue
				}
				totalPower += power
			}
			log.Printf("Total power: %d\n", totalPower)

			reporterPowerMutex.Lock()
			Current_Total_Reporter_Power = uint64(totalPower)
			reporterPowerMutex.Unlock()

			log.Printf("Updated total reporter power: %d\n", totalPower)

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
	log.Printf("Running analysis")
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
			// Try Unix timestamp (assuming milliseconds)
			if unixTime, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
				timestamp = time.Unix(unixTime/1000, (unixTime%1000)*1000000)
			} else {
				log.Printf("Error parsing timestamp %s: %v", timestampStr, parseErr)
				continue
			}
		}
		fourteenDaysAgo := time.Now().Add(-14 * 24 * time.Hour)
		if timestamp.After(fourteenDaysAgo) {
			timestamps = append(timestamps, timestamp)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading CSV file: %v", err)
		return
	}

	// Check if we have any timestamps in the last 2 weeks
	if len(timestamps) == 0 {
		log.Printf("No timestamps found in the last 2 weeks - this indicates an issue")
		// Send alert about no recent timestamps
		message := fmt.Sprintf("**Daily Validator Set Update Analysis**\n\n"+
			"**⚠️ ALERT: No validator set updates found in the last 2 weeks**\n"+
			"**Analysis Period:** Last 14 days\n"+
			"**Total Updates:** 0\n"+
			"**Status:** No recent activity detected\n"+
			"**Node:** %s",
			nodeName)

		configMutex.RLock()
		eventType, exists := eventTypeMap["valset_update_analysis"]
		configMutex.RUnlock()

		if exists {
			discordNotifier := utils.NewDiscordNotifier(eventType.WebhookURL)
			if err := discordNotifier.SendAlert(message); err != nil {
				log.Printf("Error sending no-timestamps Discord alert: %v", err)
			} else {
				log.Printf("Sent no-timestamps Discord alert")
			}
		}
		return
	}

	if len(timestamps) < 2 {
		log.Printf("Only %d timestamp(s) found in the last 2 weeks (need at least 2 for frequency analysis)", len(timestamps))
		// Send alert about insufficient data
		message := fmt.Sprintf("**Daily Validator Set Update Analysis**\n\n"+
			"**⚠️ ALERT: Insufficient data for frequency analysis**\n"+
			"**Analysis Period:** Last 14 days\n"+
			"**Total Updates:** %d\n"+
			"**Status:** Only %d update(s) found - cannot calculate frequency\n"+
			"**Node:** %s",
			len(timestamps), len(timestamps), nodeName)

		configMutex.RLock()
		eventType, exists := eventTypeMap["valset_update_analysis"]
		configMutex.RUnlock()

		if exists {
			discordNotifier := utils.NewDiscordNotifier(eventType.WebhookURL)
			if err := discordNotifier.SendAlert(message); err != nil {
				log.Printf("Error sending insufficient-data Discord alert: %v", err)
			} else {
				log.Printf("Sent insufficient-data Discord alert")
			}
		}
		return
	}

	// Sort timestamps
	sort.Slice(timestamps, func(i, j int) bool {
		return timestamps[i].Before(timestamps[j])
	})

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
		"**Analysis Period:** Last 14 days\n"+
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
		eventConfig := eventTypeMap[eventType.AlertType]
		eventConfig.LastAlerted = time.Now()
		eventTypeMap[eventConfig.AlertType] = eventConfig
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

// Add a helper function to check event type heartbeats
func checkEventTypeHeartbeats(ctx context.Context) {
	defer recoverAndAlert("checkEventTypeHeartbeats")

	// Run check once a day at 10 AM
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Calculate time until next 10 AM
	now := time.Now()
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
	if now.After(nextRun) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	// Wait until 10 AM
	time.Sleep(nextRun.Sub(now))

	// Run initial check
	runHeartbeatCheck()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runHeartbeatCheck()
		}
	}
}

func runHeartbeatCheck() {
	configMutex.RLock()
	eventTypes := make(map[string]ConfigType)
	for k, v := range eventTypeMap {
		eventTypes[k] = v
	}
	configMutex.RUnlock()

	sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)
	var inactiveEventTypes []string

	// Check each event type
	for alertType, config := range eventTypes {
		// Skip if LastAlerted is zero (never alerted) or if it's been more than 7 days
		if config.LastAlerted.IsZero() || config.LastAlerted.Before(sevenDaysAgo) {
			inactiveEventTypes = append(inactiveEventTypes, alertType)
		}
	}

	// Send individual heartbeat alerts to each inactive event type's channel
	for _, alertType := range inactiveEventTypes {
		config := eventTypes[alertType]

		// Skip if no webhook URL is configured
		if config.WebhookURL == "" {
			log.Printf("No webhook URL configured for event type %s, skipping heartbeat", alertType)
			continue
		}

		lastAlerted := "Never"
		if !config.LastAlerted.IsZero() {
			lastAlerted = config.LastAlerted.Format("2006-01-02 15:04:05 UTC")
		}

		message := fmt.Sprintf("**Heartbeat Alert: %s**\n\n"+
			"This event type has not been triggered in the last 7 days.\n"+
			"**Node:** %s\n"+
			"**Event Type:** %s\n"+
			"**Last Alerted:** %s\n\n"+
			"This is a heartbeat alert to confirm that this event type is still being monitored.",
			config.AlertName, nodeName, alertType, lastAlerted)

		discordNotifier := utils.NewDiscordNotifier(config.WebhookURL)
		if err := discordNotifier.SendAlert(message); err != nil {
			log.Printf("Error sending heartbeat Discord alert for %s: %v", alertType, err)
		} else {
			log.Printf("Sent heartbeat alert for event type: %s", alertType)
			// Update the LastAlerted timestamp for this event type
			configMutex.Lock()
			if eventConfig, exists := eventTypeMap[alertType]; exists {
				eventConfig.LastAlerted = time.Now()
				eventTypeMap[alertType] = eventConfig
			}
			configMutex.Unlock()
		}
	}

	if len(inactiveEventTypes) > 0 {
		log.Printf("Sent heartbeat alerts for %d inactive event types", len(inactiveEventTypes))
	} else {
		log.Printf("All event types have been alerted within the last 7 days")
	}
}

func main() {
	// Parse command line flags
	flag.StringVar(&rpcURL, "rpc-url", DefaultRpcURL, "RPC URL (default: 127.0.0.1:26657)")
	flag.StringVar(&swaggerURL, "swagger-url", DefaultSwaggerURL, "Swagger API URL (default: http://127.0.0.1:1317)")
	flag.StringVar(&configFilePath, "config", "", "Path to config file")
	flag.StringVar(&supportedQueryIDsMapPath, "query-ids-map", "", "Path to supported query IDs map JSON file")
	flag.StringVar(&nodeName, "node", "", "Name of the node being monitored")
	flag.BoolVar(&isTimestampAnalyzer, "timestamp-analyzer", false, "Enable analyzer of validator set update timestamps")
	flag.DurationVar(&blockTimeThreshold, "block-time-threshold", 0, "Block time threshold (e.g. 5m, 1h). If not set, block time monitoring is disabled.")
	flag.Parse()

	// Validate required parameters
	if configFilePath == "" || nodeName == "" || supportedQueryIDsMapPath == "" {
		log.Fatal("Usage: go run ./scripts/async-monitors/async-monitor-events.go -rpc-url=<rpc_url> -config=<config_file_path> -query-ids-map=<query_ids_map_file_path> -node=<node_name>")
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

	// Start event type heartbeat checker
	wg.Add(1)
	go checkEventTypeHeartbeats(ctx)

	wg.Wait()
	log.Println("Shutdown complete")
}
