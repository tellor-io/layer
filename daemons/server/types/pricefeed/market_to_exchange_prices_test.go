package types

// import (
// 	"testing"
// 	"time"

// 	pricefeed_types "github.com/tellor-io/layer/daemons/pricefeed/types"

// 	"github.com/stretchr/testify/require"
// 	"github.com/tellor-io/layer/daemons/pricefeed/api"
// 	"github.com/tellor-io/layer/daemons/testutil/constants"
// 	"github.com/tellor-io/layer/x/prices/types"
// )

// func TestNewMarketToExchangePrices_IsEmpty(t *testing.T) {
// 	mte := NewMarketToExchangePrices(pricefeed_types.MaxPriceAge)

// 	require.Empty(t, mte.marketToExchangePrices)
// }

// func TestUpdatePrices_SingleUpdateSinglePrice(t *testing.T) {
// 	mte := NewMarketToExchangePrices(pricefeed_types.MaxPriceAge)

// 	mte.UpdatePrices(
// 		[]*api.MarketPriceUpdate{
// 			{
// 				MarketId: constants.MarketId9,
// 				ExchangePrices: []*api.ExchangePrice{
// 					constants.Exchange1_Price1_TimeT,
// 				},
// 			},
// 		})

// 	require.Len(t, mte.marketToExchangePrices, 1)
// 	_, ok := mte.marketToExchangePrices[constants.MarketId9]
// 	require.True(t, ok)
// }

// func TestUpdatePrices_SingleUpdateMultiPrices(t *testing.T) {
// 	mte := NewMarketToExchangePrices(pricefeed_types.MaxPriceAge)

// 	mte.UpdatePrices(
// 		[]*api.MarketPriceUpdate{
// 			{
// 				MarketId: constants.MarketId9,
// 				ExchangePrices: []*api.ExchangePrice{
// 					constants.Exchange1_Price1_TimeT,
// 					constants.Exchange2_Price2_TimeT,
// 				},
// 			},
// 		})

// 	require.Len(t, mte.marketToExchangePrices, 1)
// 	_, ok := mte.marketToExchangePrices[constants.MarketId9]
// 	require.True(t, ok)
// }

// func TestUpdatePrices_MultiUpdatesMultiPrices(t *testing.T) {
// 	mte := NewMarketToExchangePrices(pricefeed_types.MaxPriceAge)

// 	mte.UpdatePrices(
// 		[]*api.MarketPriceUpdate{
// 			{
// 				MarketId: constants.MarketId9,
// 				ExchangePrices: []*api.ExchangePrice{
// 					constants.Exchange1_Price1_TimeT,
// 					constants.Exchange2_Price2_TimeT,
// 				},
// 			},
// 			{
// 				MarketId: constants.MarketId8,
// 				ExchangePrices: []*api.ExchangePrice{
// 					constants.Exchange1_Price1_TimeT,
// 					constants.Exchange2_Price2_TimeT,
// 				},
// 			},
// 		})

// 	require.Len(t, mte.marketToExchangePrices, 2)
// 	_, ok9 := mte.marketToExchangePrices[constants.MarketId9]
// 	require.True(t, ok9)
// 	_, ok8 := mte.marketToExchangePrices[constants.MarketId8]
// 	require.True(t, ok8)
// }

// func TestUpdatePrices_MultiUpdatesMultiPricesRepeated(t *testing.T) {
// 	mte := NewMarketToExchangePrices(pricefeed_types.MaxPriceAge)

// 	mte.UpdatePrices(
// 		[]*api.MarketPriceUpdate{
// 			{
// 				MarketId: constants.MarketId9,
// 				ExchangePrices: []*api.ExchangePrice{
// 					constants.Exchange1_Price1_TimeT,
// 					constants.Exchange2_Price2_TimeT,
// 				},
// 			},
// 			{
// 				MarketId: constants.MarketId9, // Repeated market
// 				ExchangePrices: []*api.ExchangePrice{
// 					constants.Exchange1_Price1_TimeT,
// 					constants.Exchange3_Price4_AfterTimeT,
// 				},
// 			},
// 			{
// 				MarketId: constants.MarketId8,
// 				ExchangePrices: []*api.ExchangePrice{
// 					constants.Exchange1_Price1_TimeT,
// 					constants.Exchange2_Price2_TimeT,
// 				},
// 			},
// 			{
// 				MarketId: constants.MarketId8, // Repeated market
// 				ExchangePrices: []*api.ExchangePrice{
// 					constants.Exchange1_Price1_TimeT,
// 					constants.Exchange3_Price4_AfterTimeT,
// 				},
// 			},
// 		})

// 	require.Len(t, mte.marketToExchangePrices, 2)
// 	_, ok9 := mte.marketToExchangePrices[constants.MarketId9]
// 	require.True(t, ok9)
// 	_, ok8 := mte.marketToExchangePrices[constants.MarketId8]
// 	require.True(t, ok8)
// }

// func TestGetValidMedianPrices_EmptyResult(t *testing.T) {
// 	tests := map[string]struct {
// 		updatePriceInput           []*api.MarketPriceUpdate
// 		getPricesInputMarketParams []types.MarketParam
// 		getPricesInputTime         time.Time
// 	}{
// 		"No market specified": {
// 			updatePriceInput:           constants.AtTimeTPriceUpdate,
// 			getPricesInputMarketParams: []types.MarketParam{}, // No market specified.
// 			getPricesInputTime:         constants.TimeT,
// 		},
// 		"No valid price timestamps": {
// 			updatePriceInput:           constants.AtTimeTPriceUpdate,
// 			getPricesInputMarketParams: constants.AllMarketParamsMinExchanges2,
// 			// Updates @ timeT are invalid at this read time
// 			getPricesInputTime: constants.TimeTPlusThreshold.Add(time.Duration(1)),
// 		},
// 		"Empty prices does not throw": {
// 			updatePriceInput: []*api.MarketPriceUpdate{
// 				{
// 					MarketId: constants.MarketId9,
// 					ExchangePrices: []*api.ExchangePrice{
// 						constants.Exchange1_Price3_BeforeTimeT, // Invalid time
// 					},
// 				},
// 			},
// 			getPricesInputMarketParams: []types.MarketParam{
// 				{
// 					Id:           constants.MarketId9,
// 					MinExchanges: 0, // Set to 0 to trigger median calc error
// 				},
// 			},
// 			getPricesInputTime: constants.TimeT,
// 		},
// 		"Does not meet min exchanges": {
// 			updatePriceInput: constants.AtTimeTPriceUpdate,
// 			// MinExchanges is 3 for all markets, but updates are from 2 exchanges
// 			getPricesInputMarketParams: constants.AllMarketParamsMinExchanges3,
// 			getPricesInputTime:         constants.TimeT,
// 		},
// 	}

// 	for name, tc := range tests {
// 		t.Run(name, func(t *testing.T) {
// 			mte := NewMarketToExchangePrices(pricefeed_types.MaxPriceAge)
// 			mte.UpdatePrices(tc.updatePriceInput)
// 			r := mte.GetValidMedianPrices(
// 				tc.getPricesInputMarketParams,
// 				tc.getPricesInputTime,
// 			)

// 			require.Len(t, r, 0) // The result is empty.
// 		})
// 	}
// }

// func TestGetValidMedianPrices_MultiMarketSuccess(t *testing.T) {
// 	mte := NewMarketToExchangePrices(pricefeed_types.MaxPriceAge)

// 	mte.UpdatePrices(constants.MixedTimePriceUpdate)

// 	r := mte.GetValidMedianPrices(constants.AllMarketParamsMinExchanges2, constants.TimeT)

// 	require.Len(t, r, 2)
// 	require.Equal(t, uint64(2002), r[constants.MarketId9]) // Median of 1001, 2002, 3003
// 	require.Equal(t, uint64(2503), r[constants.MarketId8]) // Median of 2002, 3003
// 	// Market7 only has 1 valid price due to update time constraint,
// 	// but the min exchanges required is 2. Therefore, no median price.
// }
