package keeper_test

import (
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
)

const (
	trbusdQueryData = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
)

func (s *KeeperTestSuite) TestMsgNoStakeReport() {
	require := s.Require()

	k := s.oracleKeeper
	require.NotNil(k)

	timestamp := time.Now()
	ctx := s.ctx.WithBlockHeight(10).WithBlockTime(timestamp)

	queryDataBz, err := hex.DecodeString(trbusdQueryData)
	require.NoError(err)

	// set query data limit to 0.5mb
	queryDataLimit := types.QueryDataLimit{
		Limit: 524288,
	}
	err = k.QueryDataLimit.Set(ctx, queryDataLimit)
	require.NoError(err)

	reporter := sample.AccAddressBytes()

	type testCase struct {
		name          string
		setup         func()
		msg           types.MsgNoStakeReport
		expectedError error
	}

	testCases := []testCase{
		{
			name:  "empty query data",
			setup: func() {},
			msg: types.MsgNoStakeReport{
				Creator:   reporter.String(),
				QueryData: []byte{},
				Value:     "100",
			},
			expectedError: errors.New("query data cannot be empty"),
		},
		{
			name:  "empty value",
			setup: func() {},
			msg: types.MsgNoStakeReport{
				Creator:   reporter.String(),
				QueryData: []byte{1, 2, 3},
				Value:     "",
			},
			expectedError: errors.New("value cannot be empty"),
		},
		{
			name:  "query data too large",
			setup: func() {},
			msg: types.MsgNoStakeReport{
				Creator:   reporter.String(),
				QueryData: []byte(strings.Repeat("a", 524289)),
				Value:     "100",
			},
			expectedError: errors.New("query data too large"),
		},
		{
			name:  "success",
			setup: func() {},
			msg: types.MsgNoStakeReport{
				Creator:   reporter.String(),
				QueryData: queryDataBz,
				Value:     "100",
			},
			expectedError: nil,
		},
		{
			name: "report already exists",
			setup: func() {
				queryId := utils.QueryIDFromData(queryDataBz)
				err := k.NoStakeReports.Set(ctx, collections.Join(queryId, uint64(timestamp.UnixMilli())), types.NoStakeMicroReport{
					Reporter:  reporter,
					QueryData: queryDataBz,
					Value:     "100",
					Timestamp: timestamp,
				})
				require.NoError(err)
			},
			msg: types.MsgNoStakeReport{
				Creator:   reporter.String(),
				QueryData: queryDataBz,
				Value:     "100",
			},
			expectedError: errors.New("report for this queryId already exists at this height, please resubmit next block"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setup()
			_, err := s.msgServer.NoStakeReport(ctx, &tc.msg)
			if tc.expectedError != nil {
				require.Error(err)
				require.Contains(err.Error(), tc.expectedError.Error())
			} else {
				require.NoError(err)
				queryId := utils.QueryIDFromData(tc.msg.QueryData)
				report, err := k.NoStakeReports.Get(ctx, collections.Join(queryId, uint64(timestamp.UnixMilli())))
				require.NoError(err)
				reporterAddr := sdk.AccAddress(report.Reporter)
				require.Equal(reporterAddr.String(), tc.msg.Creator)
				require.Equal(tc.msg.QueryData, report.QueryData)
				require.Equal(tc.msg.Value, report.Value)
			}
		})
	}
}
