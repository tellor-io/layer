package customquery

var StaticEndpointTemplateConfig = map[string]*EndpointTemplate{
	"coingecko": {
		URLTemplate: "https://api.coingecko.com/api/v3/simple/price?ids={coin_id}&vs_currencies=usd",
		Method:      "GET",
		Timeout:     5000,
	},
	"coinpaprika": {
		URLTemplate: "https://api.coinpaprika.com/v1/tickers/{coin_id}?quotes=USD",
		Method:      "GET",
		Timeout:     5000,
	},
	"curve": {
		URLTemplate: "https://prices.curve.fi/v1/usd_price/ethereum/{contract_address}",
		Method:      "GET",
		Timeout:     5000,
	},
	"crypto": {
		URLTemplate: "https://api.crypto.com/v2/public/get-ticker?instrument_name={instrument_name}",
		Method:      "GET",
		Timeout:     5000,
	},
}

var StaticQueriesConfig = map[string]*QueryConfig{
	"05cddb6b67074aa61fcbe1d2fd5924e028bb699b506267df28c88f7deac4edc6": {
		ID:                "05cddb6b67074aa61fcbe1d2fd5924e028bb699b506267df28c88f7deac4edc6",
		AggregationMethod: "median",
		MinResponses:      2,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"savings-dai", "usd"},
				Params: map[string]string{
					"coin_id": "dai",
				},
			},
			{
				EndpointType: "coinpaprika",
				ResponsePath: []string{"quotes", "USD", "price"},
				Params: map[string]string{
					"coin_id": "sdai-savings-dai",
				},
			},
			{
				EndpointType: "curve",
				ResponsePath: []string{"data", "usd_price"},
				Params: map[string]string{
					"contract_address": "0x83F20F44975D03b1b09e64809B757c47f942BEeA",
				},
			},
		},
	},
}
