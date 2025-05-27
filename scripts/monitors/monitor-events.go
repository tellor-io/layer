package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/gorilla/websocket"
	"github.com/tellor-io/layer/utils"
)

const (
	PrimaryRpcUrl       = "https://node-palmito.tellorlayer.com/rpc"
	FallbackRpcUrl      = "https://rpc.tellorlayer.com"
	DefaultConfigPath   = "scripts/monitors/event-config.yml"
	AlertCooldownPeriod = 2 * time.Hour
	AlertWindowPeriod   = 10 * time.Minute
	MaxAlertsInWindow   = 10
	PowerThreshold      = 2.0 / 3.0
)

var (
	eventConfig                  EventConfig
	configMutex                  sync.RWMutex
	configFilePath               = "scripts/monitors/event-config.yml"
	discordNotifier              *utils.DiscordNotifier
	Current_Total_Reporter_Power uint64
	reporterPowerMutex           sync.RWMutex
	// Rate limiting variables
	aggregateAlertCount      int
	aggregateAlertTimestamps []time.Time
	aggregateAlertMutex      sync.RWMutex
	aggregateAlertCooldown   time.Time
	// Map to store event types we're interested in
	eventTypeMap map[string]ConfigType
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

type WebsocketSubscribeRequest struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Id      int    `json:"id"`
	Params  Params `json:"params"`
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

type WebsocketReponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      int    `json:"id"`
	Result  struct {
		Query string `json:"query"`
		Data  struct {
			Type  string `json:"type"`
			Value struct {
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
				BlockID struct {
					Hash  string `json:"hash"`
					Parts struct {
						Total int    `json:"total"`
						Hash  string `json:"hash"`
					} `json:"parts"`
				} `json:"block_id"`
				ResultFinalizeBlock struct {
					Events                []Event       `json:"events"`
					TxResults             []TxResult    `json:"tx_results"`
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
				} `json:"result_finalize_block"`
			} `json:"value"`
		} `json:"data"`
	} `json:"result"`
}

type EventConfig struct {
	EventTypes []ConfigType `yaml:"event_types"`
}

type WebSocketClient struct {
	conn        *websocket.Conn
	url         string
	reconnectCh chan struct{}
	done        chan struct{}
	mu          sync.RWMutex
	lastMessage time.Time
}

func NewWebSocketClient(url string) *WebSocketClient {
	return &WebSocketClient{
		url:         url,
		reconnectCh: make(chan struct{}, 1),
		done:        make(chan struct{}),
		lastMessage: time.Now(),
	}
}

func (w *WebSocketClient) connect() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	dialer := websocket.Dialer{
		HandshakeTimeout: 45 * time.Second,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
	}

	conn, _, err := dialer.Dial(w.url, nil)
	if err != nil {
		if strings.EqualFold(w.url, FallbackRpcUrl) {
			return fmt.Errorf("failed to connect: %v", err)
		} else {
			w.url = FallbackRpcUrl
			return w.connect()
		}
	}

	// Set read deadline to detect stale connections
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	// Enable ping/pong
	conn.SetPingHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	w.conn = conn

	subscribeToEvents(w.conn, eventConfig.EventTypes)
	return nil
}

func (w *WebSocketClient) ensureConnection() error {
	if w.conn == nil {
		return w.connect()
	}
	return nil
}

func (w *WebSocketClient) readMessages(ctx context.Context, handler func([]byte) error) {
	timeoutTicker := time.NewTicker(1 * time.Minute)
	defer timeoutTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.reconnectCh:
			if err := w.connect(); err != nil {
				log.Printf("Failed to reconnect: %v", err)
				time.Sleep(5 * time.Second)
				w.reconnectCh <- struct{}{}
			}
		case <-timeoutTicker.C:
			w.mu.RLock()
			if time.Since(w.lastMessage) > 10*time.Minute {
				message := "**Alert: No NewBlock Events Received**\nNo NewBlock events have been received in the last 10 minutes. Please check the chain status."
				discordNotifier := utils.NewDiscordNotifier(eventConfig.EventTypes[0].WebhookURL)
				if err := discordNotifier.SendAlert(message); err != nil {
					log.Printf("Error sending timeout Discord alert: %v", err)
				} else {
					log.Printf("Sent timeout Discord alert")
				}
			}
			w.mu.RUnlock()
		default:
			if err := w.ensureConnection(); err != nil {
				log.Printf("Connection error: %v", err)
				w.reconnectCh <- struct{}{}
				continue
			}

			_, message, err := w.conn.ReadMessage()
			if err != nil {
				log.Printf("Read error: %v", err)
				w.mu.Lock()
				w.conn.Close()
				w.conn = nil
				w.mu.Unlock()
				w.reconnectCh <- struct{}{}
				continue
			}

			w.mu.Lock()
			w.lastMessage = time.Now()
			w.mu.Unlock()

			if err := handler(message); err != nil {
				log.Printf("Handler error: %v", err)
			}
		}
	}
}

func (w *WebSocketClient) healthCheck(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.mu.RLock()
			if w.conn == nil {
				w.mu.RUnlock()
				w.reconnectCh <- struct{}{}
				continue
			}

			// Send ping to check connection
			if err := w.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				w.mu.RUnlock()
				log.Printf("Health check failed: %v", err)
				w.reconnectCh <- struct{}{}
				continue
			}
			w.mu.RUnlock()
		}
	}
}

func loadConfig() error {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	var newConfig EventConfig
	if err := yaml.Unmarshal(data, &newConfig); err != nil {
		return fmt.Errorf("error parsing config file: %v", err)
	}

	// Initialize the event type map
	newEventTypeMap := make(map[string]ConfigType)
	for _, et := range newConfig.EventTypes {
		newEventTypeMap[et.AlertType] = et
		fmt.Printf("Loaded event type: %s (%s)\n", et.AlertName, et.AlertType)
	}

	configMutex.Lock()
	eventConfig = newConfig
	eventTypeMap = newEventTypeMap
	configMutex.Unlock()
	return nil
}

func subscribeToEvents(client *websocket.Conn, eventTypes []ConfigType) error {
	if len(eventTypes) == 0 {
		return fmt.Errorf("no event types configured")
	}

	configMutex.RLock()
	defer configMutex.RUnlock()

	subscribeReq := WebsocketSubscribeRequest{
		Jsonrpc: "2.0",
		Method:  "subscribe",
		Id:      1,
		Params:  Params{Query: "tm.event = 'NewBlock'"},
	}
	req, err := json.Marshal(&subscribeReq)
	if err != nil {
		fmt.Printf("Error marshalling request message: %v\n", err)
		panic(err)
	}
	err = client.WriteMessage(websocket.TextMessage, req)
	if err != nil {
		fmt.Printf("Error writing message: %v\n", err)
		return err
	}
	return nil
}

func startConfigWatcher(ctx context.Context, client *websocket.Conn) {
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
			err := subscribeToEvents(client, eventConfig.EventTypes)
			if err != nil {
				fmt.Printf("Error updating subscriptions: %v\n", err)
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

	for j := 0; j < len(event.Attributes); j++ {
		if event.Attributes[j].Key == "aggregate_power" {
			if aggregatePower, err := strconv.ParseUint(event.Attributes[j].Value, 10, 64); err == nil {
				if float64(aggregatePower) < float64(Current_Total_Reporter_Power)*2/3 {
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
				} else {
					fmt.Println("Aggregate power is greater than 2/3 of total reporter power")
				}
			} else {
				fmt.Printf("Error parsing aggregate power: %v\n", err)
			}
		}
	}
}

func MonitorBlockEvents(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	if err := loadConfig(); err != nil {
		log.Printf("Error loading initial config: %v", err)
		return
	}

	url := url.URL{Scheme: "ws", Host: "34.229.148.107:26657", Path: "/websocket"}
	client := NewWebSocketClient(url.String())

	// Start health check
	go client.healthCheck(ctx)

	// Start the message handler
	go client.readMessages(ctx, func(message []byte) error {
		var data WebsocketReponse
		if err := json.Unmarshal(message, &data); err != nil {
			return fmt.Errorf("unable to unmarshal message: %v", err)
		}

		height := data.Result.Data.Value.Block.Header.Height
		configMutex.RLock()
		if len(data.Result.Data.Value.ResultFinalizeBlock.Events) > 0 {
			processBlockEvents(data.Result.Data.Value.ResultFinalizeBlock.Events, height)
		}
		if len(data.Result.Data.Value.ResultFinalizeBlock.TxResults) > 0 {
			processTransactionEvents(data.Result.Data.Value.ResultFinalizeBlock.TxResults, height)
		}
		configMutex.RUnlock()
		return nil
	})

	// Subscribe to events
	if err := client.ensureConnection(); err != nil {
		log.Printf("Failed to establish initial connection: %v", err)
		return
	}

	subscribeReq := WebsocketSubscribeRequest{
		Jsonrpc: "2.0",
		Method:  "subscribe",
		Id:      1,
		Params:  Params{Query: "tm.event = 'NewBlock'"},
	}

	req, err := json.Marshal(&subscribeReq)
	if err != nil {
		log.Printf("Error marshalling request: %v", err)
		return
	}

	if err := client.conn.WriteMessage(websocket.TextMessage, req); err != nil {
		log.Printf("Error sending subscription request: %v", err)
		return
	}

	// Wait for context cancellation
	<-ctx.Done()
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

// Add a helper function to handle events
func handleEvent(event Event, eventType ConfigType) {
	if event.Type == "aggregate_report" {
		fmt.Println("Handling aggregate report")
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
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			resp, err := http.Get(RpcUrl + "/validators")
			if err != nil {
				fmt.Printf("Error querying validators: %v\n", err)
				continue
			}

			var validatorResp ValidatorResponse
			if err := json.NewDecoder(resp.Body).Decode(&validatorResp); err != nil {
				fmt.Printf("Error decoding validator response: %v\n", err)
				resp.Body.Close()
				continue
			}
			resp.Body.Close()

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
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Received shutdown signal")
		cancel()
	}()

	wg := sync.WaitGroup{}
	wg.Add(2)

	go MonitorBlockEvents(ctx, &wg)
	go updateTotalReporterPower(ctx)

	wg.Wait()
	log.Println("Shutdown complete")
}
