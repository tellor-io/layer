package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/gorilla/websocket"
)

type Params struct {
	Query string `json:"query"`
}

type ConfigType struct {
	AlertName string `json:"alert_name"`
	AlertType string `json:"alert_type"`
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

type EventResponseData struct {
	Height  string  `json:"height"`
	Events  []Event `json:"events"`
	Num_txs string  `json:"num_txs"`
}

type EventResponse struct {
	Type  string            `json:"type"`
	Value EventResponseData `json:"value"`
}

type QueryResult struct {
	Query string        `json:"query"`
	Data  EventResponse `json:"data"`
}

type WebsocketReponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	Id      int         `json:"id"`
	Result  QueryResult `json:"result"`
}

type EventConfig struct {
	EventTypes []ConfigType `yaml:"event_types"`
}

var (
	eventConfig    EventConfig
	configMutex    sync.RWMutex
	configFilePath = "scripts/event-config.yml"
)

func loadConfig() error {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	var newConfig EventConfig
	if err := yaml.Unmarshal(data, &newConfig); err != nil {
		return fmt.Errorf("error parsing config file: %v", err)
	}

	configMutex.Lock()
	eventConfig = newConfig
	configMutex.Unlock()
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

func subscribeToEvents(client *websocket.Conn, eventTypes []ConfigType) error {
	configMutex.RLock()
	for _, eventType := range eventTypes {
		subscribeReq := WebsocketSubscribeRequest{
			Jsonrpc: "2.0",
			Method:  "subscribe",
			Id:      0,
			Params:  Params{Query: eventType.AlertType},
		}
		req, err := json.Marshal(&subscribeReq)
		if err != nil {
			fmt.Printf("Error marshalling request message for %s: %v\n", eventType.AlertName, err)
			continue
		}
		err = client.WriteMessage(websocket.TextMessage, req)
		if err != nil {
			fmt.Printf("Error writing message for %s: %v\n", eventType.AlertName, err)
			return err
		}
	}
	configMutex.RUnlock()
	return nil
}

func MonitorBlockEvents(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	// Initial config load
	if err := loadConfig(); err != nil {
		fmt.Printf("Error loading initial config: %v\n", err)
		return
	}

	url := url.URL{Scheme: "ws", Host: "127.0.0.1:26657", Path: "/websocket"}
	client, _, err := websocket.DefaultDialer.Dial(url.String(), nil)
	if err != nil {
		fmt.Println("Error dialing: ", err)
		panic(err)
	}
	defer client.Close()

	// Start config watcher after client is created
	go startConfigWatcher(ctx, client)

	var localWG sync.WaitGroup
	localWG.Add(1)
	go func(wait *sync.WaitGroup) {
		defer wait.Done()
		for {
			_, message, err := client.ReadMessage()
			if err != nil {
				fmt.Println("Error reading message: ", err)
				panic(err)
			}
			var data WebsocketReponse
			err = json.Unmarshal(message, &data)
			if err != nil {
				fmt.Println("Unable to unmarshal read message: ", err)
				fmt.Printf("Response data: %s\n", message)
				panic(err)
			}

			if len(data.Result.Data.Value.Events) == 0 {
				continue
			}

			events := data.Result.Data.Value.Events

			for i := 0; i < len(events); i++ {
				fmt.Printf("Event read from events: %s\n", events[i].Type)
				// TODO: handle events
			}
		}
	}(&localWG)

	err = subscribeToEvents(client, eventConfig.EventTypes)
	if err != nil {
		fmt.Printf("Error updating subscriptions: %v\n", err)
	}

	localWG.Wait()
}

func main() {
	ctx := context.Background()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go MonitorBlockEvents(ctx, &wg)
	wg.Wait()
}
