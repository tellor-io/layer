package integration_test

import (
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *IntegrationTestSuite) TestRoundOnePricesAdditionalPayersWithoutBlockingFunding() {
	fx := s.newReportedStake()
	// createValidatorAccs assigns validator accounts numbers starting at 100.
	// Put the SDK's auto-generated payer accounts above that fixture range.
	accountNumbers := collections.Item[uint64](s.Setup.Accountkeeper.AccountNumber)
	s.Require().NoError(accountNumbers.Set(s.Setup.Ctx, 1024))
	quote := fx.stake
	collectible := quote.QuoRaw(2)
	s.leaveReporterSlashable(fx, collectible)
	moduleBefore := s.moduleBalance()

	opening := math.NewInt(10_000)
	proposer := sample.AccAddressBytes()
	s.Setup.MintTokens(proposer, opening.AddRaw(1))
	dispute := s.proposeDispute(fx.report, disputetypes.Major, proposer, opening, false)
	s.Require().Equal(disputetypes.Prevote, dispute.DisputeStatus)
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	add := func(payer sdk.AccAddress, amount math.Int) error {
		_, err := msgServer.AddFeeToDispute(s.Setup.Ctx, &disputetypes.MsgAddFeeToDispute{
			Creator: payer.String(), DisputeId: dispute.DisputeId,
			Amount: sdk.NewCoin(s.Setup.Denom, amount),
		})
		return err
	}

	// After the affordable 32-payer tier (including the proposer), a new
	// partial payer needs to put up at least 1% of the quote. A count-only
	// ceiling lets dust payments fill every partial slot and forces the next
	// honest payer to cover the entire remainder alone.
	unexpectedDust := 0
	for i := range 99 {
		payer := sample.AccAddressBytes()
		s.Setup.MintTokens(payer, math.OneInt())
		err := add(payer, math.OneInt())
		if i < 31 {
			s.Require().NoError(err)
		} else if err == nil {
			unexpectedDust++
		} else {
			s.ErrorContains(err, "minimum fee for a new dispute payer")
			s.True(s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, payer, s.Setup.Denom).Amount.Equal(math.OneInt()))
		}
	}
	s.Zero(unexpectedDust, "dust must not consume capacity outside the free tier")

	// Existing payers can still contribute below the floor. A new payer can
	// make a qualifying partial payment without taking on the whole quote.
	s.Require().NoError(add(proposer, math.OneInt()))
	s.True(opening.AddRaw(1).Equal(s.payerInfo(dispute.DisputeId, proposer).Amount))
	sponsor := sample.AccAddressBytes()
	pledge := quote.QuoRaw(100)
	s.Setup.MintTokens(sponsor, pledge)
	s.Require().NoError(add(sponsor, pledge))
	s.True(pledge.Equal(s.payerInfo(dispute.DisputeId, sponsor).Amount))
	stored, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, dispute.DisputeId)
	s.Require().NoError(err)
	s.Equal(disputetypes.Prevote, stored.DisputeStatus)

	finisher := sample.AccAddressBytes()
	remaining := quote.Sub(stored.FeeTotal)
	s.Setup.MintTokens(finisher, remaining)
	s.Require().NoError(add(finisher, remaining))
	stored, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, dispute.DisputeId)
	s.Require().NoError(err)
	s.Equal(disputetypes.Voting, stored.DisputeStatus)
	s.True(collectible.Equal(stored.FeeTotal))
	s.True(moduleBefore.Add(collectible.MulRaw(2)).Equal(s.moduleBalance()), "all excess must be returned before voting opens")
}
