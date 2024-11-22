package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tellor-io/layer/utils"
)

type Params struct {
	Query string `json:"query"`
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

func (c *Client) MonitorCyclelistQuery(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	url := url.URL{Scheme: "ws", Host: "127.0.0.1:26657", Path: "/websocket"}
	client, _, err := websocket.DefaultDialer.Dial(url.String(), nil)
	if err != nil {
		c.logger.Error("dial:", err)
		panic(err)
	}
	defer client.Close()

	var localWG sync.WaitGroup
	queryIdToQueryDataMap := CreateCyclelistQueryIdToQueryDataMap()
	localWG.Add(1)
	go func(wait *sync.WaitGroup) {
		defer wait.Done()
		for {
			_, message, err := client.ReadMessage()
			if err != nil {
				c.logger.Error("read:", err)
				panic(err)
			}
			var data WebsocketReponse
			err = json.Unmarshal(message, &data)
			if err != nil {
				c.logger.Error("Unable to unmarshal read message: ", err)
				c.logger.Info("Response data: ", message)
				panic(err)
			}

			if len(data.Result.Data.Value.Events) == 0 {
				continue
			}

			events := data.Result.Data.Value.Events

			var event Event
			for i := 0; i < len(events); i++ {
				c.logger.Info(fmt.Sprintf("Event read from events: %s", events[i].Type))
				if events[i].Type == "rotating-cyclelist-with-next-query" {
					c.logger.Info("Found the rotate queries event!!!")
					go c.HandleCyclelistEvents(ctx, events[i], queryIdToQueryDataMap)
				} else if events[i].Type == "tip_added" {
					c.logger.Info("Found a tipped query")
					go c.HandleTippedQueryEvents(ctx, events[i], queryIdToQueryDataMap)
				}
			}

			if event.Type == "" {
				c.logger.Error("rotate cyclelist event not found")
				continue
			}
		}
	}(&localWG)

	subscribeCyclelistReq := WebsocketSubscribeRequest{
		Jsonrpc: "2.0",
		Method:  "subscribe",
		Id:      0,
		Params:  Params{Query: "rotating-cyclelist-with-next-query.query_id EXISTS"},
	}
	subscribeTippedQueriesReq := WebsocketSubscribeRequest{
		Jsonrpc: "2.0",
		Method:  "subscribe",
		Id:      0,
		Params:  Params{Query: "tip_added.query_id EXISTS"},
	}
	req, err := json.Marshal(&subscribeCyclelistReq)
	if err != nil {
		c.logger.Error("Error marshalling request message: ", err)
		panic(err)
	}
	err = client.WriteMessage(websocket.TextMessage, req)
	if err != nil {
		c.logger.Error("write:", err)
		return
	}
	req, err = json.Marshal(&subscribeTippedQueriesReq)
	if err != nil {
		c.logger.Error("Error marshalling request message: ", err)
		panic(err)
	}
	err = client.WriteMessage(websocket.TextMessage, req)
	if err != nil {
		c.logger.Error("write:", err)
		return
	}
	localWG.Wait()
}

func (c *Client) HandleCyclelistEvents(ctx context.Context, e Event, qIdToQueryData map[string][]byte) {
	var queryId string
	var querymetaId string
	for i := 0; i < len(e.Attributes); i++ {
		if e.Attributes[i].Key == "query_id" {
			queryId = e.Attributes[i].Value
		} else if e.Attributes[i].Key == "New QueryMeta Id" {
			querymetaId = e.Attributes[i].Value
		}
	}
	c.logger.Info("Message received on websocket: ", e)

	if queryId == "" || querymetaId == "" {
		c.logger.Error("No attribute found for query_id: ", e.Attributes)
		return
	}
	qd := qIdToQueryData[queryId]
	c.logger.Info(fmt.Sprintf("Query data: %s, QueryId: %s", string(qd), queryId))

	nextId, err := strconv.Atoi(querymetaId)
	if err != nil {
		c.logger.Error("error converting id attribute to int: ", err)
		return
	}

	err = c.GenerateAndBroadcastSpotPriceReport(ctx, qd, uint64(nextId))
	if err != nil {
		c.logger.Error(fmt.Sprintf("Error broadcasting cyclelist message: %v", err))
	}
}

func (c *Client) HandleTippedQueryEvents(ctx context.Context, e Event, qIdToQueryData map[string][]byte) {
	var queryId string
	var querymetaId string
	for i := 0; i < len(e.Attributes); i++ {
		if e.Attributes[i].Key == "query_id" {
			queryId = e.Attributes[i].Value
		} else if e.Attributes[i].Key == "querymeta_id" {
			querymetaId = e.Attributes[i].Value
		}
	}

	if queryId == "" || querymetaId == "" {
		c.logger.Error("No attribute found for query_id: ", e.Attributes)
		return
	}
	qd := qIdToQueryData[queryId]
	c.logger.Info(fmt.Sprintf("Query data: %s, QueryId: %s", string(qd), queryId))

	nextId, err := strconv.Atoi(querymetaId)
	if err != nil {
		c.logger.Error("error converting id attribute to int: ", err)
		return
	}

	err = c.GenerateAndBroadcastSpotPriceReport(ctx, qd, uint64(nextId))
	if err != nil {
		c.logger.Error(fmt.Sprintf("Error broadcasting cyclelist message: %v", err))
	}
}

func (c *Client) MonitorTokenBridgeReports(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	var localWG sync.WaitGroup
	for {
		localWG.Add(1)
		go func() {
			defer localWG.Done()
			err := c.generateDepositmessages(context.Background())
			if err != nil {
				c.logger.Error("Error generating deposit messages: ", err)
			}
		}()
		localWG.Wait()

		time.Sleep(4 * time.Minute)
	}
}

func CreateCyclelistQueryIdToQueryDataMap() map[string][]byte {
	cyclelistQueryInfoMap := make(map[string][]byte, 0)
	eth_usd_query_data, _ := utils.QueryBytesFromString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
	cyclelistQueryInfoMap["83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"] = eth_usd_query_data

	btc_usd_query_data, _ := utils.QueryBytesFromString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003627463000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
	cyclelistQueryInfoMap["a6f013ee236804827b77696d350e9f0ac3e879328f2a3021d473a0b778ad78ac"] = btc_usd_query_data

	trb_usd_query_data, _ := utils.QueryBytesFromString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
	cyclelistQueryInfoMap["5c13cd9c97dbb98f2429c101a2a8150e6c7a0ddaff6124ee176a3a411067ded0"] = trb_usd_query_data

	return cyclelistQueryInfoMap
}
