// missed-aggregates scans aggregate_report events from a start height to a frozen
// end height and reports aggregates where a given reporter did not participate.
//
// Usage:
//
//	go run ./scripts/missed-aggregates \
//	  --rpc-url https://mainnet.tellorlayer.com/rpc  \
//	  --reporter tellor1... \
//	  --start-height 12345678 \
//	  --layerd ./layerd \
//	  --out missed.json
//
// For SpotPrice aggregates, query_data is ABI-decoded to an asset pair (e.g. eth/usd)
// so daemon logs can be correlated by height/time and pair.
package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const (
	aggregateReportEventType = "aggregate_report"
	defaultPageLimit         = 1000
	maxConcurrentBlocks      = 10
	statusMissed             = "missed"
	statusUnknownPruned      = "unknown_pruned"
	statusParticipated       = "participated"
)

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

type BlockResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      int    `json:"id"`
	Result  struct {
		Block struct {
			Header struct {
				Height string `json:"height"`
				Time   string `json:"time"`
			} `json:"header"`
		} `json:"block"`
	} `json:"result"`
}

type BlockResultsResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      int    `json:"id"`
	Result  struct {
		Height              string  `json:"height"`
		FinalizeBlockEvents []Event `json:"finalize_block_events"`
	} `json:"result"`
}

type AggregateEvent struct {
	Height            uint64
	QueryID           string
	QueryData         string
	Value             string
	AggregatePower    string
	MicroReportHeight string
	MicroReportType   string
	TimestampMs       uint64
}

type MicroReport struct {
	Reporter string `json:"reporter"`
}

type PageResponse struct {
	NextKey string `json:"next_key"`
	Total   string `json:"total"`
}

type ReportsByAggregateResponse struct {
	MicroReports []MicroReport `json:"microReports"`
	Pagination   *PageResponse `json:"pagination"`
}

type ResultRow struct {
	Height            uint64 `json:"height"`
	TimestampMs       uint64 `json:"timestamp_ms"`
	TimestampRFC3339  string `json:"timestamp_rfc3339"`
	QueryID           string `json:"query_id"`
	QueryType         string `json:"query_type"`
	AssetPair         string `json:"asset_pair,omitempty"`
	QueryData         string `json:"query_data"`
	AggregateValue    string `json:"aggregate_value"`
	AggregatePower    string `json:"aggregate_power"`
	MicroReportHeight string `json:"micro_report_height"`
	Status            string `json:"status"`
}

type Summary struct {
	Reporter     string      `json:"reporter"`
	RPCURL       string      `json:"rpc_url"`
	StartHeight  uint64      `json:"start_height"`
	EndHeight    uint64      `json:"end_height"`
	Scanned      int         `json:"aggregates_scanned"`
	Missed       int         `json:"missed"`
	Unknown      int         `json:"unknown_pruned"`
	Participated int         `json:"participated"`
	Results      []ResultRow `json:"results"`
}

type RPCClient struct {
	baseURL string
	client  *http.Client
}

func main() {
	log.SetOutput(os.Stderr)

	rpcURL := flag.String("rpc-url", "", "CometBFT RPC URL (host:26657 or http(s)://...)")
	reporter := flag.String("reporter", "", "Reporter bech32 address")
	startHeight := flag.Uint64("start-height", 0, "Inclusive start block height")
	endHeightFlag := flag.Uint64("end-height", 0, "Inclusive end height (default: latest at script start)")
	layerdPath := flag.String("layerd", "", "Path to layerd binary (default: ./layerd or $LAYERD_PATH)")
	outPath := flag.String("out", "missed-aggregates.json", "Output JSON path (use - for stdout)")
	flag.Parse()

	if *rpcURL == "" || *reporter == "" || *startHeight == 0 {
		flag.Usage()
		log.Fatal("required: --rpc-url, --reporter, --start-height")
	}

	layerd := resolveLayerdPath(*layerdPath)
	rpc := NewRPCClient(*rpcURL)

	endHeight := *endHeightFlag
	if endHeight == 0 {
		var err error
		endHeight, err = rpc.LatestHeight()
		if err != nil {
			log.Fatalf("failed to get latest height: %v", err)
		}
	}
	if endHeight < *startHeight {
		log.Fatalf("end-height %d is before start-height %d", endHeight, *startHeight)
	}

	log.Printf("Scanning heights %d..%d for reporter %s via %s", *startHeight, endHeight, *reporter, rpc.baseURL)

	summary := Summary{
		Reporter:    *reporter,
		RPCURL:      rpc.baseURL,
		StartHeight: *startHeight,
		EndHeight:   endHeight,
		Results:     make([]ResultRow, 0),
	}

	sem := make(chan struct{}, maxConcurrentBlocks)
	var wg sync.WaitGroup
	var resultsMu sync.Mutex
	var scanned, missed, unknown, participated atomic.Int64

	for height := *startHeight; height <= endHeight; height++ {
		h := height
		if h == *startHeight || h == endHeight || h%1000 == 0 {
			log.Printf("progress: height %d / %d (scanned=%d missed=%d unknown=%d participated=%d)",
				h, endHeight, scanned.Load(), missed.Load(), unknown.Load(), participated.Load())
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(blockHeight uint64) {
			defer wg.Done()
			defer func() { <-sem }()

			events, err := rpc.AggregateEventsAtHeight(blockHeight)
			if err != nil {
				log.Printf("warning: block_results height %d: %v", blockHeight, err)
				return
			}

			localRows := make([]ResultRow, 0)
			for _, ev := range events {
				scanned.Add(1)
				status, err := checkParticipation(layerd, rpc.baseURL, *reporter, ev.QueryID, ev.TimestampMs)
				if err != nil {
					log.Printf("warning: participation check qid=%s ts=%d: %v (marking unknown_pruned)",
						ev.QueryID, ev.TimestampMs, err)
					status = statusUnknownPruned
				}

				switch status {
				case statusParticipated:
					participated.Add(1)
					continue
				case statusUnknownPruned:
					unknown.Add(1)
				case statusMissed:
					missed.Add(1)
				}

				row := ResultRow{
					Height:            blockHeight,
					TimestampMs:       ev.TimestampMs,
					TimestampRFC3339:  time.UnixMilli(int64(ev.TimestampMs)).UTC().Format(time.RFC3339Nano),
					QueryID:           ev.QueryID,
					QueryType:         ev.MicroReportType,
					QueryData:         ev.QueryData,
					AggregateValue:    ev.Value,
					AggregatePower:    ev.AggregatePower,
					MicroReportHeight: ev.MicroReportHeight,
					Status:            status,
				}
				if strings.EqualFold(ev.MicroReportType, "SpotPrice") {
					if pair, err := decodeSpotPriceAssetPair(ev.QueryData); err == nil {
						row.AssetPair = pair
					} else {
						log.Printf("warning: SpotPrice decode qid=%s: %v", ev.QueryID, err)
					}
				}
				localRows = append(localRows, row)
			}

			if len(localRows) > 0 {
				resultsMu.Lock()
				summary.Results = append(summary.Results, localRows...)
				resultsMu.Unlock()
			}
		}(h)
	}

	wg.Wait()

	summary.Scanned = int(scanned.Load())
	summary.Missed = int(missed.Load())
	summary.Unknown = int(unknown.Load())
	summary.Participated = int(participated.Load())

	log.Printf("done: scanned=%d missed=%d unknown_pruned=%d participated=%d",
		summary.Scanned, summary.Missed, summary.Unknown, summary.Participated)

	out, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		log.Fatalf("marshal output: %v", err)
	}
	out = append(out, '\n')

	if *outPath == "-" {
		if _, err := os.Stdout.Write(out); err != nil {
			log.Fatalf("write stdout: %v", err)
		}
		return
	}
	if err := os.WriteFile(*outPath, out, 0o644); err != nil {
		log.Fatalf("write %s: %v", *outPath, err)
	}
	log.Printf("wrote %s", *outPath)
}

func resolveLayerdPath(flagPath string) string {
	if flagPath != "" {
		return flagPath
	}
	if env := os.Getenv("LAYERD_PATH"); env != "" {
		return env
	}
	return "./layerd"
}

func NewRPCClient(rawURL string) *RPCClient {
	base := strings.TrimRight(strings.TrimSpace(rawURL), "/")
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}
	return &RPCClient{
		baseURL: base,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *RPCClient) makeRPCRequest(method string, params interface{}) ([]byte, error) {
	reqBody, err := json.Marshal(RPCRequest{
		Jsonrpc: "2.0",
		Method:  method,
		Id:      1,
		Params:  params,
	})
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Post(c.baseURL, "application/json", strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (c *RPCClient) LatestHeight() (uint64, error) {
	body, err := c.makeRPCRequest("block", nil)
	if err != nil {
		return 0, err
	}
	var resp BlockResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, err
	}
	if resp.Result.Block.Header.Height == "" {
		return 0, fmt.Errorf("empty height in block response")
	}
	return strconv.ParseUint(resp.Result.Block.Header.Height, 10, 64)
}

func (c *RPCClient) AggregateEventsAtHeight(height uint64) ([]AggregateEvent, error) {
	body, err := c.makeRPCRequest("block_results", map[string]interface{}{
		"height": fmt.Sprintf("%d", height),
	})
	if err != nil {
		return nil, err
	}
	var resp BlockResultsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	out := make([]AggregateEvent, 0)
	for _, ev := range resp.Result.FinalizeBlockEvents {
		if ev.Type != aggregateReportEventType {
			continue
		}
		agg, err := parseAggregateEvent(height, ev)
		if err != nil {
			log.Printf("warning: parse aggregate_report at height %d: %v", height, err)
			continue
		}
		out = append(out, agg)
	}
	return out, nil
}

func parseAggregateEvent(height uint64, ev Event) (AggregateEvent, error) {
	attrs := map[string]string{}
	for _, a := range ev.Attributes {
		attrs[a.Key] = a.Value
	}
	tsStr, ok := attrs["timestamp"]
	if !ok || tsStr == "" {
		return AggregateEvent{}, fmt.Errorf("missing timestamp")
	}
	ts, err := strconv.ParseUint(tsStr, 10, 64)
	if err != nil {
		return AggregateEvent{}, fmt.Errorf("bad timestamp %q: %w", tsStr, err)
	}
	qid := attrs["query_id"]
	if qid == "" {
		return AggregateEvent{}, fmt.Errorf("missing query_id")
	}
	return AggregateEvent{
		Height:            height,
		QueryID:           qid,
		QueryData:         attrs["query_data"],
		Value:             attrs["value"],
		AggregatePower:    attrs["aggregate_power"],
		MicroReportHeight: attrs["micro_report_height"],
		MicroReportType:   attrs["micro_report_type"],
		TimestampMs:       ts,
	}, nil
}

// checkParticipation returns missed | participated | unknown_pruned.
func checkParticipation(layerd, nodeURL, reporter, queryID string, timestampMs uint64) (string, error) {
	reports, err := fetchAllReportsByAggregate(layerd, nodeURL, queryID, timestampMs)
	if err != nil {
		// Aggregate missing or query failed — treat as pruned/unknown so we still surface the event.
		return statusUnknownPruned, err
	}
	if len(reports) == 0 {
		return statusUnknownPruned, nil
	}
	for _, r := range reports {
		if strings.EqualFold(r.Reporter, reporter) {
			return statusParticipated, nil
		}
	}
	return statusMissed, nil
}

func fetchAllReportsByAggregate(layerd, nodeURL, queryID string, timestampMs uint64) ([]MicroReport, error) {
	all := make([]MicroReport, 0)
	var pageKey string

	for {
		args := []string{
			"query", "oracle", "get-reports-by-aggregate",
			queryID, strconv.FormatUint(timestampMs, 10),
			"--node", nodeURL,
			"--output", "json",
			"--page-limit", strconv.Itoa(defaultPageLimit),
		}
		if pageKey != "" {
			args = append(args, "--page-key", pageKey)
		}

		cmd := exec.Command(layerd, args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}

		var resp ReportsByAggregateResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("unmarshal reports: %w; raw=%s", err, truncate(string(out), 400))
		}
		all = append(all, resp.MicroReports...)

		if resp.Pagination == nil || resp.Pagination.NextKey == "" {
			break
		}
		pageKey = resp.Pagination.NextKey
	}
	return all, nil
}

func decodeSpotPriceAssetPair(queryDataHex string) (string, error) {
	raw := strings.TrimPrefix(strings.TrimPrefix(queryDataHex, "0x"), "0X")
	data, err := hex.DecodeString(raw)
	if err != nil {
		return "", fmt.Errorf("hex decode query_data: %w", err)
	}

	stringType, err := abi.NewType("string", "string", nil)
	if err != nil {
		return "", err
	}
	bytesType, err := abi.NewType("bytes", "bytes", nil)
	if err != nil {
		return "", err
	}
	outer := abi.Arguments{
		{Type: stringType},
		{Type: bytesType},
	}
	unpacked, err := outer.Unpack(data)
	if err != nil {
		return "", fmt.Errorf("unpack query type: %w", err)
	}
	queryType, ok := unpacked[0].(string)
	if !ok {
		return "", fmt.Errorf("query type is not string")
	}
	if !strings.EqualFold(queryType, "SpotPrice") {
		return "", fmt.Errorf("query type is %q, not SpotPrice", queryType)
	}
	innerBytes, ok := unpacked[1].([]byte)
	if !ok {
		return "", fmt.Errorf("query args are not bytes")
	}

	inner := abi.Arguments{
		{Type: stringType, Name: "asset"},
		{Type: stringType, Name: "currency"},
	}
	args, err := inner.Unpack(innerBytes)
	if err != nil {
		return "", fmt.Errorf("unpack SpotPrice args: %w", err)
	}
	asset, ok1 := args[0].(string)
	currency, ok2 := args[1].(string)
	if !ok1 || !ok2 {
		return "", fmt.Errorf("SpotPrice args are not strings")
	}
	return strings.ToLower(asset) + "/" + strings.ToLower(currency), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
