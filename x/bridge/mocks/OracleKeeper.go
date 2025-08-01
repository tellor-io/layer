// Code generated by mockery v2.51.1. DO NOT EDIT.

package mocks

import (
	context "context"
	time "time"

	mock "github.com/stretchr/testify/mock"

	types "github.com/tellor-io/layer/x/oracle/types"
)

// OracleKeeper is an autogenerated mock type for the OracleKeeper type
type OracleKeeper struct {
	mock.Mock
}

// GetAggregateBefore provides a mock function with given fields: ctx, queryId, timestampBefore
func (_m *OracleKeeper) GetAggregateBefore(ctx context.Context, queryId []byte, timestampBefore time.Time) (*types.Aggregate, time.Time, error) {
	ret := _m.Called(ctx, queryId, timestampBefore)

	if len(ret) == 0 {
		panic("no return value specified for GetAggregateBefore")
	}

	var r0 *types.Aggregate
	var r1 time.Time
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte, time.Time) (*types.Aggregate, time.Time, error)); ok {
		return rf(ctx, queryId, timestampBefore)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []byte, time.Time) *types.Aggregate); ok {
		r0 = rf(ctx, queryId, timestampBefore)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Aggregate)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []byte, time.Time) time.Time); ok {
		r1 = rf(ctx, queryId, timestampBefore)
	} else {
		r1 = ret.Get(1).(time.Time)
	}

	if rf, ok := ret.Get(2).(func(context.Context, []byte, time.Time) error); ok {
		r2 = rf(ctx, queryId, timestampBefore)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetAggregateByTimestamp provides a mock function with given fields: ctx, queryId, timestamp
func (_m *OracleKeeper) GetAggregateByTimestamp(ctx context.Context, queryId []byte, timestamp uint64) (types.Aggregate, error) {
	ret := _m.Called(ctx, queryId, timestamp)

	if len(ret) == 0 {
		panic("no return value specified for GetAggregateByTimestamp")
	}

	var r0 types.Aggregate
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte, uint64) (types.Aggregate, error)); ok {
		return rf(ctx, queryId, timestamp)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []byte, uint64) types.Aggregate); ok {
		r0 = rf(ctx, queryId, timestamp)
	} else {
		r0 = ret.Get(0).(types.Aggregate)
	}

	if rf, ok := ret.Get(1).(func(context.Context, []byte, uint64) error); ok {
		r1 = rf(ctx, queryId, timestamp)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAggregatedReportsByHeight provides a mock function with given fields: ctx, height
func (_m *OracleKeeper) GetAggregatedReportsByHeight(ctx context.Context, height uint64) []types.Aggregate {
	ret := _m.Called(ctx, height)

	if len(ret) == 0 {
		panic("no return value specified for GetAggregatedReportsByHeight")
	}

	var r0 []types.Aggregate
	if rf, ok := ret.Get(0).(func(context.Context, uint64) []types.Aggregate); ok {
		r0 = rf(ctx, height)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.Aggregate)
		}
	}

	return r0
}

// GetCurrentAggregateReport provides a mock function with given fields: ctx, queryId
func (_m *OracleKeeper) GetCurrentAggregateReport(ctx context.Context, queryId []byte) (*types.Aggregate, time.Time, error) {
	ret := _m.Called(ctx, queryId)

	if len(ret) == 0 {
		panic("no return value specified for GetCurrentAggregateReport")
	}

	var r0 *types.Aggregate
	var r1 time.Time
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte) (*types.Aggregate, time.Time, error)); ok {
		return rf(ctx, queryId)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []byte) *types.Aggregate); ok {
		r0 = rf(ctx, queryId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Aggregate)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []byte) time.Time); ok {
		r1 = rf(ctx, queryId)
	} else {
		r1 = ret.Get(1).(time.Time)
	}

	if rf, ok := ret.Get(2).(func(context.Context, []byte) error); ok {
		r2 = rf(ctx, queryId)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetNoStakeReportByQueryIdTimestamp provides a mock function with given fields: ctx, queryId, timestamp
func (_m *OracleKeeper) GetNoStakeReportByQueryIdTimestamp(ctx context.Context, queryId []byte, timestamp uint64) (*types.NoStakeMicroReport, error) {
	ret := _m.Called(ctx, queryId, timestamp)

	if len(ret) == 0 {
		panic("no return value specified for GetNoStakeReportByQueryIdTimestamp")
	}

	var r0 *types.NoStakeMicroReport
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte, uint64) (*types.NoStakeMicroReport, error)); ok {
		return rf(ctx, queryId, timestamp)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []byte, uint64) *types.NoStakeMicroReport); ok {
		r0 = rf(ctx, queryId, timestamp)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.NoStakeMicroReport)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []byte, uint64) error); ok {
		r1 = rf(ctx, queryId, timestamp)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTimestampAfter provides a mock function with given fields: ctx, queryId, timestamp
func (_m *OracleKeeper) GetTimestampAfter(ctx context.Context, queryId []byte, timestamp time.Time) (time.Time, error) {
	ret := _m.Called(ctx, queryId, timestamp)

	if len(ret) == 0 {
		panic("no return value specified for GetTimestampAfter")
	}

	var r0 time.Time
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte, time.Time) (time.Time, error)); ok {
		return rf(ctx, queryId, timestamp)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []byte, time.Time) time.Time); ok {
		r0 = rf(ctx, queryId, timestamp)
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	if rf, ok := ret.Get(1).(func(context.Context, []byte, time.Time) error); ok {
		r1 = rf(ctx, queryId, timestamp)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTimestampBefore provides a mock function with given fields: ctx, queryId, timestamp
func (_m *OracleKeeper) GetTimestampBefore(ctx context.Context, queryId []byte, timestamp time.Time) (time.Time, error) {
	ret := _m.Called(ctx, queryId, timestamp)

	if len(ret) == 0 {
		panic("no return value specified for GetTimestampBefore")
	}

	var r0 time.Time
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte, time.Time) (time.Time, error)); ok {
		return rf(ctx, queryId, timestamp)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []byte, time.Time) time.Time); ok {
		r0 = rf(ctx, queryId, timestamp)
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	if rf, ok := ret.Get(1).(func(context.Context, []byte, time.Time) error); ok {
		r1 = rf(ctx, queryId, timestamp)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetAggregate provides a mock function with given fields: ctx, report, queryData, queryType
func (_m *OracleKeeper) SetAggregate(ctx context.Context, report *types.Aggregate, queryData []byte, queryType string) error {
	ret := _m.Called(ctx, report, queryData, queryType)

	if len(ret) == 0 {
		panic("no return value specified for SetAggregate")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.Aggregate, []byte, string) error); ok {
		r0 = rf(ctx, report, queryData, queryType)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewOracleKeeper creates a new instance of OracleKeeper. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewOracleKeeper(t interface {
	mock.TestingT
	Cleanup(func())
}) *OracleKeeper {
	mock := &OracleKeeper{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
