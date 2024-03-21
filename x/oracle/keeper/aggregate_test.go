package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestFindTimestampBefore() {
	testCases := []struct {
		name       string
		timestamps []time.Time
		target     time.Time
		expectedTs time.Time
	}{
		{
			name:       "Empty slice",
			timestamps: []time.Time{},
			target:     time.Unix(100, 0),
			expectedTs: time.Time{},
		},
		{
			name:       "Single timestamp before target",
			timestamps: []time.Time{time.Unix(50, 0)},
			target:     time.Unix(100, 0),
			expectedTs: time.Unix(50, 0),
		},
		{
			name:       "Single timestamp after target",
			timestamps: []time.Time{time.Unix(150, 0)},
			target:     time.Unix(100, 0),
			expectedTs: time.Time{},
		},
		{
			name:       "Multiple timestamps, target present",
			timestamps: []time.Time{time.Unix(50, 0), time.Unix(100, 0), time.Unix(150, 0)},
			target:     time.Unix(100, 0),
			expectedTs: time.Unix(100, 0),
		},
		{
			name:       "Multiple timestamps, target not present",
			timestamps: []time.Time{time.Unix(50, 0), time.Unix(70, 0), time.Unix(90, 0), time.Unix(110, 0), time.Unix(130, 0)},
			target:     time.Unix(100, 0),
			expectedTs: time.Unix(90, 0),
		},
		{
			name:       "Multiple timestamps, target before all",
			timestamps: []time.Time{time.Unix(200, 0), time.Unix(300, 0), time.Unix(400, 0)},
			target:     time.Unix(100, 0),
			expectedTs: time.Time{},
		},
		{
			name:       "Multiple timestamps, target after all",
			timestamps: []time.Time{time.Unix(10, 0), time.Unix(20, 0), time.Unix(40, 0)},
			target:     time.Unix(100, 0),
			expectedTs: time.Unix(40, 0),
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.SetupTest()
			queryId := []byte("test")
			for _, v := range tc.timestamps {
				err := s.oracleKeeper.Aggregates.Set(
					s.ctx,
					collections.Join(queryId, v.Unix()),
					types.Aggregate{},
				)
				s.Require().NoError(err)
			}

			ts, err := s.oracleKeeper.GetTimestampBefore(s.ctx, queryId, tc.target)
			if ts.IsZero() {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}

			if ts != tc.expectedTs {
				t.Errorf("Test '%s' failed: expected %v, got %v", tc.name, tc.expectedTs, ts)
			}
		})
	}
}

func (s *KeeperTestSuite) TestFindTimestampAfter() {
	testCases := []struct {
		name       string
		timestamps []time.Time
		target     time.Time
		expectedTs time.Time
	}{
		{
			name:       "Empty slice",
			timestamps: []time.Time{},
			target:     time.Unix(100, 0),
			expectedTs: time.Time{},
		},
		{
			name:       "Single timestamp after target",
			timestamps: []time.Time{time.Unix(50, 0)},
			target:     time.Unix(25, 0),
			expectedTs: time.Unix(50, 0),
		},
		{
			name:       "Single timestamp before target",
			timestamps: []time.Time{time.Unix(150, 0)},
			target:     time.Unix(200, 0),
			expectedTs: time.Time{},
		},
		{
			name:       "Multiple timestamps, target present",
			timestamps: []time.Time{time.Unix(50, 0), time.Unix(100, 0), time.Unix(150, 0)},
			target:     time.Unix(100, 0),
			expectedTs: time.Unix(100, 0),
		},
		{
			name:       "Multiple timestamps, target not present",
			timestamps: []time.Time{time.Unix(50, 0), time.Unix(70, 0), time.Unix(90, 0), time.Unix(110, 0), time.Unix(130, 0)},
			target:     time.Unix(100, 0),
			expectedTs: time.Unix(110, 0),
		},
		{
			name:       "Multiple timestamps, target before all",
			timestamps: []time.Time{time.Unix(200, 0), time.Unix(300, 0), time.Unix(400, 0)},
			target:     time.Unix(100, 0),
			expectedTs: time.Unix(200, 0),
		},
		{
			name:       "Multiple timestamps, target after all",
			timestamps: []time.Time{time.Unix(10, 0), time.Unix(20, 0), time.Unix(40, 0)},
			target:     time.Unix(100, 0),
			expectedTs: time.Time{},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.SetupTest()
			queryId := []byte("test")
			for _, v := range tc.timestamps {
				err := s.oracleKeeper.Aggregates.Set(
					s.ctx,
					collections.Join(queryId, v.Unix()),
					types.Aggregate{},
				)
				s.Require().NoError(err)
			}

			ts, err := s.oracleKeeper.GetTimestampAfter(s.ctx, queryId, tc.target)
			if ts.IsZero() {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}

			if ts != tc.expectedTs {
				t.Errorf("Test '%s' failed: expected %v, got %v", tc.name, tc.expectedTs, ts)
			}
		})
	}
}
