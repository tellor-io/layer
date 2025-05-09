// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package bridge

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
	// Queries a list of GetEvmValidators items.
	GetEvmValidators(ctx context.Context, in *QueryGetEvmValidatorsRequest, opts ...grpc.CallOption) (*QueryGetEvmValidatorsResponse, error)
	// Queries the latest validator checkpoint
	GetValidatorCheckpoint(ctx context.Context, in *QueryGetValidatorCheckpointRequest, opts ...grpc.CallOption) (*QueryGetValidatorCheckpointResponse, error)
	// Queries the validator checkpoint params for a given timestamp
	GetValidatorCheckpointParams(ctx context.Context, in *QueryGetValidatorCheckpointParamsRequest, opts ...grpc.CallOption) (*QueryGetValidatorCheckpointParamsResponse, error)
	// Queries the validator timestamp by index
	GetValidatorTimestampByIndex(ctx context.Context, in *QueryGetValidatorTimestampByIndexRequest, opts ...grpc.CallOption) (*QueryGetValidatorTimestampByIndexResponse, error)
	// Queries the validator set signatures for a given timestamp
	GetValsetSigs(ctx context.Context, in *QueryGetValsetSigsRequest, opts ...grpc.CallOption) (*QueryGetValsetSigsResponse, error)
	// Queries the evm address by validator address
	GetEvmAddressByValidatorAddress(ctx context.Context, in *QueryGetEvmAddressByValidatorAddressRequest, opts ...grpc.CallOption) (*QueryGetEvmAddressByValidatorAddressResponse, error)
	// Queries the validator set by timestamp
	GetValsetByTimestamp(ctx context.Context, in *QueryGetValsetByTimestampRequest, opts ...grpc.CallOption) (*QueryGetValsetByTimestampResponse, error)
	// Queries a list of snapshots by report query id and timestamp
	GetSnapshotsByReport(ctx context.Context, in *QueryGetSnapshotsByReportRequest, opts ...grpc.CallOption) (*QueryGetSnapshotsByReportResponse, error)
	// Queries attestation data by snapshot
	GetAttestationDataBySnapshot(ctx context.Context, in *QueryGetAttestationDataBySnapshotRequest, opts ...grpc.CallOption) (*QueryGetAttestationDataBySnapshotResponse, error)
	// Queries the set of attestations by snapshot
	GetAttestationsBySnapshot(ctx context.Context, in *QueryGetAttestationsBySnapshotRequest, opts ...grpc.CallOption) (*QueryGetAttestationsBySnapshotResponse, error)
	// Queries the validator set index by timestamp
	GetValidatorSetIndexByTimestamp(ctx context.Context, in *QueryGetValidatorSetIndexByTimestampRequest, opts ...grpc.CallOption) (*QueryGetValidatorSetIndexByTimestampResponse, error)
	// Queries the current validator set timestamp
	GetCurrentValidatorSetTimestamp(ctx context.Context, in *QueryGetCurrentValidatorSetTimestampRequest, opts ...grpc.CallOption) (*QueryGetCurrentValidatorSetTimestampResponse, error)
	// Queries the snapshot limit
	GetSnapshotLimit(ctx context.Context, in *QueryGetSnapshotLimitRequest, opts ...grpc.CallOption) (*QueryGetSnapshotLimitResponse, error)
	// Queries whether a deposit is claimed
	GetDepositClaimed(ctx context.Context, in *QueryGetDepositClaimedRequest, opts ...grpc.CallOption) (*QueryGetDepositClaimedResponse, error)
	GetLastWithdrawalId(ctx context.Context, in *QueryGetLastWithdrawalIdRequest, opts ...grpc.CallOption) (*QueryGetLastWithdrawalIdResponse, error)
}

type queryClient struct {
	cc grpc.ClientConnInterface
}

func NewQueryClient(cc grpc.ClientConnInterface) QueryClient {
	return &queryClient{cc}
}

func (c *queryClient) Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error) {
	out := new(QueryParamsResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/Params", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetEvmValidators(ctx context.Context, in *QueryGetEvmValidatorsRequest, opts ...grpc.CallOption) (*QueryGetEvmValidatorsResponse, error) {
	out := new(QueryGetEvmValidatorsResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/GetEvmValidators", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetValidatorCheckpoint(ctx context.Context, in *QueryGetValidatorCheckpointRequest, opts ...grpc.CallOption) (*QueryGetValidatorCheckpointResponse, error) {
	out := new(QueryGetValidatorCheckpointResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/GetValidatorCheckpoint", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetValidatorCheckpointParams(ctx context.Context, in *QueryGetValidatorCheckpointParamsRequest, opts ...grpc.CallOption) (*QueryGetValidatorCheckpointParamsResponse, error) {
	out := new(QueryGetValidatorCheckpointParamsResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/GetValidatorCheckpointParams", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetValidatorTimestampByIndex(ctx context.Context, in *QueryGetValidatorTimestampByIndexRequest, opts ...grpc.CallOption) (*QueryGetValidatorTimestampByIndexResponse, error) {
	out := new(QueryGetValidatorTimestampByIndexResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/GetValidatorTimestampByIndex", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetValsetSigs(ctx context.Context, in *QueryGetValsetSigsRequest, opts ...grpc.CallOption) (*QueryGetValsetSigsResponse, error) {
	out := new(QueryGetValsetSigsResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/GetValsetSigs", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetEvmAddressByValidatorAddress(ctx context.Context, in *QueryGetEvmAddressByValidatorAddressRequest, opts ...grpc.CallOption) (*QueryGetEvmAddressByValidatorAddressResponse, error) {
	out := new(QueryGetEvmAddressByValidatorAddressResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/GetEvmAddressByValidatorAddress", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetValsetByTimestamp(ctx context.Context, in *QueryGetValsetByTimestampRequest, opts ...grpc.CallOption) (*QueryGetValsetByTimestampResponse, error) {
	out := new(QueryGetValsetByTimestampResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/GetValsetByTimestamp", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetSnapshotsByReport(ctx context.Context, in *QueryGetSnapshotsByReportRequest, opts ...grpc.CallOption) (*QueryGetSnapshotsByReportResponse, error) {
	out := new(QueryGetSnapshotsByReportResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/GetSnapshotsByReport", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetAttestationDataBySnapshot(ctx context.Context, in *QueryGetAttestationDataBySnapshotRequest, opts ...grpc.CallOption) (*QueryGetAttestationDataBySnapshotResponse, error) {
	out := new(QueryGetAttestationDataBySnapshotResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/GetAttestationDataBySnapshot", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetAttestationsBySnapshot(ctx context.Context, in *QueryGetAttestationsBySnapshotRequest, opts ...grpc.CallOption) (*QueryGetAttestationsBySnapshotResponse, error) {
	out := new(QueryGetAttestationsBySnapshotResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/GetAttestationsBySnapshot", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetValidatorSetIndexByTimestamp(ctx context.Context, in *QueryGetValidatorSetIndexByTimestampRequest, opts ...grpc.CallOption) (*QueryGetValidatorSetIndexByTimestampResponse, error) {
	out := new(QueryGetValidatorSetIndexByTimestampResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/GetValidatorSetIndexByTimestamp", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetCurrentValidatorSetTimestamp(ctx context.Context, in *QueryGetCurrentValidatorSetTimestampRequest, opts ...grpc.CallOption) (*QueryGetCurrentValidatorSetTimestampResponse, error) {
	out := new(QueryGetCurrentValidatorSetTimestampResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/GetCurrentValidatorSetTimestamp", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetSnapshotLimit(ctx context.Context, in *QueryGetSnapshotLimitRequest, opts ...grpc.CallOption) (*QueryGetSnapshotLimitResponse, error) {
	out := new(QueryGetSnapshotLimitResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/GetSnapshotLimit", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetDepositClaimed(ctx context.Context, in *QueryGetDepositClaimedRequest, opts ...grpc.CallOption) (*QueryGetDepositClaimedResponse, error) {
	out := new(QueryGetDepositClaimedResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/GetDepositClaimed", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetLastWithdrawalId(ctx context.Context, in *QueryGetLastWithdrawalIdRequest, opts ...grpc.CallOption) (*QueryGetLastWithdrawalIdResponse, error) {
	out := new(QueryGetLastWithdrawalIdResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/GetLastWithdrawalId", in, out, opts...)
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
	// Queries a list of GetEvmValidators items.
	GetEvmValidators(context.Context, *QueryGetEvmValidatorsRequest) (*QueryGetEvmValidatorsResponse, error)
	// Queries the latest validator checkpoint
	GetValidatorCheckpoint(context.Context, *QueryGetValidatorCheckpointRequest) (*QueryGetValidatorCheckpointResponse, error)
	// Queries the validator checkpoint params for a given timestamp
	GetValidatorCheckpointParams(context.Context, *QueryGetValidatorCheckpointParamsRequest) (*QueryGetValidatorCheckpointParamsResponse, error)
	// Queries the validator timestamp by index
	GetValidatorTimestampByIndex(context.Context, *QueryGetValidatorTimestampByIndexRequest) (*QueryGetValidatorTimestampByIndexResponse, error)
	// Queries the validator set signatures for a given timestamp
	GetValsetSigs(context.Context, *QueryGetValsetSigsRequest) (*QueryGetValsetSigsResponse, error)
	// Queries the evm address by validator address
	GetEvmAddressByValidatorAddress(context.Context, *QueryGetEvmAddressByValidatorAddressRequest) (*QueryGetEvmAddressByValidatorAddressResponse, error)
	// Queries the validator set by timestamp
	GetValsetByTimestamp(context.Context, *QueryGetValsetByTimestampRequest) (*QueryGetValsetByTimestampResponse, error)
	// Queries a list of snapshots by report query id and timestamp
	GetSnapshotsByReport(context.Context, *QueryGetSnapshotsByReportRequest) (*QueryGetSnapshotsByReportResponse, error)
	// Queries attestation data by snapshot
	GetAttestationDataBySnapshot(context.Context, *QueryGetAttestationDataBySnapshotRequest) (*QueryGetAttestationDataBySnapshotResponse, error)
	// Queries the set of attestations by snapshot
	GetAttestationsBySnapshot(context.Context, *QueryGetAttestationsBySnapshotRequest) (*QueryGetAttestationsBySnapshotResponse, error)
	// Queries the validator set index by timestamp
	GetValidatorSetIndexByTimestamp(context.Context, *QueryGetValidatorSetIndexByTimestampRequest) (*QueryGetValidatorSetIndexByTimestampResponse, error)
	// Queries the current validator set timestamp
	GetCurrentValidatorSetTimestamp(context.Context, *QueryGetCurrentValidatorSetTimestampRequest) (*QueryGetCurrentValidatorSetTimestampResponse, error)
	// Queries the snapshot limit
	GetSnapshotLimit(context.Context, *QueryGetSnapshotLimitRequest) (*QueryGetSnapshotLimitResponse, error)
	// Queries whether a deposit is claimed
	GetDepositClaimed(context.Context, *QueryGetDepositClaimedRequest) (*QueryGetDepositClaimedResponse, error)
	GetLastWithdrawalId(context.Context, *QueryGetLastWithdrawalIdRequest) (*QueryGetLastWithdrawalIdResponse, error)
	mustEmbedUnimplementedQueryServer()
}

// UnimplementedQueryServer must be embedded to have forward compatible implementations.
type UnimplementedQueryServer struct {
}

func (UnimplementedQueryServer) Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Params not implemented")
}
func (UnimplementedQueryServer) GetEvmValidators(context.Context, *QueryGetEvmValidatorsRequest) (*QueryGetEvmValidatorsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetEvmValidators not implemented")
}
func (UnimplementedQueryServer) GetValidatorCheckpoint(context.Context, *QueryGetValidatorCheckpointRequest) (*QueryGetValidatorCheckpointResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetValidatorCheckpoint not implemented")
}
func (UnimplementedQueryServer) GetValidatorCheckpointParams(context.Context, *QueryGetValidatorCheckpointParamsRequest) (*QueryGetValidatorCheckpointParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetValidatorCheckpointParams not implemented")
}
func (UnimplementedQueryServer) GetValidatorTimestampByIndex(context.Context, *QueryGetValidatorTimestampByIndexRequest) (*QueryGetValidatorTimestampByIndexResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetValidatorTimestampByIndex not implemented")
}
func (UnimplementedQueryServer) GetValsetSigs(context.Context, *QueryGetValsetSigsRequest) (*QueryGetValsetSigsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetValsetSigs not implemented")
}
func (UnimplementedQueryServer) GetEvmAddressByValidatorAddress(context.Context, *QueryGetEvmAddressByValidatorAddressRequest) (*QueryGetEvmAddressByValidatorAddressResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetEvmAddressByValidatorAddress not implemented")
}
func (UnimplementedQueryServer) GetValsetByTimestamp(context.Context, *QueryGetValsetByTimestampRequest) (*QueryGetValsetByTimestampResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetValsetByTimestamp not implemented")
}
func (UnimplementedQueryServer) GetSnapshotsByReport(context.Context, *QueryGetSnapshotsByReportRequest) (*QueryGetSnapshotsByReportResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetSnapshotsByReport not implemented")
}
func (UnimplementedQueryServer) GetAttestationDataBySnapshot(context.Context, *QueryGetAttestationDataBySnapshotRequest) (*QueryGetAttestationDataBySnapshotResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetAttestationDataBySnapshot not implemented")
}
func (UnimplementedQueryServer) GetAttestationsBySnapshot(context.Context, *QueryGetAttestationsBySnapshotRequest) (*QueryGetAttestationsBySnapshotResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetAttestationsBySnapshot not implemented")
}
func (UnimplementedQueryServer) GetValidatorSetIndexByTimestamp(context.Context, *QueryGetValidatorSetIndexByTimestampRequest) (*QueryGetValidatorSetIndexByTimestampResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetValidatorSetIndexByTimestamp not implemented")
}
func (UnimplementedQueryServer) GetCurrentValidatorSetTimestamp(context.Context, *QueryGetCurrentValidatorSetTimestampRequest) (*QueryGetCurrentValidatorSetTimestampResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetCurrentValidatorSetTimestamp not implemented")
}
func (UnimplementedQueryServer) GetSnapshotLimit(context.Context, *QueryGetSnapshotLimitRequest) (*QueryGetSnapshotLimitResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetSnapshotLimit not implemented")
}
func (UnimplementedQueryServer) GetDepositClaimed(context.Context, *QueryGetDepositClaimedRequest) (*QueryGetDepositClaimedResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetDepositClaimed not implemented")
}
func (UnimplementedQueryServer) GetLastWithdrawalId(context.Context, *QueryGetLastWithdrawalIdRequest) (*QueryGetLastWithdrawalIdResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetLastWithdrawalId not implemented")
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
		FullMethod: "/layer.bridge.Query/Params",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetEvmValidators_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetEvmValidatorsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetEvmValidators(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Query/GetEvmValidators",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetEvmValidators(ctx, req.(*QueryGetEvmValidatorsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetValidatorCheckpoint_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetValidatorCheckpointRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetValidatorCheckpoint(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Query/GetValidatorCheckpoint",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetValidatorCheckpoint(ctx, req.(*QueryGetValidatorCheckpointRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetValidatorCheckpointParams_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetValidatorCheckpointParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetValidatorCheckpointParams(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Query/GetValidatorCheckpointParams",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetValidatorCheckpointParams(ctx, req.(*QueryGetValidatorCheckpointParamsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetValidatorTimestampByIndex_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetValidatorTimestampByIndexRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetValidatorTimestampByIndex(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Query/GetValidatorTimestampByIndex",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetValidatorTimestampByIndex(ctx, req.(*QueryGetValidatorTimestampByIndexRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetValsetSigs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetValsetSigsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetValsetSigs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Query/GetValsetSigs",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetValsetSigs(ctx, req.(*QueryGetValsetSigsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetEvmAddressByValidatorAddress_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetEvmAddressByValidatorAddressRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetEvmAddressByValidatorAddress(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Query/GetEvmAddressByValidatorAddress",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetEvmAddressByValidatorAddress(ctx, req.(*QueryGetEvmAddressByValidatorAddressRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetValsetByTimestamp_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetValsetByTimestampRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetValsetByTimestamp(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Query/GetValsetByTimestamp",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetValsetByTimestamp(ctx, req.(*QueryGetValsetByTimestampRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetSnapshotsByReport_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetSnapshotsByReportRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetSnapshotsByReport(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Query/GetSnapshotsByReport",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetSnapshotsByReport(ctx, req.(*QueryGetSnapshotsByReportRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetAttestationDataBySnapshot_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetAttestationDataBySnapshotRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetAttestationDataBySnapshot(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Query/GetAttestationDataBySnapshot",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetAttestationDataBySnapshot(ctx, req.(*QueryGetAttestationDataBySnapshotRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetAttestationsBySnapshot_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetAttestationsBySnapshotRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetAttestationsBySnapshot(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Query/GetAttestationsBySnapshot",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetAttestationsBySnapshot(ctx, req.(*QueryGetAttestationsBySnapshotRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetValidatorSetIndexByTimestamp_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetValidatorSetIndexByTimestampRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetValidatorSetIndexByTimestamp(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Query/GetValidatorSetIndexByTimestamp",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetValidatorSetIndexByTimestamp(ctx, req.(*QueryGetValidatorSetIndexByTimestampRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetCurrentValidatorSetTimestamp_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetCurrentValidatorSetTimestampRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetCurrentValidatorSetTimestamp(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Query/GetCurrentValidatorSetTimestamp",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetCurrentValidatorSetTimestamp(ctx, req.(*QueryGetCurrentValidatorSetTimestampRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetSnapshotLimit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetSnapshotLimitRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetSnapshotLimit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Query/GetSnapshotLimit",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetSnapshotLimit(ctx, req.(*QueryGetSnapshotLimitRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetDepositClaimed_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetDepositClaimedRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetDepositClaimed(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Query/GetDepositClaimed",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetDepositClaimed(ctx, req.(*QueryGetDepositClaimedRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetLastWithdrawalId_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetLastWithdrawalIdRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetLastWithdrawalId(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Query/GetLastWithdrawalId",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetLastWithdrawalId(ctx, req.(*QueryGetLastWithdrawalIdRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Query_ServiceDesc is the grpc.ServiceDesc for Query service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Query_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "layer.bridge.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Params",
			Handler:    _Query_Params_Handler,
		},
		{
			MethodName: "GetEvmValidators",
			Handler:    _Query_GetEvmValidators_Handler,
		},
		{
			MethodName: "GetValidatorCheckpoint",
			Handler:    _Query_GetValidatorCheckpoint_Handler,
		},
		{
			MethodName: "GetValidatorCheckpointParams",
			Handler:    _Query_GetValidatorCheckpointParams_Handler,
		},
		{
			MethodName: "GetValidatorTimestampByIndex",
			Handler:    _Query_GetValidatorTimestampByIndex_Handler,
		},
		{
			MethodName: "GetValsetSigs",
			Handler:    _Query_GetValsetSigs_Handler,
		},
		{
			MethodName: "GetEvmAddressByValidatorAddress",
			Handler:    _Query_GetEvmAddressByValidatorAddress_Handler,
		},
		{
			MethodName: "GetValsetByTimestamp",
			Handler:    _Query_GetValsetByTimestamp_Handler,
		},
		{
			MethodName: "GetSnapshotsByReport",
			Handler:    _Query_GetSnapshotsByReport_Handler,
		},
		{
			MethodName: "GetAttestationDataBySnapshot",
			Handler:    _Query_GetAttestationDataBySnapshot_Handler,
		},
		{
			MethodName: "GetAttestationsBySnapshot",
			Handler:    _Query_GetAttestationsBySnapshot_Handler,
		},
		{
			MethodName: "GetValidatorSetIndexByTimestamp",
			Handler:    _Query_GetValidatorSetIndexByTimestamp_Handler,
		},
		{
			MethodName: "GetCurrentValidatorSetTimestamp",
			Handler:    _Query_GetCurrentValidatorSetTimestamp_Handler,
		},
		{
			MethodName: "GetSnapshotLimit",
			Handler:    _Query_GetSnapshotLimit_Handler,
		},
		{
			MethodName: "GetDepositClaimed",
			Handler:    _Query_GetDepositClaimed_Handler,
		},
		{
			MethodName: "GetLastWithdrawalId",
			Handler:    _Query_GetLastWithdrawalId_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "layer/bridge/query.proto",
}
