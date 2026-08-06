package rpc

import "errors"

// ErrBlockNotFound means the height has not been produced yet (or was pruned).
var ErrBlockNotFound = errors.New("block not found")

// Attribute is a Tendermint/CometBFT event attribute.
type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Index bool   `json:"index"`
}

// Event is a Tendermint/CometBFT ABCI event.
type Event struct {
	Type       string      `json:"type"`
	Attributes []Attribute `json:"attributes"`
}

// TxResult holds per-tx ABCI results including events.
type TxResult struct {
	Code      uint32  `json:"code"`
	Codespace string  `json:"codespace"`
	Events    []Event `json:"events"`
}

// BlockHeader is the subset of the block header we need.
type BlockHeader struct {
	ChainID string `json:"chain_id"`
	Height  string `json:"height"`
	Time    string `json:"time"`
}

// Block is a minimal block payload.
type Block struct {
	Header BlockHeader `json:"header"`
}

// BlockResult is block + results for one height.
type BlockResult struct {
	Height              uint64
	Time                string
	ChainID             string
	FinalizeBlockEvents []Event
	TxsResults          []TxResult
}

type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	ID      int         `json:"id"`
	Params  interface{} `json:"params"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

type blockResponse struct {
	JSONRPC string       `json:"jsonrpc"`
	ID      int          `json:"id"`
	Error   *jsonRPCError `json:"error,omitempty"`
	Result  struct {
		Block Block `json:"block"`
	} `json:"result"`
}

type blockResultsResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Error   *jsonRPCError `json:"error,omitempty"`
	Result  struct {
		Height              string     `json:"height"`
		TxsResults          []TxResult `json:"txs_results"`
		FinalizeBlockEvents []Event    `json:"finalize_block_events"`
	} `json:"result"`
}

type statusResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Error   *jsonRPCError `json:"error,omitempty"`
	Result  struct {
		SyncInfo struct {
			LatestBlockHeight string `json:"latest_block_height"`
		} `json:"sync_info"`
	} `json:"result"`
}

type validatorsResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Error   *jsonRPCError `json:"error,omitempty"`
	Result  struct {
		Validators []struct {
			VotingPower string `json:"voting_power"`
		} `json:"validators"`
	} `json:"result"`
}
