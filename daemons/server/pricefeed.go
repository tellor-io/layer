package server

import (
	"context"
	"fmt"
	"time"

	gometrics "github.com/hashicorp/go-metrics"
	"github.com/tellor-io/layer/daemons/constants"
	"github.com/tellor-io/layer/daemons/lib/metrics"
	pricefeedmetrics "github.com/tellor-io/layer/daemons/pricefeed/metrics"
	"github.com/tellor-io/layer/daemons/server/types"
	daemontypes "github.com/tellor-io/layer/daemons/server/types/daemons"
	pricefeedtypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/telemetry"
)

// PriceFeedServer defines the fields required for price updates.
type PriceFeedServer struct {
	marketToExchange *pricefeedtypes.MarketToExchangePrices
}

// WithPriceFeedMarketToExchangePrices sets the `MarketToExchangePrices` field.
// This is used by the price feed service to communicate price updates
// to the main application.
func (server *Server) WithPriceFeedMarketToExchangePrices(
	marketToExchange *pricefeedtypes.MarketToExchangePrices,
) *Server {
	server.marketToExchange = marketToExchange
	return server
}

// UpdateMarketPrices updates prices from exchanges for each market provided.
func (s *Server) UpdateMarketPrices(
	ctx context.Context,
	req *daemontypes.UpdateMarketPricesRequest,
) (
	response *daemontypes.UpdateMarketPricesResponse,
	err error,
) {
	// Measure latency in ingesting and handling gRPC price update.
	defer telemetry.ModuleMeasureSince(
		metrics.PricefeedServer,
		time.Now(),
		metrics.PricefeedServerUpdatePrices,
		metrics.Latency,
	)

	// This panic is an unexpected condition because we initialize the market price cache in app initialization before
	// starting the server or daemons.
	if s.marketToExchange == nil {
		panic(fmt.Errorf("server not initialized correctly, marketToExchange not initialized"))
	}

	if err = validateMarketPricesUpdatesMessage(req); err != nil {
		// Log if failure occurs during an update.
		s.logger.Error("Failed to validate price update message", "error", err)
		return nil, err
	}

	s.marketToExchange.UpdatePrices(req.MarketPriceUpdates)

	// Capture valid responses in metrics.
	s.reportValidResponse(types.PricefeedDaemonServiceName)

	return &daemontypes.UpdateMarketPricesResponse{}, nil
}

// validateMarketPricesUpdatesMessage validates a `UpdateMarketPricesRequest`.
func validateMarketPricesUpdatesMessage(req *daemontypes.UpdateMarketPricesRequest) error {
	if len(req.MarketPriceUpdates) == 0 {
		return fmt.Errorf("ErrPriceFeedMarketPriceUpdateEmpty")
	}

	for _, mpu := range req.MarketPriceUpdates {
		if err := validateMarketPriceUpdate(mpu); err != nil {
			// Measure failure per market in validation.
			telemetry.IncrCounterWithLabels(
				[]string{
					metrics.PricefeedServer,
					metrics.PricefeedServerValidatePrices,
					metrics.Error,
				},
				1,
				[]gometrics.Label{
					pricefeedmetrics.GetLabelForMarketId(mpu.MarketId),
				},
			)

			return err
		}
	}

	return nil
}

// validateMarketPriceUpdate validates a single `MarketPriceUpdate`.
func validateMarketPriceUpdate(mpu *daemontypes.MarketPriceUpdate) error {
	for _, ep := range mpu.ExchangePrices {
		if ep.Price == constants.DefaultPrice {
			return generateSdkErrorForExchangePrice(
				fmt.Errorf("ErrPriceFeedInvalidPrice"),
				ep,
				mpu.MarketId,
			)
		}

		if ep.LastUpdateTime == nil {
			return generateSdkErrorForExchangePrice(
				fmt.Errorf("ErrPriceFeedLastUpdateTimeNotSet"),
				ep,
				mpu.MarketId,
			)
		}
	}
	return nil
}

// generateSdkErrorForExchangePrice generates an error for an invalid `ExchangePrice`.
func generateSdkErrorForExchangePrice(err error, ep *daemontypes.ExchangePrice, marketId uint32) error {
	return errorsmod.Wrapf(err, "ExchangePrice: %v and MarketId: %d", ep, marketId)
}
