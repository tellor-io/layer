// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package oracle

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// QueryClient is the client API for Query service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type QueryClient interface {
	// Parameters queries the parameters of the module.
	Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error)
	// Queries a list of GetReportsbyQid items.
	GetReportsbyQid(ctx context.Context, in *QueryGetReportsbyQidRequest, opts ...grpc.CallOption) (*QueryMicroReportsResponse, error)
	GetReportsbyReporter(ctx context.Context, in *QueryGetReportsbyReporterRequest, opts ...grpc.CallOption) (*QueryMicroReportsResponse, error)
	GetReportsbyReporterQid(ctx context.Context, in *QueryGetReportsbyReporterQidRequest, opts ...grpc.CallOption) (*QueryMicroReportsResponse, error)
	// Queries a list of GetCurrentTip items.
	GetCurrentTip(ctx context.Context, in *QueryGetCurrentTipRequest, opts ...grpc.CallOption) (*QueryGetCurrentTipResponse, error)
	// Queries a list of GetUserTipTotal items.
	GetUserTipTotal(ctx context.Context, in *QueryGetUserTipTotalRequest, opts ...grpc.CallOption) (*QueryGetUserTipTotalResponse, error)
	// Queries a list of GetAggregatedReport items.
	GetDataBefore(ctx context.Context, in *QueryGetDataBeforeRequest, opts ...grpc.CallOption) (*QueryGetDataBeforeResponse, error)
	// Queries a list of GetTimeBasedRewards items.
	GetTimeBasedRewards(ctx context.Context, in *QueryGetTimeBasedRewardsRequest, opts ...grpc.CallOption) (*QueryGetTimeBasedRewardsResponse, error)
	// Queries a list of CurrentCyclelistQuery items.
	CurrentCyclelistQuery(ctx context.Context, in *QueryCurrentCyclelistQueryRequest, opts ...grpc.CallOption) (*QueryCurrentCyclelistQueryResponse, error)
	// Queries a list of NextCyclelistQuery items.
	NextCyclelistQuery(ctx context.Context, in *QueryNextCyclelistQueryRequest, opts ...grpc.CallOption) (*QueryNextCyclelistQueryResponse, error)
	RetrieveData(ctx context.Context, in *QueryRetrieveDataRequest, opts ...grpc.CallOption) (*QueryRetrieveDataResponse, error)
	GetCurrentAggregateReport(ctx context.Context, in *QueryGetCurrentAggregateReportRequest, opts ...grpc.CallOption) (*QueryGetCurrentAggregateReportResponse, error)
	GetAggregateBeforeByReporter(ctx context.Context, in *QueryGetAggregateBeforeByReporterRequest, opts ...grpc.CallOption) (*QueryGetAggregateBeforeByReporterResponse, error)
	GetQuery(ctx context.Context, in *QueryGetQueryRequest, opts ...grpc.CallOption) (*QueryGetQueryResponse, error)
	TippedQueries(ctx context.Context, in *QueryTippedQueriesRequest, opts ...grpc.CallOption) (*QueryTippedQueriesResponse, error)
	GetReportsByAggregate(ctx context.Context, in *QueryGetReportsByAggregateRequest, opts ...grpc.CallOption) (*QueryGetReportsByAggregateResponse, error)
	GetCurrentQueryByQueryId(ctx context.Context, in *QueryGetCurrentQueryByQueryIdRequest, opts ...grpc.CallOption) (*QueryGetCurrentQueryByQueryIdResponse, error)
	QueryState(ctx context.Context, in *QueryStateRequest, opts ...grpc.CallOption) (*QueryStateResponse, error)
}

type queryClient struct {
	cc grpc.ClientConnInterface
}

func NewQueryClient(cc grpc.ClientConnInterface) QueryClient {
	return &queryClient{cc}
}

func (c *queryClient) Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error) {
	out := new(QueryParamsResponse)
	err := c.cc.Invoke(ctx, "/layer.oracle.Query/Params", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetReportsbyQid(ctx context.Context, in *QueryGetReportsbyQidRequest, opts ...grpc.CallOption) (*QueryMicroReportsResponse, error) {
	out := new(QueryMicroReportsResponse)
	err := c.cc.Invoke(ctx, "/layer.oracle.Query/GetReportsbyQid", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetReportsbyReporter(ctx context.Context, in *QueryGetReportsbyReporterRequest, opts ...grpc.CallOption) (*QueryMicroReportsResponse, error) {
	out := new(QueryMicroReportsResponse)
	err := c.cc.Invoke(ctx, "/layer.oracle.Query/GetReportsbyReporter", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetReportsbyReporterQid(ctx context.Context, in *QueryGetReportsbyReporterQidRequest, opts ...grpc.CallOption) (*QueryMicroReportsResponse, error) {
	out := new(QueryMicroReportsResponse)
	err := c.cc.Invoke(ctx, "/layer.oracle.Query/GetReportsbyReporterQid", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetCurrentTip(ctx context.Context, in *QueryGetCurrentTipRequest, opts ...grpc.CallOption) (*QueryGetCurrentTipResponse, error) {
	out := new(QueryGetCurrentTipResponse)
	err := c.cc.Invoke(ctx, "/layer.oracle.Query/GetCurrentTip", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetUserTipTotal(ctx context.Context, in *QueryGetUserTipTotalRequest, opts ...grpc.CallOption) (*QueryGetUserTipTotalResponse, error) {
	out := new(QueryGetUserTipTotalResponse)
	err := c.cc.Invoke(ctx, "/layer.oracle.Query/GetUserTipTotal", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetDataBefore(ctx context.Context, in *QueryGetDataBeforeRequest, opts ...grpc.CallOption) (*QueryGetDataBeforeResponse, error) {
	out := new(QueryGetDataBeforeResponse)
	err := c.cc.Invoke(ctx, "/layer.oracle.Query/GetDataBefore", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetTimeBasedRewards(ctx context.Context, in *QueryGetTimeBasedRewardsRequest, opts ...grpc.CallOption) (*QueryGetTimeBasedRewardsResponse, error) {
	out := new(QueryGetTimeBasedRewardsResponse)
	err := c.cc.Invoke(ctx, "/layer.oracle.Query/GetTimeBasedRewards", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) CurrentCyclelistQuery(ctx context.Context, in *QueryCurrentCyclelistQueryRequest, opts ...grpc.CallOption) (*QueryCurrentCyclelistQueryResponse, error) {
	out := new(QueryCurrentCyclelistQueryResponse)
	err := c.cc.Invoke(ctx, "/layer.oracle.Query/CurrentCyclelistQuery", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) NextCyclelistQuery(ctx context.Context, in *QueryNextCyclelistQueryRequest, opts ...grpc.CallOption) (*QueryNextCyclelistQueryResponse, error) {
	out := new(QueryNextCyclelistQueryResponse)
	err := c.cc.Invoke(ctx, "/layer.oracle.Query/NextCyclelistQuery", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) RetrieveData(ctx context.Context, in *QueryRetrieveDataRequest, opts ...grpc.CallOption) (*QueryRetrieveDataResponse, error) {
	out := new(QueryRetrieveDataResponse)
	err := c.cc.Invoke(ctx, "/layer.oracle.Query/RetrieveData", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetCurrentAggregateReport(ctx context.Context, in *QueryGetCurrentAggregateReportRequest, opts ...grpc.CallOption) (*QueryGetCurrentAggregateReportResponse, error) {
	out := new(QueryGetCurrentAggregateReportResponse)
	err := c.cc.Invoke(ctx, "/layer.oracle.Query/GetCurrentAggregateReport", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetAggregateBeforeByReporter(ctx context.Context, in *QueryGetAggregateBeforeByReporterRequest, opts ...grpc.CallOption) (*QueryGetAggregateBeforeByReporterResponse, error) {
	out := new(QueryGetAggregateBeforeByReporterResponse)
	err := c.cc.Invoke(ctx, "/layer.oracle.Query/GetAggregateBeforeByReporter", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetQuery(ctx context.Context, in *QueryGetQueryRequest, opts ...grpc.CallOption) (*QueryGetQueryResponse, error) {
	out := new(QueryGetQueryResponse)
	err := c.cc.Invoke(ctx, "/layer.oracle.Query/GetQuery", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) TippedQueries(ctx context.Context, in *QueryTippedQueriesRequest, opts ...grpc.CallOption) (*QueryTippedQueriesResponse, error) {
	out := new(QueryTippedQueriesResponse)
	err := c.cc.Invoke(ctx, "/layer.oracle.Query/TippedQueries", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetReportsByAggregate(ctx context.Context, in *QueryGetReportsByAggregateRequest, opts ...grpc.CallOption) (*QueryGetReportsByAggregateResponse, error) {
	out := new(QueryGetReportsByAggregateResponse)
	err := c.cc.Invoke(ctx, "/layer.oracle.Query/GetReportsByAggregate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetCurrentQueryByQueryId(ctx context.Context, in *QueryGetCurrentQueryByQueryIdRequest, opts ...grpc.CallOption) (*QueryGetCurrentQueryByQueryIdResponse, error) {
	out := new(QueryGetCurrentQueryByQueryIdResponse)
	err := c.cc.Invoke(ctx, "/layer.oracle.Query/GetCurrentQueryByQueryId", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) QueryState(ctx context.Context, in *QueryStateRequest, opts ...grpc.CallOption) (*QueryStateResponse, error) {
	out := new(QueryStateResponse)
	err := c.cc.Invoke(ctx, "/layer.oracle.Query/QueryState", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// QueryServer is the server API for Query service.
// All implementations must embed UnimplementedQueryServer
// for forward compatibility
type QueryServer interface {
	// Parameters queries the parameters of the module.
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
	// Queries a list of GetReportsbyQid items.
	GetReportsbyQid(context.Context, *QueryGetReportsbyQidRequest) (*QueryMicroReportsResponse, error)
	GetReportsbyReporter(context.Context, *QueryGetReportsbyReporterRequest) (*QueryMicroReportsResponse, error)
	GetReportsbyReporterQid(context.Context, *QueryGetReportsbyReporterQidRequest) (*QueryMicroReportsResponse, error)
	// Queries a list of GetCurrentTip items.
	GetCurrentTip(context.Context, *QueryGetCurrentTipRequest) (*QueryGetCurrentTipResponse, error)
	// Queries a list of GetUserTipTotal items.
	GetUserTipTotal(context.Context, *QueryGetUserTipTotalRequest) (*QueryGetUserTipTotalResponse, error)
	// Queries a list of GetAggregatedReport items.
	GetDataBefore(context.Context, *QueryGetDataBeforeRequest) (*QueryGetDataBeforeResponse, error)
	// Queries a list of GetTimeBasedRewards items.
	GetTimeBasedRewards(context.Context, *QueryGetTimeBasedRewardsRequest) (*QueryGetTimeBasedRewardsResponse, error)
	// Queries a list of CurrentCyclelistQuery items.
	CurrentCyclelistQuery(context.Context, *QueryCurrentCyclelistQueryRequest) (*QueryCurrentCyclelistQueryResponse, error)
	// Queries a list of NextCyclelistQuery items.
	NextCyclelistQuery(context.Context, *QueryNextCyclelistQueryRequest) (*QueryNextCyclelistQueryResponse, error)
	RetrieveData(context.Context, *QueryRetrieveDataRequest) (*QueryRetrieveDataResponse, error)
	GetCurrentAggregateReport(context.Context, *QueryGetCurrentAggregateReportRequest) (*QueryGetCurrentAggregateReportResponse, error)
	GetAggregateBeforeByReporter(context.Context, *QueryGetAggregateBeforeByReporterRequest) (*QueryGetAggregateBeforeByReporterResponse, error)
	GetQuery(context.Context, *QueryGetQueryRequest) (*QueryGetQueryResponse, error)
	TippedQueries(context.Context, *QueryTippedQueriesRequest) (*QueryTippedQueriesResponse, error)
	GetReportsByAggregate(context.Context, *QueryGetReportsByAggregateRequest) (*QueryGetReportsByAggregateResponse, error)
	GetCurrentQueryByQueryId(context.Context, *QueryGetCurrentQueryByQueryIdRequest) (*QueryGetCurrentQueryByQueryIdResponse, error)
	QueryState(context.Context, *QueryStateRequest) (*QueryStateResponse, error)
	mustEmbedUnimplementedQueryServer()
}

// UnimplementedQueryServer must be embedded to have forward compatible implementations.
type UnimplementedQueryServer struct {
}

func (UnimplementedQueryServer) Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Params not implemented")
}
func (UnimplementedQueryServer) GetReportsbyQid(context.Context, *QueryGetReportsbyQidRequest) (*QueryMicroReportsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetReportsbyQid not implemented")
}
func (UnimplementedQueryServer) GetReportsbyReporter(context.Context, *QueryGetReportsbyReporterRequest) (*QueryMicroReportsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetReportsbyReporter not implemented")
}
func (UnimplementedQueryServer) GetReportsbyReporterQid(context.Context, *QueryGetReportsbyReporterQidRequest) (*QueryMicroReportsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetReportsbyReporterQid not implemented")
}
func (UnimplementedQueryServer) GetCurrentTip(context.Context, *QueryGetCurrentTipRequest) (*QueryGetCurrentTipResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetCurrentTip not implemented")
}
func (UnimplementedQueryServer) GetUserTipTotal(context.Context, *QueryGetUserTipTotalRequest) (*QueryGetUserTipTotalResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUserTipTotal not implemented")
}
func (UnimplementedQueryServer) GetDataBefore(context.Context, *QueryGetDataBeforeRequest) (*QueryGetDataBeforeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetDataBefore not implemented")
}
func (UnimplementedQueryServer) GetTimeBasedRewards(context.Context, *QueryGetTimeBasedRewardsRequest) (*QueryGetTimeBasedRewardsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTimeBasedRewards not implemented")
}
func (UnimplementedQueryServer) CurrentCyclelistQuery(context.Context, *QueryCurrentCyclelistQueryRequest) (*QueryCurrentCyclelistQueryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CurrentCyclelistQuery not implemented")
}
func (UnimplementedQueryServer) NextCyclelistQuery(context.Context, *QueryNextCyclelistQueryRequest) (*QueryNextCyclelistQueryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NextCyclelistQuery not implemented")
}
func (UnimplementedQueryServer) RetrieveData(context.Context, *QueryRetrieveDataRequest) (*QueryRetrieveDataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RetrieveData not implemented")
}
func (UnimplementedQueryServer) GetCurrentAggregateReport(context.Context, *QueryGetCurrentAggregateReportRequest) (*QueryGetCurrentAggregateReportResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetCurrentAggregateReport not implemented")
}
func (UnimplementedQueryServer) GetAggregateBeforeByReporter(context.Context, *QueryGetAggregateBeforeByReporterRequest) (*QueryGetAggregateBeforeByReporterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetAggregateBeforeByReporter not implemented")
}
func (UnimplementedQueryServer) GetQuery(context.Context, *QueryGetQueryRequest) (*QueryGetQueryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetQuery not implemented")
}
func (UnimplementedQueryServer) TippedQueries(context.Context, *QueryTippedQueriesRequest) (*QueryTippedQueriesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TippedQueries not implemented")
}
func (UnimplementedQueryServer) GetReportsByAggregate(context.Context, *QueryGetReportsByAggregateRequest) (*QueryGetReportsByAggregateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetReportsByAggregate not implemented")
}
func (UnimplementedQueryServer) GetCurrentQueryByQueryId(context.Context, *QueryGetCurrentQueryByQueryIdRequest) (*QueryGetCurrentQueryByQueryIdResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetCurrentQueryByQueryId not implemented")
}
func (UnimplementedQueryServer) QueryState(context.Context, *QueryStateRequest) (*QueryStateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method QueryState not implemented")
}
func (UnimplementedQueryServer) mustEmbedUnimplementedQueryServer() {}

// UnsafeQueryServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to QueryServer will
// result in compilation errors.
type UnsafeQueryServer interface {
	mustEmbedUnimplementedQueryServer()
}

func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&Query_ServiceDesc, srv)
}

func _Query_Params_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Params(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.oracle.Query/Params",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetReportsbyQid_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetReportsbyQidRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetReportsbyQid(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.oracle.Query/GetReportsbyQid",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetReportsbyQid(ctx, req.(*QueryGetReportsbyQidRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetReportsbyReporter_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetReportsbyReporterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetReportsbyReporter(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.oracle.Query/GetReportsbyReporter",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetReportsbyReporter(ctx, req.(*QueryGetReportsbyReporterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetReportsbyReporterQid_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetReportsbyReporterQidRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetReportsbyReporterQid(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.oracle.Query/GetReportsbyReporterQid",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetReportsbyReporterQid(ctx, req.(*QueryGetReportsbyReporterQidRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetCurrentTip_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetCurrentTipRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetCurrentTip(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.oracle.Query/GetCurrentTip",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetCurrentTip(ctx, req.(*QueryGetCurrentTipRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetUserTipTotal_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetUserTipTotalRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetUserTipTotal(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.oracle.Query/GetUserTipTotal",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetUserTipTotal(ctx, req.(*QueryGetUserTipTotalRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetDataBefore_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetDataBeforeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetDataBefore(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.oracle.Query/GetDataBefore",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetDataBefore(ctx, req.(*QueryGetDataBeforeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetTimeBasedRewards_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetTimeBasedRewardsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetTimeBasedRewards(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.oracle.Query/GetTimeBasedRewards",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetTimeBasedRewards(ctx, req.(*QueryGetTimeBasedRewardsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_CurrentCyclelistQuery_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryCurrentCyclelistQueryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).CurrentCyclelistQuery(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.oracle.Query/CurrentCyclelistQuery",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).CurrentCyclelistQuery(ctx, req.(*QueryCurrentCyclelistQueryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_NextCyclelistQuery_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryNextCyclelistQueryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).NextCyclelistQuery(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.oracle.Query/NextCyclelistQuery",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).NextCyclelistQuery(ctx, req.(*QueryNextCyclelistQueryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_RetrieveData_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryRetrieveDataRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).RetrieveData(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.oracle.Query/RetrieveData",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).RetrieveData(ctx, req.(*QueryRetrieveDataRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetCurrentAggregateReport_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetCurrentAggregateReportRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetCurrentAggregateReport(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.oracle.Query/GetCurrentAggregateReport",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetCurrentAggregateReport(ctx, req.(*QueryGetCurrentAggregateReportRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetAggregateBeforeByReporter_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetAggregateBeforeByReporterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetAggregateBeforeByReporter(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.oracle.Query/GetAggregateBeforeByReporter",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetAggregateBeforeByReporter(ctx, req.(*QueryGetAggregateBeforeByReporterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetQuery_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetQueryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetQuery(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.oracle.Query/GetQuery",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetQuery(ctx, req.(*QueryGetQueryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_TippedQueries_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryTippedQueriesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).TippedQueries(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.oracle.Query/TippedQueries",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).TippedQueries(ctx, req.(*QueryTippedQueriesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetReportsByAggregate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetReportsByAggregateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetReportsByAggregate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.oracle.Query/GetReportsByAggregate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetReportsByAggregate(ctx, req.(*QueryGetReportsByAggregateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetCurrentQueryByQueryId_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetCurrentQueryByQueryIdRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetCurrentQueryByQueryId(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.oracle.Query/GetCurrentQueryByQueryId",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetCurrentQueryByQueryId(ctx, req.(*QueryGetCurrentQueryByQueryIdRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_QueryState_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryStateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).QueryState(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.oracle.Query/QueryState",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).QueryState(ctx, req.(*QueryStateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Query_ServiceDesc is the grpc.ServiceDesc for Query service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Query_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "layer.oracle.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Params",
			Handler:    _Query_Params_Handler,
		},
		{
			MethodName: "GetReportsbyQid",
			Handler:    _Query_GetReportsbyQid_Handler,
		},
		{
			MethodName: "GetReportsbyReporter",
			Handler:    _Query_GetReportsbyReporter_Handler,
		},
		{
			MethodName: "GetReportsbyReporterQid",
			Handler:    _Query_GetReportsbyReporterQid_Handler,
		},
		{
			MethodName: "GetCurrentTip",
			Handler:    _Query_GetCurrentTip_Handler,
		},
		{
			MethodName: "GetUserTipTotal",
			Handler:    _Query_GetUserTipTotal_Handler,
		},
		{
			MethodName: "GetDataBefore",
			Handler:    _Query_GetDataBefore_Handler,
		},
		{
			MethodName: "GetTimeBasedRewards",
			Handler:    _Query_GetTimeBasedRewards_Handler,
		},
		{
			MethodName: "CurrentCyclelistQuery",
			Handler:    _Query_CurrentCyclelistQuery_Handler,
		},
		{
			MethodName: "NextCyclelistQuery",
			Handler:    _Query_NextCyclelistQuery_Handler,
		},
		{
			MethodName: "RetrieveData",
			Handler:    _Query_RetrieveData_Handler,
		},
		{
			MethodName: "GetCurrentAggregateReport",
			Handler:    _Query_GetCurrentAggregateReport_Handler,
		},
		{
			MethodName: "GetAggregateBeforeByReporter",
			Handler:    _Query_GetAggregateBeforeByReporter_Handler,
		},
		{
			MethodName: "GetQuery",
			Handler:    _Query_GetQuery_Handler,
		},
		{
			MethodName: "TippedQueries",
			Handler:    _Query_TippedQueries_Handler,
		},
		{
			MethodName: "GetReportsByAggregate",
			Handler:    _Query_GetReportsByAggregate_Handler,
		},
		{
			MethodName: "GetCurrentQueryByQueryId",
			Handler:    _Query_GetCurrentQueryByQueryId_Handler,
		},
		{
			MethodName: "QueryState",
			Handler:    _Query_QueryState_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "layer/oracle/query.proto",
}
