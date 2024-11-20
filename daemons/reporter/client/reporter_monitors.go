package client

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tellor-io/layer/utils"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
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
			c.logger.Info("Websocket message: ", string(message))
			var event sdk.Event
			err = json.Unmarshal(message, &event)
			if err != nil {
				c.logger.Error("Unable to unmarshal read message: ", err)
				c.logger.Info("Response data: ", message)
				panic(err)
			}

			if len(message) == 0 {
				c.logger.Info("EMPTY MESSAGE RECEIVED")
				continue
			}

			c.logger.Info("Message received on websocket: ", event)

			qidAttribute, ok := event.GetAttribute("query_id")
			if !ok {
				for i := 0; i < len(event.Attributes); i++ {
					c.logger.Info("Attribute: ", event.Attributes)
				}
				c.logger.Error("No attribute found for query_id: ", event.Attributes)
				continue
			}
			query_id := qidAttribute.Value
			qd := queryIdToQueryDataMap[query_id]

			idAttribute, ok := event.GetAttribute("New QueryMeta Id")
			if !ok {
				c.logger.Error("no next id found in rotatequeries event")
			}
			nextId, err := strconv.Atoi(idAttribute.Value)
			if err != nil {
				c.logger.Error("error converting id attribute to int: ", err)
				return
			}
			go func(query_data []byte, querymetaId uint64) {
				err := c.GenerateAndBroadcastCyclelistReport(ctx, query_data, querymetaId)
				if err != nil {
					c.logger.Error("Error broadcasting cyclelist message: ", err)
				}
			}(qd, uint64(nextId))
		}
	}(&localWG)

	subscribeReq := WebsocketSubscribeRequest{
		Jsonrpc: "2.0",
		Method:  "subscribe",
		Id:      0,
		Params:  Params{Query: "tm.event=\"rotating-cyclelist-with-existing-nontipped-query\""},
	}
	req, err := json.Marshal(&subscribeReq)
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

func (c *Client) MonitorForTippedQueries(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	var localWG sync.WaitGroup
	for {
		res, err := c.OracleQueryClient.TippedQueries(ctx, &oracletypes.QueryTippedQueriesRequest{
			Pagination: &query.PageRequest{
				Offset: 0,
			},
		})
		if err != nil {
			c.logger.Error("Error querying for TippedQueries: ", err)
			time.Sleep(200 * time.Millisecond)
			continue
		}
		if len(res.Queries) == 0 {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		status, err := c.cosmosCtx.Client.Status(ctx)
		if err != nil {
			c.logger.Info("Error getting status from client: ", err)
		}
		height := uint64(status.SyncInfo.LatestBlockHeight)
		for i := 0; i < len(res.Queries); i++ {
			if height > res.Queries[i].Expiration || strings.EqualFold(res.Queries[i].QueryType, "SpotPrice") {
				if len(res.Queries) == 1 || i == (len(res.Queries)-1) {
					time.Sleep(200 * time.Millisecond)
				}
				continue
			}
			if commitedIds[res.Queries[i].Id] {
				if len(res.Queries) == 1 || i == (len(res.Queries)-1) {
					time.Sleep(200 * time.Millisecond)
				}
				continue
			}

			localWG.Add(1)
			go func(query *oracletypes.QueryMeta) {
				defer localWG.Done()
				err := c.GenerateAndBroadcastSpotPriceReport(ctx, query.QueryData, query)
				if err != nil {
					c.logger.Error("Error generating report for tipped query: ", err)
				} else {
					c.logger.Info("Broadcasted report for tipped query")
				}
			}(res.Queries[i])
		}
		localWG.Wait()
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
