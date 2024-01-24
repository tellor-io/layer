package types

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
)

const (
	MaxPriceChangePpm = uint32(10_000)
)

type MarketParam struct {
	// Unique, sequentially-generated value.
	Id uint32
	// The human-readable name of the market pair (e.g. `BTC-USD`).
	Pair string
	// Static value. The exponent of the price.
	// For example if `Exponent == -5` then a `Value` of `1,000,000,000`
	// represents “$10,000`. Therefore `10 ^ Exponent` represents the smallest
	// price step (in dollars) that can be recorded.
	Exponent int32
	// The minimum number of exchanges that should be reporting a live price for
	// a price update to be considered valid.
	MinExchanges uint32
	// The minimum allowable change in `price` value that would cause a price
	// update on the network. Measured as `1e-6` (parts per million).
	MinPriceChangePpm uint32
	// A string of json that encodes the configuration for resolving the price
	// of this market on various exchanges.
	ExchangeConfigJson string
	// query data representation of the market for layer
	QueryData string
}

// Validate checks that the MarketParam is valid.
func (mp *MarketParam) Validate() error {
	// Validate pair.
	if mp.Pair == "" {
		return fmt.Errorf("Invalid input: Pair cannot be empty")
	}

	if mp.MinExchanges == 0 {
		return fmt.Errorf("Min exchanges must be greater than zero")
	}

	// Validate min price change.
	if mp.MinPriceChangePpm == 0 || mp.MinPriceChangePpm >= MaxPriceChangePpm {
		return fmt.Errorf(
			"Invalid input Min price change in parts-per-million must be greater than 0 and less than %d",
			MaxPriceChangePpm)
	}

	if err := IsValidJSON(mp.ExchangeConfigJson); err != nil {
		return fmt.Errorf(
			"Invalid input: ExchangeConfigJson string is not valid: err=%v, input=%v",
			err,
			mp.ExchangeConfigJson,
		)
	}

	if mp.QueryData == "" {
		return fmt.Errorf("Invalid input: QueryData cannot be empty")
	}
	// try to decode query data from hex to bytes if this fails then return error
	_, err := hex.DecodeString(mp.QueryData)
	if err != nil {
		return fmt.Errorf("Invalid input: QueryData is not valid hex: %v", err)
	}

	return nil
}

// IsValidJSON checks if a JSON string is well-formed.
func IsValidJSON(str string) error {
	var js map[string]interface{}
	err := json.Unmarshal([]byte(str), &js)
	if err != nil {
		return err
	}
	return nil
}
