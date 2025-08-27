package coinbase_rates_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/daemons/lib"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/coinbase_rates"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/testutil"
	"github.com/tellor-io/layer/daemons/testutil/constants"
	"github.com/tellor-io/layer/daemons/testutil/daemons/pricefeed"
)

const (
	BTCUSD_TICKER = "BTC-USD"
	ETHUSD_TICKER = "ETH-USD"
	TRBUSD_TICKER = "TRB-USD"
)

var (
	BtcExponentMap = map[string]int32{
		BTCUSD_TICKER: constants.BtcUsdExponent,
	}
	EthExponentMap = map[string]int32{
		ETHUSD_TICKER: constants.EthUsdExponent,
	}
	TrbExponentMap = map[string]int32{
		TRBUSD_TICKER: -6,
	}
	MultiExponentMap = map[string]int32{
		BTCUSD_TICKER: constants.BtcUsdExponent,
		ETHUSD_TICKER: constants.EthUsdExponent,
		TRBUSD_TICKER: -6,
	}
)

func TestCoinbaseRatesPriceFunction_Mixed(t *testing.T) {
	validResponseString := `{
		"data": {
			"currency": "USD",
			"rates": {
				"BTC": "0.00001",
				"ETH": "0.0003",
				"TRB": "0.011765"
			}
		}
	}`

	multiCurrencyResponse := `{
		"data": {
			"currency": "USD",
			"rates": {
				"BTC": "0.00001",
				"ETH": "0.0003",
				"TRB": "0.011765",
				"SOL": "0.005",
				"USDC": "1.0001"
			}
		}
	}`

	tests := map[string]struct {
		responseJsonString  string
		exponentMap         map[string]int32
		medianFunctionFails bool

		expectedPriceMap       map[string]uint64
		expectedUnavailableMap map[string]error
		expectedError          error
	}{
		"Unavailable - invalid response": {
			responseJsonString: `{"data":{"currency":"USD","rates":{"BTC":"0.00001",}}}`,
			exponentMap:        BtcExponentMap,
			expectedError:      errors.New("failed to decode Coinbase rates response: invalid character '}' looking for beginning of object key string"),
		},
		"Unavailable - invalid type in response: number instead of string": {
			responseJsonString: `{"data":{"currency":"USD","rates":{"BTC":0.00001}}}`,
			exponentMap:        BtcExponentMap,
			expectedError: errors.New("failed to decode Coinbase rates response: json: cannot unmarshal number into Go struct field " +
				"CoinbaseRatesData.data.rates of type string"),
		},
		"Unavailable - rate is zero": {
			responseJsonString: `{"data":{"currency":"USD","rates":{"BTC":"0"}}}`,
			exponentMap:        BtcExponentMap,
			expectedPriceMap:   make(map[string]uint64),
			expectedUnavailableMap: map[string]error{
				BTCUSD_TICKER: errors.New("zero rate for currency: BTC"),
			},
		},
		"Unavailable - rate is negative": {
			responseJsonString: `{"data":{"currency":"USD","rates":{"BTC":"-0.00001"}}}`,
			exponentMap:        BtcExponentMap,
			expectedPriceMap:   make(map[string]uint64),
			expectedUnavailableMap: map[string]error{
				BTCUSD_TICKER: errors.New("value underflows uint64"),
			},
		},
		"Unavailable - empty response": {
			responseJsonString: `{}`,
			exponentMap:        BtcExponentMap,
			expectedPriceMap:   make(map[string]uint64),
			expectedUnavailableMap: map[string]error{
				BTCUSD_TICKER: errors.New("rate not found for currency: BTC"),
			},
		},
		"Unavailable - missing currency in rates": {
			responseJsonString: `{"data":{"currency":"USD","rates":{"ETH":"0.0003"}}}`,
			exponentMap:        BtcExponentMap,
			expectedPriceMap:   make(map[string]uint64),
			expectedUnavailableMap: map[string]error{
				BTCUSD_TICKER: errors.New("rate not found for currency: BTC"),
			},
		},
		"Unavailable - non-USD quote currency": {
			responseJsonString: validResponseString,
			exponentMap: map[string]int32{
				"BTC-EUR": -5,
			},
			expectedPriceMap: make(map[string]uint64),
			expectedUnavailableMap: map[string]error{
				"BTC-EUR": errors.New("only USD quote currency is supported, got: EUR"),
			},
		},
		"Unavailable - invalid ticker format": {
			responseJsonString: validResponseString,
			exponentMap: map[string]int32{
				"BTCUSD": -5,
			},
			expectedPriceMap: make(map[string]uint64),
			expectedUnavailableMap: map[string]error{
				"BTCUSD": errors.New("invalid ticker format: BTCUSD"),
			},
		},
		"Unavailable - invalid rate string": {
			responseJsonString: `{"data":{"currency":"USD","rates":{"BTC":"not_a_number"}}}`,
			exponentMap:        BtcExponentMap,
			expectedPriceMap:   make(map[string]uint64),
			expectedUnavailableMap: map[string]error{
				BTCUSD_TICKER: errors.New("failed to parse rate for BTC: strconv.ParseFloat: parsing \"not_a_number\": invalid syntax"),
			},
		},
		"Failure - overflow due to massively negative exponent": {
			responseJsonString: validResponseString,
			exponentMap:        map[string]int32{BTCUSD_TICKER: -3000},
			expectedPriceMap:   make(map[string]uint64),
			expectedUnavailableMap: map[string]error{
				BTCUSD_TICKER: errors.New("value overflows uint64"),
			},
		},
		"Failure - medianization error": {
			responseJsonString:  validResponseString,
			exponentMap:         BtcExponentMap,
			medianFunctionFails: true,
			expectedPriceMap:    make(map[string]uint64),
			expectedUnavailableMap: map[string]error{
				BTCUSD_TICKER: testutil.ErrMedianization,
			},
		},
		"Success - BTC with negative exponent": {
			responseJsonString: validResponseString,
			exponentMap:        BtcExponentMap,
			expectedPriceMap: map[string]uint64{
				BTCUSD_TICKER: uint64(10_000_000_000),
			},
		},
		"Success - ETH with negative exponent": {
			responseJsonString: validResponseString,
			exponentMap:        EthExponentMap,
			expectedPriceMap: map[string]uint64{
				ETHUSD_TICKER: uint64(3_333_333_333),
			},
		},
		"Success - TRB with negative exponent": {
			responseJsonString: validResponseString,
			exponentMap:        TrbExponentMap,
			expectedPriceMap: map[string]uint64{
				TRBUSD_TICKER: uint64(84_997_875),
			},
		},
		"Success - multiple currencies": {
			responseJsonString: multiCurrencyResponse,
			exponentMap:        MultiExponentMap,
			expectedPriceMap: map[string]uint64{
				BTCUSD_TICKER: uint64(10_000_000_000),
				ETHUSD_TICKER: uint64(3_333_333_333),
				TRBUSD_TICKER: uint64(84_997_875),
			},
		},
		"Success - positive exponent": {
			responseJsonString: validResponseString,
			exponentMap: map[string]int32{
				BTCUSD_TICKER: 2,
			},
			expectedPriceMap: map[string]uint64{
				BTCUSD_TICKER: uint64(1_000),
			},
		},
		"Success - zero exponent": {
			responseJsonString: validResponseString,
			exponentMap: map[string]int32{
				BTCUSD_TICKER: 0,
			},
			expectedPriceMap: map[string]uint64{
				BTCUSD_TICKER: uint64(100_000),
			},
		},
		"Success - very small rate (large price)": {
			responseJsonString: `{"data":{"currency":"USD","rates":{"BTC":"0.000001"}}}`,
			exponentMap:        BtcExponentMap,
			expectedPriceMap: map[string]uint64{
				BTCUSD_TICKER: uint64(100_000_000_000),
			},
		},
		"Success - rate close to 1": {
			responseJsonString: `{"data":{"currency":"USD","rates":{"BTC":"0.999"}}}`,
			exponentMap:        BtcExponentMap,
			expectedPriceMap: map[string]uint64{
				BTCUSD_TICKER: uint64(100_100),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			response := testutil.CreateResponseFromJson(tc.responseJsonString)

			var prices map[string]uint64
			var unavailable map[string]error
			var err error
			if tc.medianFunctionFails {
				prices, unavailable, err = coinbase_rates.CoinbaseRatesPriceFunction(response, tc.exponentMap, testutil.MedianErr)
			} else {
				prices, unavailable, err = coinbase_rates.CoinbaseRatesPriceFunction(response, tc.exponentMap, lib.Median[uint64])
			}

			if tc.expectedError != nil {
				require.EqualError(t, err, tc.expectedError.Error())
				require.Nil(t, prices)
				require.Nil(t, unavailable)
			} else {
				require.Equal(t, tc.expectedPriceMap, prices)
				pricefeed.ErrorMapsEqual(t, tc.expectedUnavailableMap, unavailable)
				require.NoError(t, err)
			}
		})
	}
}
