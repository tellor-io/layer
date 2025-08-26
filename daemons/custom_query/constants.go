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
	"coinmarketcap": {
		URLTemplate: "https://pro-api.coinmarketcap.com/v1/cryptocurrency/quotes/latest?id={id}",
		Method:      "GET",
		Timeout:     5000,
		ApiKey:      "${CMC_PRO_API_KEY}",
		Headers: map[string]string{
			"Accept":            "application/json",
			"X-CMC_PRO_API_KEY": "api_key",
		},
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
	"c444759b83c7bb0f6694306e1f719e65679d48ad754a31d3a366856becf1e71e": {
		ID:                "c444759b83c7bb0f6694306e1f719e65679d48ad754a31d3a366856becf1e71e",
		AggregationMethod: "median",
		MinResponses:      2,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"ignition-fbtc", "usd"},
				Params: map[string]string{
					"coin_id": "ignition-fbtc",
				},
			},
			{
				EndpointType: "coinpaprika",
				ResponsePath: []string{"quotes", "USD", "price"},
				Params: map[string]string{
					"coin_id": "fbtc-ignition-fbtc",
				},
			},
			{
				EndpointType: "coinmarketcap",
				ResponsePath: []string{"data", "32306", "quote", "USD", "price"},
				Params: map[string]string{
					// "symbol": "FBTC",
					"id": "32306",
				},
			},
		},
	},
	"e010d752f28dcd2804004d0b57ab1bdc4eca092895d49160204120af11d15f3e": {
		ID:                "e010d752f28dcd2804004d0b57ab1bdc4eca092895d49160204120af11d15f3e",
		AggregationMethod: "median",
		MinResponses:      2,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"noble-dollar-usdn", "usd"},
				Params: map[string]string{
					"coin_id": "noble-dollar-usdn",
				},
			},
			{
				EndpointType: "coinmarketcap",
				ResponsePath: []string{"data", "36538", "quote", "USD", "price"},
				Params: map[string]string{
					// "symbol": "USDN",
					"id": "36538",
				},
			},
		},
	},
	"59ae85cec665c779f18255dd4f3d97821e6a122691ee070b9a26888bc2a0e45a": {
		ID:                "59ae85cec665c779f18255dd4f3d97821e6a122691ee070b9a26888bc2a0e45a",
		AggregationMethod: "median",
		MinResponses:      2,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"susds", "usd"},
				Params: map[string]string{
					"coin_id": "susds",
				},
			},
		},
	},
	"35155b44678db9e9e021c2cf49dd20c31b49e03415325c2beffb5221cf63882d": {
		ID:                "35155b44678db9e9e021c2cf49dd20c31b49e03415325c2beffb5221cf63882d",
		AggregationMethod: "median",
		MinResponses:      1,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"yieldfi-ytoken", "usd"},
				Params: map[string]string{
					"coin_id": "yieldfi-ytoken",
				},
			},
		},
	},
	"03731257e35c49e44b267640126358e5decebdd8f18b5e8f229542ec86e318cf": {
		ID:                "03731257e35c49e44b267640126358e5decebdd8f18b5e8f229542ec86e318cf",
		AggregationMethod: "median",
		MinResponses:      2,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"ethena-staked-usde", "usd"},
				Params: map[string]string{
					"coin_id": "ethena-staked-usde",
				},
			},
			{
				EndpointType: "coinmarketcap",
				ResponsePath: []string{"data", "29471", "quote", "USD", "price"},
				Params: map[string]string{
					// "symbol": "SUSDE",
					"id": "29471",
				},
			},
		},
	},
	"76b504e33305a63a3b80686c0b7bb99e7697466927ba78e224728e80bfaaa0be": {
		ID:                "76b504e33305a63a3b80686c0b7bb99e7697466927ba78e224728e80bfaaa0be",
		AggregationMethod: "median",
		MinResponses:      2,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"tbtc", "usd"},
				Params: map[string]string{
					"coin_id": "tbtc",
				},
			},
			{
				EndpointType: "coinmarketcap",
				ResponsePath: []string{"data", "26133", "quote", "USD", "price"},
				Params: map[string]string{
					// "symbol": "TBTC",
					"id": "26133",
				},
			},
		},
	},
	"0bc2d41117ae8779da7623ee76a109c88b84b9bf4d9b404524df04f7d0ca4ca7": {
		ID:                "0bc2d41117ae8779da7623ee76a109c88b84b9bf4d9b404524df04f7d0ca4ca7",
		AggregationMethod: "median",
		MinResponses:      2,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"rocket-pool-eth", "usd"},
				Params: map[string]string{
					"coin_id": "rocket-pool-eth",
				},
			},
			{
				EndpointType: "coinmarketcap",
				ResponsePath: []string{"data", "15060", "quote", "USD", "price"},
				Params: map[string]string{
					// "symbol": "RETH",
					"id": "15060",
				},
			},
		},
	},
	"1962cde2f19178fe2bb2229e78a6d386e6406979edc7b9a1966d89d83b3ebf2e": {
		ID:                "1962cde2f19178fe2bb2229e78a6d386e6406979edc7b9a1966d89d83b3ebf2e",
		AggregationMethod: "median",
		MinResponses:      2,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"wrapped-steth", "usd"},
				Params: map[string]string{
					"coin_id": "wrapped-steth",
				},
			},
			{
				EndpointType: "coinmarketcap",
				ResponsePath: []string{"data", "12409", "quote", "USD", "price"},
				Params: map[string]string{
					// "symbol": "WSTETH",
					"id": "12409",
				},
			},
		},
	},
	"d62f132d9d04dde6e223d4366c48b47cd9f90228acdc6fa755dab93266db5176": {
		ID:                "d62f132d9d04dde6e223d4366c48b47cd9f90228acdc6fa755dab93266db5176",
		AggregationMethod: "median",
		MinResponses:      2,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"lrt-squared", "usd"},
				Params: map[string]string{
					"coin_id": "lrt-squared",
				},
			},
			{
				EndpointType: "coinmarketcap",
				ResponsePath: []string{"data", "33695", "quote", "USD", "price"},
				Params: map[string]string{
					"id": "33695",
					// "symbol": "KING",
				},
			},
		},
	},
}
