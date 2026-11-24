package integration_test

import (
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Compare two funded histories with the same aggregate pledge. Only the
// number of earlier payers differs. Meter the completing message in isolation.
func (s *IntegrationTestSuite) TestResizeGasIndependentOfEarlierPayers() {
	fx := s.newReportedStake()
	s.leaveReporterSlashable(fx, math.NewInt(60))
	first, last := sample.AccAddressBytes(), sample.AccAddressBytes()
	opening := fx.stake.SubRaw(1)
	s.Setup.MintTokens(first, opening)
	s.Setup.MintTokens(last, math.OneInt())
	d := s.proposeDispute(fx.report, types.Major, first, opening, false)
	s.Require().Equal(types.Prevote, d.DisputeStatus)
	server := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	gas := func(others int64) uint64 {
		ctx, _ := s.Setup.Ctx.CacheContext()
		payers := s.Setup.Disputekeeper.DisputeFeePayer
		s.Require().NoError(payers.Set(ctx, collections.Join(d.DisputeId, first.Bytes()), types.PayerInfo{Amount: opening.SubRaw(others)}))
		for i := int64(0); i < others; i++ {
			payer := sample.AccAddressBytes()
			s.Require().NoError(payers.Set(ctx, collections.Join(d.DisputeId, payer.Bytes()), types.PayerInfo{Amount: math.OneInt()}))
		}
		meter := storetypes.NewInfiniteGasMeter()
		_, err := server.AddFeeToDispute(ctx.WithGasMeter(meter), &types.MsgAddFeeToDispute{
			Creator: last.String(), DisputeId: d.DisputeId, Amount: sdk.NewCoin(s.Setup.Denom, math.OneInt()),
		})
		s.Require().NoError(err)
		funded, err := s.Setup.Disputekeeper.Disputes.Get(ctx, d.DisputeId)
		s.Require().NoError(err)
		s.Require().Equal(math.NewInt(60), funded.DisputeFee)
		return meter.GasConsumed()
	}
	onePayer := gas(0)
	s.Require().Positive(onePayer)
	s.Equal(onePayer, gas(31))
}
