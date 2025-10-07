package customquery

import "github.com/tellor-io/layer/daemons/exchange_common"

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
	"coinbase": {
		URLTemplate: "https://api.coinbase.com/v2/prices/{currency_pair}/spot",
		Method:      "GET",
		Timeout:     5000,
	},
	"osmosis": {
		URLTemplate: "https://lcd.osmosis.zone/osmosis/gamm/v1beta1/pools/{pool_id}",
		Method:      "GET",
		Timeout:     5000,
	},
	"uniswapV4ethereum": {
		// docs: https://docs.uniswap.org/api/subgraph/overview
		URLTemplate: "https://gateway.thegraph.com/api/{api_key}/subgraphs/id/DiYPVdygkfjDWhbxGSqAQxwBKmfKnkWQojqeM2rkLb3G",
		Query:       `{"query": "{ token(id: \"{token_address}\") { derivedETH } }"}`,
		Method:      "POST",
		Timeout:     5000,
		Headers:     map[string]string{"Content-Type": "application/json"},
		ApiKey:      "${SUBGRAPH_API_KEY}",
	},
	"uniswapV3ethereum": {
		// docs: https://docs.uniswap.org/api/subgraph/overview
		URLTemplate: "https://gateway.thegraph.com/api/{api_key}/subgraphs/id/5zvR82QoaXYFyDEKLZ9t6v9adgnptxYpKpSbxtgVENFV",
		Query:       `{"query": "{ token(id: \"{token_address}\") { derivedETH } }"}`,
		Method:      "POST",
		Timeout:     5000,
		Headers:     map[string]string{"Content-Type": "application/json"},
		ApiKey:      "${SUBGRAPH_API_KEY}",
	},
	"sushiswapKatana": {
		// docs: https://docs.sushi.com/api/examples/pricing
		URLTemplate: "https://api.sushi.com/price/v1/747474",
		Method:      "GET",
		Timeout:     5000,
	},
}

var StaticRPCEndpointTemplateConfig = map[string]*RPCEndpointTemplate{
	"ethereum": {
		URLs: []string{
			"https://mainnet.infura.io/v3/${INFURA_API_KEY}",
			"https://eth-mainnet.alchemyapi.io/v2/${ALCHEMY_API_KEY}",
			"https://rpc.ankr.com/eth",
		},
	},
}

var StaticQueriesConfig = map[string]*QueryConfig{
	"05cddb6b67074aa61fcbe1d2fd5924e028bb699b506267df28c88f7deac4edc6": {
		ID:                "05cddb6b67074aa61fcbe1d2fd5924e028bb699b506267df28c88f7deac4edc6",
		AggregationMethod: "median",
		MaxSpreadPercent:  1.0,
		MinResponses:      2,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"savings-dai", "usd"},
				Params: map[string]string{
					"coin_id": "savings-dai",
				},
				MarketId: "SDAI-USD",
			},
			{
				EndpointType: "coinpaprika",
				ResponsePath: []string{"quotes", "USD", "price"},
				Params: map[string]string{
					"coin_id": "sdai-savings-dai",
				},
				MarketId: "SDAI-USD",
			},
			{
				EndpointType: "curve",
				ResponsePath: []string{"data", "usd_price"},
				Params: map[string]string{
					"contract_address": "0x83F20F44975D03b1b09e64809B757c47f942BEeA",
				},
				MarketId: "SDAI-USD",
			},
		},
	},
	"c444759b83c7bb0f6694306e1f719e65679d48ad754a31d3a366856becf1e71e": {
		ID:                "c444759b83c7bb0f6694306e1f719e65679d48ad754a31d3a366856becf1e71e",
		AggregationMethod: "median",
		MaxSpreadPercent:  1.0,
		MinResponses:      2,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"ignition-fbtc", "usd"},
				Params: map[string]string{
					"coin_id": "ignition-fbtc",
				},
				MarketId: "FBTC-USD",
			},
			{
				EndpointType: "coinpaprika",
				ResponsePath: []string{"quotes", "USD", "price"},
				Params: map[string]string{
					"coin_id": "fbtc-ignition-fbtc",
				},
				MarketId: "FBTC-USD",
			},
			{
				EndpointType: "coinmarketcap",
				ResponsePath: []string{"data", "32306", "quote", "USD", "price"},
				Params: map[string]string{
					// "symbol": "FBTC",
					"id": "32306",
				},
				MarketId: "FBTC-USD",
			},
		},
	},
	"e010d752f28dcd2804004d0b57ab1bdc4eca092895d49160204120af11d15f3e": {
		ID:                "e010d752f28dcd2804004d0b57ab1bdc4eca092895d49160204120af11d15f3e",
		AggregationMethod: "median",
		MinResponses:      1,
		MaxSpreadPercent:  0.0,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"noble-dollar-usdn", "usd"},
				Params: map[string]string{
					"coin_id": "noble-dollar-usdn",
				},
				MarketId: "USDN-USD",
			},
			{
				EndpointType: "coinmarketcap",
				ResponsePath: []string{"data", "36538", "quote", "USD", "price"},
				Params: map[string]string{
					// "symbol": "USDN",
					"id": "36538",
				},
				MarketId: "USDN-USD",
			},
		},
	},
	"59ae85cec665c779f18255dd4f3d97821e6a122691ee070b9a26888bc2a0e45a": {
		ID:                "59ae85cec665c779f18255dd4f3d97821e6a122691ee070b9a26888bc2a0e45a",
		AggregationMethod: "median",
		MaxSpreadPercent:  0.0,
		MinResponses:      1,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"susds", "usd"},
				Params: map[string]string{
					"coin_id": "susds",
				},
				MarketId: "SUSDS-USD",
			},
		},
	},
	"35155b44678db9e9e021c2cf49dd20c31b49e03415325c2beffb5221cf63882d": {
		ID:                "35155b44678db9e9e021c2cf49dd20c31b49e03415325c2beffb5221cf63882d",
		AggregationMethod: "median",
		MaxSpreadPercent:  0.0,
		MinResponses:      1,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"yieldfi-ytoken", "usd"},
				Params: map[string]string{
					"coin_id": "yieldfi-ytoken",
				},
				MarketId: "YTOKEN-USD",
			},
			{
				EndpointType: "sushiswapKatana",
				ResponsePath: []string{"0x4772d2e014f9fc3a820c444e3313968e9a5c8121"},
				MarketId:     "YTOKEN-USD",
			},
		},
	},
	"03731257e35c49e44b267640126358e5decebdd8f18b5e8f229542ec86e318cf": {
		ID:                "03731257e35c49e44b267640126358e5decebdd8f18b5e8f229542ec86e318cf",
		AggregationMethod: "median",
		MaxSpreadPercent:  0.0,
		MinResponses:      1,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"ethena-staked-usde", "usd"},
				Params: map[string]string{
					"coin_id": "ethena-staked-usde",
				},
				MarketId: "USDE-USD",
			},
			{
				EndpointType: "coinmarketcap",
				ResponsePath: []string{"data", "29471", "quote", "USD", "price"},
				Params: map[string]string{
					// "symbol": "SUSDE",
					"id": "29471",
				},
				MarketId: "USDE-USD",
			},
		},
	},
	"76b504e33305a63a3b80686c0b7bb99e7697466927ba78e224728e80bfaaa0be": {
		ID:                "76b504e33305a63a3b80686c0b7bb99e7697466927ba78e224728e80bfaaa0be",
		AggregationMethod: "median",
		MaxSpreadPercent:  1.0,
		MinResponses:      2,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"tbtc", "usd"},
				Params: map[string]string{
					"coin_id": "tbtc",
				},
				MarketId: "TBTC-USD",
			},
			{
				EndpointType: "coinmarketcap",
				ResponsePath: []string{"data", "26133", "quote", "USD", "price"},
				Params: map[string]string{
					// "symbol": "TBTC",
					"id": "26133",
				},
				MarketId: "TBTC-USD",
			},
			{
				EndpointType: "coinbase",
				ResponsePath: []string{"data", "amount"},
				Params: map[string]string{
					"currency_pair": "TBTC-USD",
				},
				MarketId: "TBTC-USD",
			},
		},
	},
	"0bc2d41117ae8779da7623ee76a109c88b84b9bf4d9b404524df04f7d0ca4ca7": {
		ID:                "0bc2d41117ae8779da7623ee76a109c88b84b9bf4d9b404524df04f7d0ca4ca7",
		AggregationMethod: "median",
		MaxSpreadPercent:  1.0,
		MinResponses:      2,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"rocket-pool-eth", "usd"},
				Params: map[string]string{
					"coin_id": "rocket-pool-eth",
				},
				MarketId: "RETH-USD",
			},
			{
				EndpointType: "coinmarketcap",
				ResponsePath: []string{"data", "15060", "quote", "USD", "price"},
				Params: map[string]string{
					// "symbol": "RETH",
					"id": "15060",
				},
				MarketId: "RETH-USD",
			},
			{
				EndpointType: "contract",
				Handler:      "reth_handler",
				Chain:        "ethereum",
				MarketId:     "RETH-USD",
			},
		},
	},
	"1962cde2f19178fe2bb2229e78a6d386e6406979edc7b9a1966d89d83b3ebf2e": {
		ID:                "1962cde2f19178fe2bb2229e78a6d386e6406979edc7b9a1966d89d83b3ebf2e",
		AggregationMethod: "median",
		MaxSpreadPercent:  1.0,
		MinResponses:      2,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"wrapped-steth", "usd"},
				Params: map[string]string{
					"coin_id": "wrapped-steth",
				},
				MarketId: "WSTETH-USD",
			},
			{
				EndpointType: "coinmarketcap",
				ResponsePath: []string{"data", "12409", "quote", "USD", "price"},
				Params: map[string]string{
					// "symbol": "WSTETH",
					"id": "12409",
				},
				MarketId: "WSTETH-USD",
			},
			{
				EndpointType: "contract",
				Handler:      "wsteth_handler",
				Chain:        "ethereum",
				MarketId:     "WSTETH-USD",
			},
		},
	},
	"d62f132d9d04dde6e223d4366c48b47cd9f90228acdc6fa755dab93266db5176": {
		ID:                "d62f132d9d04dde6e223d4366c48b47cd9f90228acdc6fa755dab93266db5176",
		AggregationMethod: "median",
		MaxSpreadPercent:  1.0,
		MinResponses:      2,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"lrt-squared", "usd"},
				Params: map[string]string{
					"coin_id": "lrt-squared",
				},
				MarketId: "KING-USD",
			},
			{
				EndpointType: "coinmarketcap",
				ResponsePath: []string{"data", "33695", "quote", "USD", "price"},
				Params: map[string]string{
					"id": "33695",
					// "symbol": "KING",
				},
				MarketId: "KING-USD",
			},
			{
				EndpointType: "uniswapV4ethereum",
				ResponsePath: []string{"data", "token", "derivedETH"},
				Params:       map[string]string{"token_address": "0x8f08b70456eb22f6109f57b8fafe862ed28e6040"},
				UsdViaID:     exchange_common.ETHUSD_ID,
				Invert:       false,
				MarketId:     "KING-USD",
			},
			{
				EndpointType: "uniswapV3ethereum",
				ResponsePath: []string{"data", "token", "derivedETH"},
				Params:       map[string]string{"token_address": "0x8f08b70456eb22f6109f57b8fafe862ed28e6040"},
				UsdViaID:     exchange_common.ETHUSD_ID,
				Invert:       false,
				MarketId:     "KING-USD",
			},
		},
	},
	"611fd0e88850bf0cc036d96d04d47605c90b993485c2971e022b5751bbb04f23": {
		ID:                "611fd0e88850bf0cc036d96d04d47605c90b993485c2971e022b5751bbb04f23",
		AggregationMethod: "median",
		MaxSpreadPercent:  0.0,
		MinResponses:      1,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"stride-staked-atom", "usd"},
				Params: map[string]string{
					"coin_id": "stride-staked-atom",
				},
				MarketId: "ATOM-USD",
			},
			{
				EndpointType: "coinmarketcap",
				ResponsePath: []string{"data", "21686", "quote", "USD", "price"},
				Params: map[string]string{
					"id": "21686",
					// "symbol": "stATOM",
				},
				MarketId: "ATOM-USD",
			},
			{
				EndpointType: "osmosis",
				Handler:      "osmosis_pool_price_handler",
				ResponsePath: []string{"pool"},
				Params: map[string]string{
					"pool_id": "1136",
				},
				UsdViaID: exchange_common.ATOMUSD_ID,
				Invert:   false,
				MarketId: "ATOM-USD",
			},
		},
	},
	"91513b15db3cef441d52058b24412957f9cc8645c53aecf39446ac9b0d2dcca4": {
		ID:                "91513b15db3cef441d52058b24412957f9cc8645c53aecf39446ac9b0d2dcca4",
		AggregationMethod: "median",
		MaxSpreadPercent:  0.0,
		MinResponses:      1,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "combined",
				Handler:      "vyusd_price",
				CombinedSources: map[string]string{
					"ethereum":      "contract:ethereum",
					"sushiswap_api": "rpc:sushiswapKatana",
					"coingecko_api": "rpc:coingecko",
				},
				CombinedConfig: map[string]any{
					"sushiswap_api_response_path": []string{"0x4772d2e014f9fc3a820c444e3313968e9a5c8121"},
					"coingecko_api_response_path": []string{"yieldfi-ytoken", "usd"},
					"coingecko_api_params": map[string]string{
						"coin_id": "yieldfi-ytoken",
					},
				},
				MarketId: "VYUSD-USD",
			},
		},
	},
	"187f74d310dc494e6efd928107713d4229cd319c2cf300224de02776090809f1": {
		ID:                "187f74d310dc494e6efd928107713d4229cd319c2cf300224de02776090809f1",
		AggregationMethod: "median",
		MaxSpreadPercent:  0.0,
		MinResponses:      1,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"staked-usn", "usd"},
				Params: map[string]string{
					"coin_id": "staked-usn",
				},
				MarketId: "SUSN-USD",
			},
		},
	},
	"ab30caa3e7827a27c153063bce02c0b260b29c0c164040c003f0f9ec66002510": {
		ID:                "ab30caa3e7827a27c153063bce02c0b260b29c0c164040c003f0f9ec66002510",
		AggregationMethod: "median",
		MaxSpreadPercent:  1.0,
		MinResponses:      2,
		ResponseType:      "ufixed256x18",
		Endpoints: []EndpointConfig{
			{
				EndpointType: "coingecko",
				ResponsePath: []string{"staked-frax-usd", "usd"},
				Params: map[string]string{
					"coin_id": "staked-frax-usd",
				},
				MarketId: "SFRXUSD-USD",
			},
			{
				EndpointType: "coinpaprika",
				ResponsePath: []string{"quotes", "USD", "price"},
				Params: map[string]string{
					"coin_id": "sfrxusd-staked-frax-usd",
				},
				MarketId: "SFRXUSD-USD",
			},
			{
				EndpointType: "coinmarketcap",
				ResponsePath: []string{"data", "36038", "quote", "USD", "price"},
				Params: map[string]string{
					// "symbol": "SFRXUSD",
					"id": "36038",
				},
				MarketId: "SFRXUSD-USD",
			},
			{
				EndpointType: "curve",
				ResponsePath: []string{"data", "usd_price"},
				Params: map[string]string{
					"contract_address": "0xcf62F905562626CfcDD2261162a51fd02Fc9c5b6",
				},
				MarketId: "SFRXUSD-USD",
			},
		},
	},
}
