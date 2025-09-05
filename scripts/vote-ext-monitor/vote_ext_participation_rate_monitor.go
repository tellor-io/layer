package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tellor-io/layer/utils"
	"gopkg.in/yaml.v3"
)

const (
	DefaultRpcURL        = "127.0.0.1:26657"
	MaxReconnectAttempts = 5
	BlockQueryInterval   = 1 * time.Second
)

var (
	// Command line parameters
	rpcURL         string
	swaggerAPIURL  string
	configFilePath string
	nodeName       string
	// Block tracking variables
	currentBlockHeight uint64
	blockHeightMutex   sync.RWMutex
	// CSV file variables
	csvFile   *os.File
	csvWriter *csv.Writer
	csvMutex  sync.Mutex

	eventTypeMap map[string]ConfigType
	configMutex  sync.RWMutex

	// Add new variables for file rotation
	currentCSVFileName string
	rotationTicker     *time.Ticker
	lastRotationDate   time.Time

	lastNotificationTimeMapMutex sync.RWMutex
	lastNotificationTimeMap      map[string]time.Time
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

type OracleAttestations struct {
	OperatorAddresses []string `json:"operator_addresses"`
	Attestations      []string `json:"attestations"`
	Snapshots         []string `json:"snapshots"`
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
	OracleAttestations OracleAttestations `json:"oracle_attestations"`
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
	return rotateCSVFile()
}

// writeToCSV writes a row to the CSV file
func writeToCSV(height, timestamp uint64, participationRate float64) error {
	// Check if we need to rotate to a new daily file
	if shouldRotateFile() {
		if err := rotateCSVFile(); err != nil {
			return fmt.Errorf("failed to rotate CSV file: %w", err)
		}
	}

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
				eventType, ok := eventTypeMap["liveness-alert"]
				if !ok {
					log.Printf("liveness alert event not found")
					continue
				}
				// Send liveness alert
				message := fmt.Sprintf("**Alert: Node %s is Not Responding**\nFailed to get latest block height from node %s. Please check the node status and logs.", nodeName, nodeName)
				discordNotifier := utils.NewDiscordNotifier(eventType.WebhookURL)
				if err := discordNotifier.SendAlert(message); err != nil {
					log.Printf("Error sending final Discord alert: %v\n", err)
				} else {
					log.Printf("Sent final Discord alert and starting cooldown\n")
				}
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

func analyzeEVMAddressesFromOracleAttestation(oracleAttestations OracleAttestations) (map[string][]string, error) {
	// Map to store operator address -> EVM address mapping
	validatorEVMAddresses := make(map[string][]string)

	// Check if we have matching arrays
	if len(oracleAttestations.OperatorAddresses) != len(oracleAttestations.Attestations) ||
		len(oracleAttestations.OperatorAddresses) != len(oracleAttestations.Snapshots) {
		return nil, fmt.Errorf("mismatch between arrays: operator addresses (%d), attestations (%d), snapshots (%d)",
			len(oracleAttestations.OperatorAddresses), len(oracleAttestations.Attestations), len(oracleAttestations.Snapshots))
	}

	// Loop through each operator address, attestation, and snapshot triplet
	for i, operatorAddr := range oracleAttestations.OperatorAddresses {
		attestation := oracleAttestations.Attestations[i]
		snapshot := oracleAttestations.Snapshots[i]

		// Skip if we already processed this operator address
		if _, exists := validatorEVMAddresses[operatorAddr]; exists {
			continue
		}

		// Skip empty attestations or snapshots
		if attestation == "" {
			log.Printf("Skipping empty attestation for operator %s", operatorAddr)
			continue
		}
		if snapshot == "" {
			log.Printf("Skipping empty snapshot for operator %s", operatorAddr)
			continue
		}

		// Decode the attestation (signature) from base64
		sigBytes, err := base64.StdEncoding.DecodeString(attestation)
		if err != nil {
			log.Printf("Failed to decode attestation signature for operator %s: %v", operatorAddr, err)
			continue
		}

		// Decode the snapshot from base64
		snapshotBytes, err := base64.StdEncoding.DecodeString(snapshot)
		if err != nil {
			log.Printf("Failed to decode snapshot for operator %s: %v", operatorAddr, err)
			continue
		}

		// The attestation is the signature, and the snapshot is what was signed
		// Check if signature has the correct length (64 bytes for R || S without recovery ID)
		if len(sigBytes) != 64 {
			log.Printf("Invalid signature length for operator %s: got %d bytes, expected 64", operatorAddr, len(sigBytes))
			continue
		}

		// Hash the snapshot data (this is what was signed)
		msgHash := sha256.Sum256(snapshotBytes)

		// Try to recover the address with both recovery IDs (0 and 1)
		recoveredAddresses := make([]string, 2)
		for i, recoveryID := range []byte{0, 1} {
			// Append recovery ID to signature
			sigWithID := append(sigBytes[:64], recoveryID)

			// Recover public key from signature
			pubKey, err := crypto.SigToPub(msgHash[:], sigWithID)
			if err != nil {
				continue // Try next recovery ID
			}

			// Derive EVM address from public key
			recoveredAddr := crypto.PubkeyToAddress(*pubKey)
			recoveredAddresses[i] = strings.ToLower(recoveredAddr.Hex()[2:])
		}

		if len(recoveredAddresses) == 0 {
			log.Printf("Failed to recover EVM address from attestation for operator %s", operatorAddr)
			continue
		}

		validatorEVMAddresses[operatorAddr] = recoveredAddresses
	}

	if len(validatorEVMAddresses) == 0 {
		return nil, fmt.Errorf("no valid EVM addresses could be derived from oracle attestations")
	}

	return validatorEVMAddresses, nil
}

// CheckpointResponse represents the response from the validator checkpoint params API
type CheckpointResponse struct {
	Checkpoint     string `json:"checkpoint"`
	ValsetHash     string `json:"valset_hash"`
	Timestamp      string `json:"timestamp"`
	PowerThreshold string `json:"power_threshold"`
}

// getValidatorCheckpointParams queries the bridge module to get checkpoint data for a given timestamp
func getValidatorCheckpointParams(timestamp uint64) (*CheckpointResponse, error) {
	// Use swagger API URL if provided, otherwise fall back to the node URL
	if swaggerAPIURL == "" {
		return nil, fmt.Errorf("swagger API URL is not provided")
	}

	url := fmt.Sprintf("%s/layer/bridge/get_validator_checkpoint_params/%d", swaggerAPIURL, timestamp)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to query checkpoint params: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("checkpoint params query failed with status: %d", resp.StatusCode)
	}

	var checkpointResp CheckpointResponse
	if err := json.NewDecoder(resp.Body).Decode(&checkpointResp); err != nil {
		return nil, fmt.Errorf("failed to decode checkpoint response: %w", err)
	}

	return &checkpointResp, nil
}

func deriveEVMAddressFromValsetSigs(extensionSignature string, timestamp uint64) ([]string, error) {
	// Decode the signature from base64
	sigBytes, err := base64.StdEncoding.DecodeString(extensionSignature)
	if err != nil {
		return []string{}, fmt.Errorf("failed to decode signature from base64: %w", err)
	}

	// The signature should be 64 bytes (R || S without recovery ID)
	if len(sigBytes) != 64 {
		return []string{}, fmt.Errorf("signature length is not 64 bytes: got %d", len(sigBytes))
	}

	// Get the checkpoint data from the bridge module
	checkpointResp, err := getValidatorCheckpointParams(timestamp)
	if err != nil {
		return []string{}, fmt.Errorf("failed to get checkpoint params: %w", err)
	}

	// Decode the checkpoint from hex string to bytes
	checkpointBytes, err := hex.DecodeString(checkpointResp.Checkpoint)
	if err != nil {
		return []string{}, fmt.Errorf("failed to decode checkpoint from hex: %w", err)
	}

	// Hash the checkpoint data (this is what was signed)
	msgHash := sha256.Sum256(checkpointBytes)

	// Try to recover the address with both recovery IDs (0 and 1)
	evmAddresses := make([]string, 2)
	for i, recoveryID := range []byte{0, 1} {
		// Append recovery ID to signature
		sigWithID := append(sigBytes[:64], recoveryID)

		// Recover public key from signature
		pubKey, err := crypto.SigToPub(msgHash[:], sigWithID)
		if err != nil {
			continue // Try next recovery ID
		}

		// Derive EVM address from public key
		recoveredAddr := crypto.PubkeyToAddress(*pubKey)
		evmAddresses[i] = strings.ToLower(recoveredAddr.Hex()[2:]) // Remove 0x prefix
	}

	if len(evmAddresses) == 0 {
		return []string{}, fmt.Errorf("failed to recover any address from signature")
	}

	return evmAddresses, nil
}

// EvmAddressResponse represents the response from the get-evm-address-by-validator-address API
type EvmAddressResponse struct {
	EvmAddress string `json:"evm_address"`
}

// APIErrorResponse represents an error response from the API
type APIErrorResponse struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Details []interface{} `json:"details"`
}

// Custom error types for EVM address lookup
type EVMAddressNotFoundError struct {
	OperatorAddress string
}

func (e *EVMAddressNotFoundError) Error() string {
	return fmt.Sprintf("no EVM address found for operator %s", e.OperatorAddress)
}

type EVMAddressLookupError struct {
	OperatorAddress string
	Err             error
}

func (e *EVMAddressLookupError) Error() string {
	return fmt.Sprintf("failed to lookup EVM address for operator %s: %v", e.OperatorAddress, e.Err)
}

// logNotificationMapState logs the current state of the notification map for debugging
func logNotificationMapState() {
	lastNotificationTimeMapMutex.RLock()
	defer lastNotificationTimeMapMutex.RUnlock()

	log.Printf("DEBUG: Current notification map has %d entries:", len(lastNotificationTimeMap))
	for operator, lastTime := range lastNotificationTimeMap {
		timeSince := time.Since(lastTime)
		log.Printf("DEBUG:   %s: last notification %v ago", operator, timeSince)
	}
}

// getEVMAddressFromOperatorAddress calls the Swagger API to get EVM address by validator address
func getEVMAddressFromOperatorAddress(operatorAddress string) (string, error) {
	// Use swagger API URL if provided, otherwise return error
	if swaggerAPIURL == "" {
		return "", &EVMAddressLookupError{
			OperatorAddress: operatorAddress,
			Err:             fmt.Errorf("swagger API URL is not provided"),
		}
	}

	// URL encode the operator address to handle special characters
	encodedAddress := url.QueryEscape(operatorAddress)
	url := fmt.Sprintf("%s/layer/bridge/get_evm_address_by_validator_address/%s", swaggerAPIURL, encodedAddress)

	resp, err := http.Get(url)
	if err != nil {
		return "", &EVMAddressLookupError{
			OperatorAddress: operatorAddress,
			Err:             err,
		}
	}
	defer resp.Body.Close()

	// Read the response body first to see what we got
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", &EVMAddressLookupError{
			OperatorAddress: operatorAddress,
			Err:             err,
		}
	}

	// First try to parse as an error response (regardless of HTTP status code)
	var apiError APIErrorResponse
	if err := json.Unmarshal(body, &apiError); err == nil && apiError.Code != 0 {
		// This is an API error response (like code 13 "failed to get eth address")
		// Treat this as an invalid signature case by returning a specific error
		log.Printf("API error response detected: code=%d, message=%s", apiError.Code, apiError.Message)
		return "", &EVMAddressNotFoundError{
			OperatorAddress: operatorAddress,
		}
	}

	if resp.StatusCode != http.StatusOK {
		return "", &EVMAddressLookupError{
			OperatorAddress: operatorAddress,
			Err:             fmt.Errorf("HTTP status %d", resp.StatusCode),
		}
	}

	// Try to parse as successful response
	var evmAddressResp EvmAddressResponse
	if err := json.Unmarshal(body, &evmAddressResp); err != nil {
		return "", &EVMAddressLookupError{
			OperatorAddress: operatorAddress,
			Err:             err,
		}
	}

	if evmAddressResp.EvmAddress == "" {
		return "", &EVMAddressNotFoundError{OperatorAddress: operatorAddress}
	}

	return evmAddressResp.EvmAddress, nil
}

// verifySignaturesInVoteExtension verifies all signatures found in a vote extension
func verifySignaturesInVoteExtension(voteExtData VoteExtensionData, height uint64, timestamp uint64) error {
	log.Printf("Verifying signatures in vote extension for block %d", height)

	invalidSignatures := []string{}

	// Verify valset signatures if they exist
	if len(voteExtData.ValsetSigs.OperatorAddresses) > 0 {
		log.Printf("Found %d valset signatures to verify", len(voteExtData.ValsetSigs.OperatorAddresses))

		for i, operatorAddr := range voteExtData.ValsetSigs.OperatorAddresses {
			if i >= len(voteExtData.ValsetSigs.Signatures) {
				log.Printf("Warning: Missing signature for operator %s", operatorAddr)
				continue
			}

			signature := voteExtData.ValsetSigs.Signatures[i]
			if signature == "" {
				log.Printf("Warning: Empty signature for operator %s", operatorAddr)
				continue
			}

			// Derive EVM address from signature
			derivedEVMAddrs, err := deriveEVMAddressFromValsetSigs(signature, timestamp)
			if err != nil {
				log.Printf("Failed to derive EVM address from valset signature for operator %s: %v", operatorAddr, err)
				lastNotificationTime, ok := lastNotificationTimeMap[operatorAddr]
				if !ok || time.Since(lastNotificationTime) > 24*time.Hour {
					lastNotificationTimeMapMutex.Lock()
					lastNotificationTimeMap[operatorAddr] = time.Now()
					lastNotificationTimeMapMutex.Unlock()
					invalidSignatures = append(invalidSignatures, fmt.Sprintf("failure to derive evm address from signature for operator %s: %v", operatorAddr, err))
				} else {
					log.Printf("Skipping duplicate failure to derive evm address from signature for operator %s", operatorAddr)
				}
				continue
			}

			// Get expected EVM address from API
			expectedEVMAddr, err := getEVMAddressFromOperatorAddress(operatorAddr)
			if err != nil {
				// Check if it's a lookup error (RPC down, network issues, etc.) vs not found/API error
				if _, ok := err.(*EVMAddressLookupError); ok {
					log.Printf("Could not get expected EVM address for operator %s due to lookup error: %v", operatorAddr, err)
					// Don't add to invalidSignatures for lookup errors - just log and continue
					continue
				}
				// For EVMAddressNotFoundError (including API errors like code 13), treat as invalid signature
				if _, ok := err.(*EVMAddressNotFoundError); ok {
					log.Printf("EVM address not found for operator %s (API error): %v", operatorAddr, err)
					lastNotificationTime, ok := lastNotificationTimeMap[operatorAddr]
					if !ok || time.Since(lastNotificationTime) > 24*time.Hour {
						lastNotificationTimeMapMutex.Lock()
						lastNotificationTimeMap[operatorAddr] = time.Now()
						lastNotificationTimeMapMutex.Unlock()
						invalidSignatures = append(invalidSignatures, fmt.Sprintf("Valset signature for operator %s: derived %s, expected %s", operatorAddr, derivedEVMAddrs[0], expectedEVMAddr))
					} else {
						log.Printf("Skipping duplicate valset signature attestation for operator %s: derived %s, expected %s", operatorAddr, derivedEVMAddrs[0], expectedEVMAddr)
					}
					continue
				}
				// For other errors, we still want to track as invalid
				log.Printf("Could not get expected EVM address for operator %s: %v", operatorAddr, err)
				continue
			}

			// Compare derived vs expected EVM address
			if derivedEVMAddrs[0] != expectedEVMAddr && derivedEVMAddrs[1] != expectedEVMAddr {
				log.Printf("Invalid valset signature for operator %s: derived %s, expected %s", operatorAddr, derivedEVMAddrs[0], expectedEVMAddr)
				lastNotificationTime, ok := lastNotificationTimeMap[operatorAddr]
				if !ok || time.Since(lastNotificationTime) > 24*time.Hour {
					lastNotificationTimeMapMutex.Lock()
					lastNotificationTimeMap[operatorAddr] = time.Now()
					lastNotificationTimeMapMutex.Unlock()
					invalidSignatures = append(invalidSignatures, fmt.Sprintf("Valset signature for operator %s: derived %s, expected %s", operatorAddr, derivedEVMAddrs[0], expectedEVMAddr))
				} else {
					log.Printf("Skipping duplicate valset signature attestation for operator %s: derived %s, expected %s", operatorAddr, derivedEVMAddrs[0], expectedEVMAddr)
				}
			} else {
				log.Printf("Valid valset signature for operator %s: %s", operatorAddr, derivedEVMAddrs[0])
			}
		}
	}

	// Verify oracle attestations if they exist
	if len(voteExtData.OracleAttestations.OperatorAddresses) > 0 {
		log.Printf("Found %d oracle attestations to verify", len(voteExtData.OracleAttestations.OperatorAddresses))

		// Derive EVM addresses from oracle attestations
		derivedEVMAddresses, err := analyzeEVMAddressesFromOracleAttestation(voteExtData.OracleAttestations)
		if err != nil {
			log.Printf("Failed to derive EVM addresses from oracle attestations: %v", err)
			invalidSignatures = append(invalidSignatures, fmt.Sprintf("Oracle attestations: %v", err))
		} else {
			// Verify each derived address against expected address
			for operatorAddr, derivedEVMAddr := range derivedEVMAddresses {
				expectedEVMAddr, err := getEVMAddressFromOperatorAddress(operatorAddr)
				if err != nil {
					// Check if it's a lookup error (RPC down, network issues, etc.) vs not found/API error
					if _, ok := err.(*EVMAddressLookupError); ok {
						log.Printf("Could not get expected EVM address for oracle operator %s due to lookup error: %v", operatorAddr, err)
						// Don't add to invalidSignatures for lookup errors - just log and continue
						continue
					}
					// For EVMAddressNotFoundError (including API errors like code 13), treat as invalid signature
					if _, ok := err.(*EVMAddressNotFoundError); ok {
						log.Printf("EVM address not found for oracle operator %s (API error): %v", operatorAddr, err)
						lastNotificationTime, ok := lastNotificationTimeMap[operatorAddr]
						if !ok {
							log.Printf("DEBUG: No previous notification time found for operator %s (EVM not found), will send alert", operatorAddr)
							lastNotificationTimeMapMutex.Lock()
							lastNotificationTimeMap[operatorAddr] = time.Now()
							lastNotificationTimeMapMutex.Unlock()
							invalidSignatures = append(invalidSignatures, fmt.Sprintf("Oracle attestation for operator %s: EVM address not found", operatorAddr))
						} else if time.Since(lastNotificationTime) > 24*time.Hour {
							log.Printf("DEBUG: Previous notification for operator %s (EVM not found) was %v ago (>24h), will send alert", operatorAddr, time.Since(lastNotificationTime))
							lastNotificationTimeMapMutex.Lock()
							lastNotificationTimeMap[operatorAddr] = time.Now()
							lastNotificationTimeMapMutex.Unlock()
							invalidSignatures = append(invalidSignatures, fmt.Sprintf("Oracle attestation for operator %s: EVM address not found", operatorAddr))
						} else {
							log.Printf("DEBUG: Skipping duplicate oracle attestation for operator %s (EVM not found): last notification was %v ago", operatorAddr, time.Since(lastNotificationTime))
						}
						continue
					}
					// For other errors, we still want to track as invalid
					log.Printf("Could not get expected EVM address for oracle operator %s: %v", operatorAddr, err)
					lastNotificationTime, ok := lastNotificationTimeMap[operatorAddr]
					if !ok {
						log.Printf("DEBUG: No previous notification time found for operator %s, will send alert", operatorAddr)
						lastNotificationTimeMapMutex.Lock()
						lastNotificationTimeMap[operatorAddr] = time.Now()
						lastNotificationTimeMapMutex.Unlock()
						invalidSignatures = append(invalidSignatures, fmt.Sprintf("Oracle attestation for operator %s: Could not get expected EVM address", operatorAddr))
					} else if time.Since(lastNotificationTime) > 24*time.Hour {
						log.Printf("DEBUG: Previous notification for operator %s was %v ago (>24h), will send alert", operatorAddr, time.Since(lastNotificationTime))
						lastNotificationTimeMapMutex.Lock()
						lastNotificationTimeMap[operatorAddr] = time.Now()
						lastNotificationTimeMapMutex.Unlock()
						invalidSignatures = append(invalidSignatures, fmt.Sprintf("Oracle attestation for operator %s: Could not get expected EVM address", operatorAddr))
					} else {
						log.Printf("DEBUG: Skipping duplicate oracle attestation for operator %s: last notification was %v ago", operatorAddr, time.Since(lastNotificationTime))
					}
					continue
				}

				if !strings.EqualFold(derivedEVMAddr[0], expectedEVMAddr) && !strings.EqualFold(derivedEVMAddr[1], expectedEVMAddr) {
					log.Printf("Invalid oracle attestation for operator %s: derived %s, expected %s", operatorAddr, derivedEVMAddr, expectedEVMAddr)
					lastNotificationTime, ok := lastNotificationTimeMap[operatorAddr]
					if !ok || time.Since(lastNotificationTime) > 24*time.Hour {
						lastNotificationTimeMapMutex.Lock()
						lastNotificationTimeMap[operatorAddr] = time.Now()
						lastNotificationTimeMapMutex.Unlock()
						invalidSignatures = append(invalidSignatures, fmt.Sprintf("Oracle attestation for operator %s: derived %s, expected %s", operatorAddr, derivedEVMAddr, expectedEVMAddr))
					} else {
						log.Printf("Skipping duplicate oracle attestation for operator %s: derived %s, expected %s", operatorAddr, derivedEVMAddr, expectedEVMAddr)
					}

				}
			}
		}
	}

	// Send Discord alert if any invalid signatures were found
	if len(invalidSignatures) > 0 {
		message := fmt.Sprintf("**ALERT: Invalid Signatures Detected**\nBlock %d on node %s contains invalid signatures:\n", height, nodeName)
		for _, invalidSig := range invalidSignatures {
			message += fmt.Sprintf("• %s\n", invalidSig)
		}

		eventType, ok := eventTypeMap["invalid-signature-alert"]
		if !ok {
			log.Printf("Warning: invalid-signature-alert event type not found in config")
			// Fall back to vote-ext-part-rate if available
			eventType, ok = eventTypeMap["vote-ext-part-rate"]
			if !ok {
				log.Printf("Error: No suitable event type found for signature alert")
				return fmt.Errorf("no suitable event type found for signature alert")
			}
		}

		discordNotifier := utils.NewDiscordNotifier(eventType.WebhookURL)
		if err := discordNotifier.SendAlert(message); err != nil {
			log.Printf("Error sending Discord alert for invalid signatures: %v", err)
		} else {
			log.Printf("Sent Discord alert for %d invalid signatures", len(invalidSignatures))
		}
	} else {
		log.Printf("All signatures in block %d are valid", height)
	}

	return nil
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

	// Parse block timestamp
	blockTime, err := time.Parse(time.RFC3339Nano, blockResponse.Result.Block.Header.Time)
	if err != nil {
		log.Printf("Block %d failed to parse block time: %v", height, err)
		return fmt.Errorf("failed to parse block time: %w", err)
	}
	timestamp := uint64(blockTime.UnixNano() / 1e6) // Convert to milliseconds

	// Verify signatures separately from participation rate monitoring
	if err := verifySignaturesInVoteExtension(voteExtData, height, timestamp); err != nil {
		log.Printf("Block %d signature verification failed: %v", height, err)
		// Continue with participation rate monitoring even if signature verification fails
	}

	// Log notification map state every 100 blocks for debugging
	if height%100 == 0 {
		logNotificationMapState()
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

	// Count votes with valid vote extensions
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

	// Write to CSV file (removed signature verification from CSV data)
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
	log.Printf("  - Block timestamp: %d", timestamp)

	// TODO: Add alerting logic for low participation rates
	if participationRate < 80.0 {
		message := fmt.Sprintf("WARNING: Block %d has low vote extension participation rate: %.2f%%", height, participationRate)
		eventType, ok := eventTypeMap["vote-ext-part-rate"]
		if !ok {
			panic("error getting vote ext event type")
		}

		discordNotifier := utils.NewDiscordNotifier(eventType.WebhookURL)
		if err := discordNotifier.SendAlert(message); err != nil {
			log.Printf("Error sending Discord alert: %v\n", err)
		} else {
			log.Printf("Sent Discord alert for low participation rate\n")
		}
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
		log.Printf("Loaded event type: %s (%s)\n", et.AlertName, et.AlertType)
	}

	configMutex.Lock()
	eventTypeMap = newEventTypeMap
	configMutex.Unlock()
	return nil
}

// getCurrentCSVFileName returns the CSV filename for the current date
func getCurrentCSVFileName() string {
	return fmt.Sprintf("vote_extension_participation_%s.csv", time.Now().Format("2006-01-02"))
}

// shouldRotateFile checks if we need to rotate to a new daily file
func shouldRotateFile() bool {
	today := time.Now().Format("2006-01-02")
	return currentCSVFileName == "" || !strings.Contains(currentCSVFileName, today)
}

// rotateCSVFile closes the current file and opens a new one for the current date
func rotateCSVFile() error {
	csvMutex.Lock()
	defer csvMutex.Unlock()

	// Close current file if it exists
	if csvFile != nil {
		csvFile.Close()
		csvFile = nil
		csvWriter = nil
	}

	// Get new filename for today
	newFileName := getCurrentCSVFileName()

	// Check if today's file already exists
	fileExists := false
	if _, err := os.Stat(newFileName); err == nil {
		fileExists = true
		log.Printf("Today's CSV file already exists: %s", newFileName)
	}

	// Open file in append mode if it exists, create if it doesn't
	var err error
	if fileExists {
		csvFile, err = os.OpenFile(newFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("failed to open existing CSV file %s: %w", newFileName, err)
		}
	} else {
		csvFile, err = os.Create(newFileName)
		if err != nil {
			return fmt.Errorf("failed to create CSV file %s: %w", newFileName, err)
		}
	}

	csvWriter = csv.NewWriter(csvFile)
	currentCSVFileName = newFileName

	// Only write headers if this is a new file
	if !fileExists {
		headers := []string{"height", "timestamp", "vote_ext_participation_rate"}
		if err := csvWriter.Write(headers); err != nil {
			return fmt.Errorf("failed to write CSV headers: %w", err)
		}

		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			return fmt.Errorf("failed to flush CSV headers: %w", err)
		}

		log.Printf("Created new CSV file: %s", newFileName)
	} else {
		log.Printf("Appending to existing CSV file: %s", newFileName)
	}

	lastRotationDate = time.Now()
	return nil
}

// cleanupOldFiles removes CSV files older than the specified duration
func cleanupOldFiles(retentionDays int) error {
	pattern := "vote_extension_participation_*.csv"
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob CSV files: %w", err)
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	deletedCount := 0

	for _, file := range files {
		fileInfo, err := os.Stat(file)
		if err != nil {
			log.Printf("Warning: Could not stat file %s: %v", file, err)
			continue
		}

		if fileInfo.ModTime().Before(cutoffTime) {
			if err := os.Remove(file); err != nil {
				log.Printf("Warning: Could not delete old file %s: %v", file, err)
			} else {
				log.Printf("Deleted old file: %s", file)
				deletedCount++
			}
		}
	}

	if deletedCount > 0 {
		log.Printf("Cleanup completed: deleted %d old files", deletedCount)
	}
	return nil
}

// fileRotationManager handles daily file rotation and cleanup
func fileRotationManager(ctx context.Context, wg *sync.WaitGroup) {
	defer func() {
		recoverAndAlert("fileRotationManager")
		wg.Done()
	}()

	// Calculate time until next midnight
	now := time.Now()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	initialDelay := nextMidnight.Sub(now)

	log.Printf("File rotation manager started. Next rotation at: %s (in %v)", nextMidnight.Format("2006-01-02 15:04:05"), initialDelay)

	// Wait for initial rotation time
	select {
	case <-time.After(initialDelay):
	case <-ctx.Done():
		return
	}

	// Start daily rotation ticker
	rotationTicker = time.NewTicker(24 * time.Hour)
	defer rotationTicker.Stop()

	// Perform initial rotation
	if err := rotateCSVFile(); err != nil {
		log.Printf("Error during initial file rotation: %v", err)
	}

	// Perform initial cleanup
	if err := cleanupOldFiles(7); err != nil {
		log.Printf("Error during initial cleanup: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-rotationTicker.C:
			log.Printf("Performing daily file rotation and cleanup")

			// Rotate to new file
			if err := rotateCSVFile(); err != nil {
				log.Printf("Error during file rotation: %v", err)
				continue
			}

			// Clean up old files (older than 7 days)
			if err := cleanupOldFiles(7); err != nil {
				log.Printf("Error during cleanup: %v", err)
			}

			// Trigger daily report generation
			go generateDailyReport()
		}
	}
}

// generateDailyReport analyzes yesterday's data and sends a report
func generateDailyReport() {
	log.Printf("Generating daily report")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	fileName := fmt.Sprintf("vote_extension_participation_%s.csv", yesterday)

	// Check if yesterday's file exists
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		log.Printf("No data file found for %s, skipping daily report", yesterday)
		return
	}

	report, err := analyzeDailyData(fileName, yesterday)
	if err != nil {
		log.Printf("Error generating daily report for %s: %v", yesterday, err)
		return
	}

	// Send the report
	if err := sendDailyReport(report); err != nil {
		log.Printf("Error sending daily report for %s: %v", yesterday, err)
	}
}

func main() {
	// Parse command line flags
	flag.StringVar(&rpcURL, "rpc-url", DefaultRpcURL, "RPC URL (default: 127.0.0.1:26657)")
	flag.StringVar(&swaggerAPIURL, "swagger-api-url", "", "Swagger API URL for bridge module queries (optional)")
	flag.StringVar(&configFilePath, "config", "", "Path to config file")
	flag.StringVar(&nodeName, "node", "", "Name of the node being monitored")
	flag.Parse()

	// Validate required parameters
	if nodeName == "" {
		log.Fatal("Usage: go run ./scripts/vote-ext-monitor/vote_ext_participation_rate_monitor.go -rpc-url=<rpc_url> -node=<node_name> [-swagger-api-url=<swagger_api_url>]")
	}

	// Initialize the notification time map
	lastNotificationTimeMap = make(map[string]time.Time)
	log.Printf("DEBUG: Process started with PID %d at %v", os.Getpid(), time.Now())
	log.Printf("DEBUG: Initialized lastNotificationTimeMap at %v", time.Now())

	// Initialize CSV file with rotation
	if err := initCSVFile(); err != nil {
		log.Fatalf("Failed to initialize CSV file: %v", err)
	}
	defer func() {
		if csvFile != nil {
			csvFile.Close()
		}
	}()

	err := loadConfig()
	if err != nil {
		panic(err)
	}

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
	wg.Add(2) // MonitorBlocks + fileRotationManager

	go MonitorBlocks(ctx, &wg)
	go fileRotationManager(ctx, &wg)

	wg.Wait()
	log.Println("Shutdown complete")
}
