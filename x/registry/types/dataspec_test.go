package types

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestTupleListEncoding(t *testing.T) {
	t.Parallel()
	dataSpec := DataSpec{
		DocumentHash:      "",
		ResponseValueType: "uint256",
		AbiComponents: []*ABIComponent{
			{Name: "metric", FieldType: "string"},
			{Name: "currency", FieldType: "string"},
			{
				Name: "collection", FieldType: "tuple[]",
				NestedComponent: []*ABIComponent{
					{Name: "chainName", FieldType: "string"},
					{Name: "collectionAddress", FieldType: "address"},
				},
			},
			{
				Name: "tokens", FieldType: "tuple[]",
				NestedComponent: []*ABIComponent{
					{Name: "chainName", FieldType: "string"},
					{Name: "tokenName", FieldType: "string"},
					{Name: "tokenAddress", FieldType: "address"},
				},
			},
		},
	}
	// Test encoding of data spec
	encodedDataSpec, err := dataSpec.EncodeData("MimicryMacroMarketMashup",
		`[
		"market-cap",
		"usd",
		[
			["ethereum-mainnet","0x50f5474724e0Ee42D9a4e711ccFB275809Fd6d4a"],
			["ethereum-mainnet","0xF87E31492Faf9A91B02Ee0dEAAd50d51d56D5d4d"],
			["ethereum-mainnet","0x34d85c9CDeB23FA97cb08333b511ac86E1C4E258"]
		],
		[
			["ethereum-mainnet","sand","0x3845badAde8e6dFF049820680d1F14bD3903a5d0"],
			["ethereum-mainnet","mana","0x0F5D2fB29fb7d3CFeE444a200298f468908cC942"],
			["ethereum-mainnet","ape","0x4d224452801ACEd8B2F0aebE155379bb5D594381"]
		]
	]`)
	require.NoError(t, err)
	expectedResult := hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000184d696d696372794d6163726f4d61726b65744d617368757000000000000000000000000000000000000000000000000000000000000000000000000000000620000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000000000000000000000000000000000000000a6d61726b65742d63617000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000375736400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000160000000000000000000000000000000000000000000000000000000000000004000000000000000000000000050f5474724e0ee42d9a4e711ccfb275809fd6d4a0000000000000000000000000000000000000000000000000000000000000010657468657265756d2d6d61696e6e6574000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000040000000000000000000000000f87e31492faf9a91b02ee0deaad50d51d56d5d4d0000000000000000000000000000000000000000000000000000000000000010657468657265756d2d6d61696e6e657400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000034d85c9cdeb23fa97cb08333b511ac86e1c4e2580000000000000000000000000000000000000000000000000000000000000010657468657265756d2d6d61696e6e6574000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000001400000000000000000000000000000000000000000000000000000000000000220000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000003845badade8e6dff049820680d1f14bd3903a5d00000000000000000000000000000000000000000000000000000000000000010657468657265756d2d6d61696e6e657400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000473616e6400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000f5d2fb29fb7d3cfee444a200298f468908cc9420000000000000000000000000000000000000000000000000000000000000010657468657265756d2d6d61696e6e65740000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000046d616e6100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000004d224452801aced8b2f0aebe155379bb5d5943810000000000000000000000000000000000000000000000000000000000000010657468657265756d2d6d61696e6e65740000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000036170650000000000000000000000000000000000000000000000000000000000")
	require.Equal(t, expectedResult, encodedDataSpec)
	queryType, resultBytes, err := DecodeQueryType(encodedDataSpec)
	require.Equal(t, queryType, "MimicryMacroMarketMashup")
	require.NoError(t, err)
	res, err := DecodeParamtypes(resultBytes, dataSpec.AbiComponents)
	expectedDecodedResult := `["market-cap","usd",[{"chainName":"ethereum-mainnet","collectionAddress":"0x50f5474724e0ee42d9a4e711ccfb275809fd6d4a"},{"chainName":"ethereum-mainnet","collectionAddress":"0xf87e31492faf9a91b02ee0deaad50d51d56d5d4d"},{"chainName":"ethereum-mainnet","collectionAddress":"0x34d85c9cdeb23fa97cb08333b511ac86e1c4e258"}],[{"chainName":"ethereum-mainnet","tokenName":"sand","tokenAddress":"0x3845badade8e6dff049820680d1f14bd3903a5d0"},{"chainName":"ethereum-mainnet","tokenName":"mana","tokenAddress":"0x0f5d2fb29fb7d3cfee444a200298f468908cc942"},{"chainName":"ethereum-mainnet","tokenName":"ape","tokenAddress":"0x4d224452801aced8b2f0aebe155379bb5d594381"}]]`
	require.Equal(t, res, expectedDecodedResult)
	require.NoError(t, err)
}

func TestTupleDataSpecEncoding(t *testing.T) {
	t.Parallel()
	dataspec := DataSpec{
		DocumentHash:      "",
		ResponseValueType: "uint256",
		AbiComponents: []*ABIComponent{
			{
				Name: "tupletest", FieldType: "tuple",
				NestedComponent: []*ABIComponent{
					{Name: "num1", FieldType: "uint256"},
					{Name: "num2", FieldType: "uint256"},
					{Name: "text", FieldType: "string"},
				},
			},
		},
	}
	// Test encoding of data spec
	encodedDataSpec, err := dataspec.EncodeData("Tuplenoarray", `[["123","456","hello"]]`)
	require.NoError(t, err)
	expectedResult := hexutil.MustDecode("0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000c5475706c656e6f6172726179000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000007b00000000000000000000000000000000000000000000000000000000000001c80000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000568656c6c6f000000000000000000000000000000000000000000000000000000")
	require.Equal(t, expectedResult, encodedDataSpec)
	queryType, resultBytes, err := DecodeQueryType(expectedResult)
	require.Equal(t, queryType, "Tuplenoarray")
	require.NoError(t, err)
	res, err := DecodeParamtypes(resultBytes, dataspec.AbiComponents)
	expectedDecodedResult := `[{"num1":123,"num2":456,"text":"hello"}]`
	require.Equal(t, res, expectedDecodedResult)
	require.NoError(t, err)
}

func TestMixedEncoding(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name                  string
		dataSpec              DataSpec
		queryType             string
		datafields            string
		expectedDecodedResult string
		expectedEncodedResult string
	}{
		{
			"Test case eth/usd",
			DataSpec{
				ResponseValueType: "uint256",
				AbiComponents: []*ABIComponent{
					{Name: "asset", FieldType: "string"},
					{Name: "currency", FieldType: "string"},
				},
				AggregationMethod: "weighted-median",
			},
			"SpotPrice",
			`["eth","usd"]`,
			`["eth","usd"]`,
			"0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000",
		},
		{
			"Test case btc/usd",
			DataSpec{
				ResponseValueType: "uint256",
				AbiComponents: []*ABIComponent{
					{Name: "asset", FieldType: "string"},
					{Name: "currency", FieldType: "string"},
				},
				AggregationMethod: "weighted-median",
			},
			"SpotPrice",
			`["btc","usd"]`,
			`["btc","usd"]`,
			"0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003627463000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000",
		},
		{
			"Test case trb/usd",
			DataSpec{
				ResponseValueType: "uint256",
				AbiComponents: []*ABIComponent{
					{Name: "asset", FieldType: "string"},
					{Name: "currency", FieldType: "string"},
				},
				AggregationMethod: "weighted-median",
			},
			"SpotPrice",
			`["trb","usd"]`,
			`["trb","usd"]`,
			"0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000",
		},
		{
			"Test case BTCbalance",
			DataSpec{
				ResponseValueType: "uint256",
				AbiComponents: []*ABIComponent{
					{Name: "address", FieldType: "string"},
					{Name: "timestamp", FieldType: "uint256"},
				},
				AggregationMethod: "weighted-mode",
			},
			"BTCbalance",
			`["3Cyd2ExaAEoTzmLNyixJxBsJ4X16t1VePc","1705954706"]`,
			`["3Cyd2ExaAEoTzmLNyixJxBsJ4X16t1VePc",1705954706]`,
			"0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000a42544362616c616e63650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000065aecd920000000000000000000000000000000000000000000000000000000000000022334379643245786141456f547a6d4c4e7969784a7842734a34583136743156655063000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"Test case CrossChainBalance",
			DataSpec{
				ResponseValueType: "uint256",
				AbiComponents: []*ABIComponent{
					{Name: "chainId", FieldType: "uint256"},
					{Name: "contractAddress", FieldType: "address"},
					{Name: "timestamp", FieldType: "uint256"},
				},
				AggregationMethod: "weighted-mode",
			},
			"CrossChainBalance",
			`["1","0x88dF592F8eb5D7Bd38bFeF7dEb0fBc02cf3778a0","15998590"]`,
			`[1,"0x88df592f8eb5d7bd38bfef7deb0fbc02cf3778a0",15998590]`,
			"0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000001143726f7373436861696e42616c616e63650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000100000000000000000000000088df592f8eb5d7bd38bfef7deb0fbc02cf3778a00000000000000000000000000000000000000000000000000000000000f41e7e",
		},
		{
			"Test case AmpleforthCustomSpotPrice",
			DataSpec{
				ResponseValueType: "uint256",
				AbiComponents: []*ABIComponent{
					{Name: "phantom", FieldType: "bytes"},
				},
				AggregationMethod: "weighted-median",
			},
			"AmpleforthCustomSpotPrice",
			`[""]`,
			`[""]`,
			"0x000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000019416d706c65666f727468437573746f6d53706f74507269636500000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"Test case ChatGPTResponse",
			DataSpec{
				ResponseValueType: "string",
				AbiComponents: []*ABIComponent{
					{Name: "", FieldType: "bytes"},
				},
				AggregationMethod: "weighted-mode",
			},
			"ChatGPTResponse",
			`["What is Tellor?"]`,
			`["V2hhdCBpcyBUZWxsb3I/"]`,
			"0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000f43686174475054526573706f6e7365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000f576861742069732054656c6c6f723f0000000000000000000000000000000000",
		},
		{
			"Test case EVMHeaderslist",
			DataSpec{
				ResponseValueType: "string",
				AbiComponents: []*ABIComponent{
					{Name: "", FieldType: "uint256"},
					{Name: "", FieldType: "uint256[]"},
				},
				AggregationMethod: "weighted-mode",
			},
			"EVMHeaderslist",
			`["1","[17430128, 17430127, 17430126]"]`,
			`[1,[17430128,17430127,17430126]]`,
			"0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000e45564d486561646572736c69737400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000109f670000000000000000000000000000000000000000000000000000000000109f66f000000000000000000000000000000000000000000000000000000000109f66e",
		},
		{
			"Test case Info",
			DataSpec{
				ResponseValueType: "string",
				AbiComponents: []*ABIComponent{
					{Name: "", FieldType: "string"},
					{Name: "", FieldType: "string[]"},
				},
				AggregationMethod: "weighted-mode",
			},
			"Info",
			`["_gAAAAAareXQ","[name]"]`,
			`["_gAAAAAareXQ",["name"]]`,
			"0x000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000004496e666f00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000c5f674141414141617265585100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000046e616d6500000000000000000000000000000000000000000000000000000000",
		},
		{
			"Test case EVMCall",
			DataSpec{
				ResponseValueType: "string",
				AbiComponents: []*ABIComponent{
					{Name: "", FieldType: "uint256"},
					{Name: "", FieldType: "address"},
					{Name: "", FieldType: "bytes"},
				},
				AggregationMethod: "weighted-mode",
			},
			"EVMCall",
			`["1","0x88dF592F8eb5D7Bd38bFeF7dEb0fBc02cf3778a0", "0x18160ddd"]`,
			`[1,"0x88df592f8eb5d7bd38bfef7deb0fbc02cf3778a0","GBYN3Q=="]`,
			"0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000745564d43616c6c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000000100000000000000000000000088df592f8eb5d7bd38bfef7deb0fbc02cf3778a00000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000418160ddd00000000000000000000000000000000000000000000000000000000",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			encodedData, err := tc.dataSpec.EncodeData(tc.queryType, tc.datafields)
			require.NoError(t, err)

			expEncodedResult, _ := hexutil.Decode(tc.expectedEncodedResult)
			require.Equal(t, expEncodedResult, encodedData)
			resultBytes := hexutil.MustDecode(tc.expectedEncodedResult)
			queryType, resultBytes, err := DecodeQueryType(resultBytes)
			require.Equal(t, queryType, tc.queryType)
			require.NoError(t, err)
			res, err := DecodeParamtypes(resultBytes, tc.dataSpec.AbiComponents)
			require.Equal(t, res, tc.expectedDecodedResult)
			require.NoError(t, err)
		})
	}
}

func TestGenesisDataSpec(t *testing.T) {
	val := GenesisDataSpec()
	require.NotNil(t, val)
	require.Equal(t, val.ResponseValueType, "uint256")
	require.Equal(t, val.AggregationMethod, "weighted-median")
	require.Equal(t, val.Registrar, "genesis")
}

func TestValidateValue(t *testing.T) {
	dataspec := DataSpec{
		ResponseValueType: "uint256",
		AbiComponents: []*ABIComponent{
			{Name: "asset", FieldType: "string"},
			{Name: "currency", FieldType: "string"},
		},
		AggregationMethod: "weighted-median",
	}
	err := dataspec.ValidateValue("0x0000000000000000000000000000000000000000000000000000000000000009")
	require.NoError(t, err)
}

func TestDecodeValue(t *testing.T) {
	dataspec := DataSpec{
		ResponseValueType: "uint256",
		AbiComponents: []*ABIComponent{
			{Name: "asset", FieldType: "string"},
			{Name: "currency", FieldType: "string"},
		},
		AggregationMethod: "weighted-median",
	}
	val, err := dataspec.DecodeValue("0x0000000000000000000000000000000000000000000000000000000000000009")
	require.NoError(t, err)
	require.Equal(t, val, "[9]")
}

func TestMakeArgMarshaller(t *testing.T) {
	dataspec := DataSpec{
		ResponseValueType: "uint256",
		AbiComponents: []*ABIComponent{
			{Name: "asset", FieldType: "string"},
			{Name: "currency", FieldType: "string"},
		},
		AggregationMethod: "weighted-median",
	}
	val := dataspec.MakeArgMarshaller()
	require.Equal(t, val[0].Name, "asset")
	require.Equal(t, val[1].Name, "currency")
}

func TestMakeAruments(t *testing.T) {
	dataspec := DataSpec{
		ResponseValueType: "uint256",
		AbiComponents: []*ABIComponent{
			{Name: "asset", FieldType: "string"},
			{Name: "currency", FieldType: "string"},
		},
		AggregationMethod: "weighted-median",
	}
	val := dataspec.MakeArgMarshaller()
	args := MakeArguments(val)
	require.Equal(t, args[0].Name, "asset")
}

func TestValidateMultiValue(t *testing.T) {
	dataspec := DataSpec{
		ResponseValueType: "uint256, string, uint256",
		AbiComponents: []*ABIComponent{
			{Name: "asset", FieldType: "string"},
			{Name: "currency", FieldType: "string"},
		},
		AggregationMethod: "weighted-median",
	}
	err := dataspec.ValidateValue("0x000000000000000000000000000000000000000000000000000000000000007b000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000001c8000000000000000000000000000000000000000000000000000000000000000a74657374696e6731323300000000000000000000000000000000000000000000")
	require.NoError(t, err)
}
