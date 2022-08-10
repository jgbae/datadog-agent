// Code generated by MockGen. DO NOT EDIT.
// Source: /Users/kyle.ames/go/src/github.com/DataDog/datadog-agent/pkg/proto/pbgo/api.pb.go

// Package mock_pbgo is a generated GoMock package.
package mock_pbgo

import (
	context "context"
	reflect "reflect"

	pbgo "github.com/DataDog/datadog-agent/pkg/proto/pbgo"
	gomock "github.com/golang/mock/gomock"
	grpc "google.golang.org/grpc"
	metadata "google.golang.org/grpc/metadata"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// MockAgentClient is a mock of AgentClient interface.
type MockAgentClient struct {
	ctrl     *gomock.Controller
	recorder *MockAgentClientMockRecorder
}

// MockAgentClientMockRecorder is the mock recorder for MockAgentClient.
type MockAgentClientMockRecorder struct {
	mock *MockAgentClient
}

// NewMockAgentClient creates a new mock instance.
func NewMockAgentClient(ctrl *gomock.Controller) *MockAgentClient {
	mock := &MockAgentClient{ctrl: ctrl}
	mock.recorder = &MockAgentClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAgentClient) EXPECT() *MockAgentClientMockRecorder {
	return m.recorder
}

// GetHostname mocks base method.
func (m *MockAgentClient) GetHostname(ctx context.Context, in *pbgo.HostnameRequest, opts ...grpc.CallOption) (*pbgo.HostnameReply, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetHostname", varargs...)
	ret0, _ := ret[0].(*pbgo.HostnameReply)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetHostname indicates an expected call of GetHostname.
func (mr *MockAgentClientMockRecorder) GetHostname(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHostname", reflect.TypeOf((*MockAgentClient)(nil).GetHostname), varargs...)
}

// MockAgentServer is a mock of AgentServer interface.
type MockAgentServer struct {
	ctrl     *gomock.Controller
	recorder *MockAgentServerMockRecorder
}

// MockAgentServerMockRecorder is the mock recorder for MockAgentServer.
type MockAgentServerMockRecorder struct {
	mock *MockAgentServer
}

// NewMockAgentServer creates a new mock instance.
func NewMockAgentServer(ctrl *gomock.Controller) *MockAgentServer {
	mock := &MockAgentServer{ctrl: ctrl}
	mock.recorder = &MockAgentServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAgentServer) EXPECT() *MockAgentServerMockRecorder {
	return m.recorder
}

// GetHostname mocks base method.
func (m *MockAgentServer) GetHostname(arg0 context.Context, arg1 *pbgo.HostnameRequest) (*pbgo.HostnameReply, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHostname", arg0, arg1)
	ret0, _ := ret[0].(*pbgo.HostnameReply)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetHostname indicates an expected call of GetHostname.
func (mr *MockAgentServerMockRecorder) GetHostname(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHostname", reflect.TypeOf((*MockAgentServer)(nil).GetHostname), arg0, arg1)
}

// MockAgentSecureClient is a mock of AgentSecureClient interface.
type MockAgentSecureClient struct {
	ctrl     *gomock.Controller
	recorder *MockAgentSecureClientMockRecorder
}

// MockAgentSecureClientMockRecorder is the mock recorder for MockAgentSecureClient.
type MockAgentSecureClientMockRecorder struct {
	mock *MockAgentSecureClient
}

// NewMockAgentSecureClient creates a new mock instance.
func NewMockAgentSecureClient(ctrl *gomock.Controller) *MockAgentSecureClient {
	mock := &MockAgentSecureClient{ctrl: ctrl}
	mock.recorder = &MockAgentSecureClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAgentSecureClient) EXPECT() *MockAgentSecureClientMockRecorder {
	return m.recorder
}

// ClientGetConfigs mocks base method.
func (m *MockAgentSecureClient) ClientGetConfigs(ctx context.Context, in *pbgo.ClientGetConfigsRequest, opts ...grpc.CallOption) (*pbgo.ClientGetConfigsResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ClientGetConfigs", varargs...)
	ret0, _ := ret[0].(*pbgo.ClientGetConfigsResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ClientGetConfigs indicates an expected call of ClientGetConfigs.
func (mr *MockAgentSecureClientMockRecorder) ClientGetConfigs(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ClientGetConfigs", reflect.TypeOf((*MockAgentSecureClient)(nil).ClientGetConfigs), varargs...)
}

// DogstatsdCaptureTrigger mocks base method.
func (m *MockAgentSecureClient) DogstatsdCaptureTrigger(ctx context.Context, in *pbgo.CaptureTriggerRequest, opts ...grpc.CallOption) (*pbgo.CaptureTriggerResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DogstatsdCaptureTrigger", varargs...)
	ret0, _ := ret[0].(*pbgo.CaptureTriggerResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DogstatsdCaptureTrigger indicates an expected call of DogstatsdCaptureTrigger.
func (mr *MockAgentSecureClientMockRecorder) DogstatsdCaptureTrigger(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DogstatsdCaptureTrigger", reflect.TypeOf((*MockAgentSecureClient)(nil).DogstatsdCaptureTrigger), varargs...)
}

// DogstatsdSetTaggerState mocks base method.
func (m *MockAgentSecureClient) DogstatsdSetTaggerState(ctx context.Context, in *pbgo.TaggerState, opts ...grpc.CallOption) (*pbgo.TaggerStateResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DogstatsdSetTaggerState", varargs...)
	ret0, _ := ret[0].(*pbgo.TaggerStateResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DogstatsdSetTaggerState indicates an expected call of DogstatsdSetTaggerState.
func (mr *MockAgentSecureClientMockRecorder) DogstatsdSetTaggerState(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DogstatsdSetTaggerState", reflect.TypeOf((*MockAgentSecureClient)(nil).DogstatsdSetTaggerState), varargs...)
}

// GetConfigState mocks base method.
func (m *MockAgentSecureClient) GetConfigState(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*pbgo.GetStateConfigResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetConfigState", varargs...)
	ret0, _ := ret[0].(*pbgo.GetStateConfigResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetConfigState indicates an expected call of GetConfigState.
func (mr *MockAgentSecureClientMockRecorder) GetConfigState(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetConfigState", reflect.TypeOf((*MockAgentSecureClient)(nil).GetConfigState), varargs...)
}

// TaggerFetchEntity mocks base method.
func (m *MockAgentSecureClient) TaggerFetchEntity(ctx context.Context, in *pbgo.FetchEntityRequest, opts ...grpc.CallOption) (*pbgo.FetchEntityResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "TaggerFetchEntity", varargs...)
	ret0, _ := ret[0].(*pbgo.FetchEntityResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// TaggerFetchEntity indicates an expected call of TaggerFetchEntity.
func (mr *MockAgentSecureClientMockRecorder) TaggerFetchEntity(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TaggerFetchEntity", reflect.TypeOf((*MockAgentSecureClient)(nil).TaggerFetchEntity), varargs...)
}

// TaggerStreamEntities mocks base method.
func (m *MockAgentSecureClient) TaggerStreamEntities(ctx context.Context, in *pbgo.StreamTagsRequest, opts ...grpc.CallOption) (pbgo.AgentSecure_TaggerStreamEntitiesClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "TaggerStreamEntities", varargs...)
	ret0, _ := ret[0].(pbgo.AgentSecure_TaggerStreamEntitiesClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// TaggerStreamEntities indicates an expected call of TaggerStreamEntities.
func (mr *MockAgentSecureClientMockRecorder) TaggerStreamEntities(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TaggerStreamEntities", reflect.TypeOf((*MockAgentSecureClient)(nil).TaggerStreamEntities), varargs...)
}

// MockAgentSecure_TaggerStreamEntitiesClient is a mock of AgentSecure_TaggerStreamEntitiesClient interface.
type MockAgentSecure_TaggerStreamEntitiesClient struct {
	ctrl     *gomock.Controller
	recorder *MockAgentSecure_TaggerStreamEntitiesClientMockRecorder
}

// MockAgentSecure_TaggerStreamEntitiesClientMockRecorder is the mock recorder for MockAgentSecure_TaggerStreamEntitiesClient.
type MockAgentSecure_TaggerStreamEntitiesClientMockRecorder struct {
	mock *MockAgentSecure_TaggerStreamEntitiesClient
}

// NewMockAgentSecure_TaggerStreamEntitiesClient creates a new mock instance.
func NewMockAgentSecure_TaggerStreamEntitiesClient(ctrl *gomock.Controller) *MockAgentSecure_TaggerStreamEntitiesClient {
	mock := &MockAgentSecure_TaggerStreamEntitiesClient{ctrl: ctrl}
	mock.recorder = &MockAgentSecure_TaggerStreamEntitiesClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAgentSecure_TaggerStreamEntitiesClient) EXPECT() *MockAgentSecure_TaggerStreamEntitiesClientMockRecorder {
	return m.recorder
}

// CloseSend mocks base method.
func (m *MockAgentSecure_TaggerStreamEntitiesClient) CloseSend() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloseSend")
	ret0, _ := ret[0].(error)
	return ret0
}

// CloseSend indicates an expected call of CloseSend.
func (mr *MockAgentSecure_TaggerStreamEntitiesClientMockRecorder) CloseSend() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloseSend", reflect.TypeOf((*MockAgentSecure_TaggerStreamEntitiesClient)(nil).CloseSend))
}

// Context mocks base method.
func (m *MockAgentSecure_TaggerStreamEntitiesClient) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context.
func (mr *MockAgentSecure_TaggerStreamEntitiesClientMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockAgentSecure_TaggerStreamEntitiesClient)(nil).Context))
}

// Header mocks base method.
func (m *MockAgentSecure_TaggerStreamEntitiesClient) Header() (metadata.MD, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Header")
	ret0, _ := ret[0].(metadata.MD)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Header indicates an expected call of Header.
func (mr *MockAgentSecure_TaggerStreamEntitiesClientMockRecorder) Header() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Header", reflect.TypeOf((*MockAgentSecure_TaggerStreamEntitiesClient)(nil).Header))
}

// Recv mocks base method.
func (m *MockAgentSecure_TaggerStreamEntitiesClient) Recv() (*pbgo.StreamTagsResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Recv")
	ret0, _ := ret[0].(*pbgo.StreamTagsResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Recv indicates an expected call of Recv.
func (mr *MockAgentSecure_TaggerStreamEntitiesClientMockRecorder) Recv() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Recv", reflect.TypeOf((*MockAgentSecure_TaggerStreamEntitiesClient)(nil).Recv))
}

// RecvMsg mocks base method.
func (m_2 *MockAgentSecure_TaggerStreamEntitiesClient) RecvMsg(m interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "RecvMsg", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecvMsg indicates an expected call of RecvMsg.
func (mr *MockAgentSecure_TaggerStreamEntitiesClientMockRecorder) RecvMsg(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecvMsg", reflect.TypeOf((*MockAgentSecure_TaggerStreamEntitiesClient)(nil).RecvMsg), m)
}

// SendMsg mocks base method.
func (m_2 *MockAgentSecure_TaggerStreamEntitiesClient) SendMsg(m interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "SendMsg", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMsg indicates an expected call of SendMsg.
func (mr *MockAgentSecure_TaggerStreamEntitiesClientMockRecorder) SendMsg(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMsg", reflect.TypeOf((*MockAgentSecure_TaggerStreamEntitiesClient)(nil).SendMsg), m)
}

// Trailer mocks base method.
func (m *MockAgentSecure_TaggerStreamEntitiesClient) Trailer() metadata.MD {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Trailer")
	ret0, _ := ret[0].(metadata.MD)
	return ret0
}

// Trailer indicates an expected call of Trailer.
func (mr *MockAgentSecure_TaggerStreamEntitiesClientMockRecorder) Trailer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Trailer", reflect.TypeOf((*MockAgentSecure_TaggerStreamEntitiesClient)(nil).Trailer))
}

// MockAgentSecureServer is a mock of AgentSecureServer interface.
type MockAgentSecureServer struct {
	ctrl     *gomock.Controller
	recorder *MockAgentSecureServerMockRecorder
}

// MockAgentSecureServerMockRecorder is the mock recorder for MockAgentSecureServer.
type MockAgentSecureServerMockRecorder struct {
	mock *MockAgentSecureServer
}

// NewMockAgentSecureServer creates a new mock instance.
func NewMockAgentSecureServer(ctrl *gomock.Controller) *MockAgentSecureServer {
	mock := &MockAgentSecureServer{ctrl: ctrl}
	mock.recorder = &MockAgentSecureServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAgentSecureServer) EXPECT() *MockAgentSecureServerMockRecorder {
	return m.recorder
}

// ClientGetConfigs mocks base method.
func (m *MockAgentSecureServer) ClientGetConfigs(arg0 context.Context, arg1 *pbgo.ClientGetConfigsRequest) (*pbgo.ClientGetConfigsResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ClientGetConfigs", arg0, arg1)
	ret0, _ := ret[0].(*pbgo.ClientGetConfigsResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ClientGetConfigs indicates an expected call of ClientGetConfigs.
func (mr *MockAgentSecureServerMockRecorder) ClientGetConfigs(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ClientGetConfigs", reflect.TypeOf((*MockAgentSecureServer)(nil).ClientGetConfigs), arg0, arg1)
}

// DogstatsdCaptureTrigger mocks base method.
func (m *MockAgentSecureServer) DogstatsdCaptureTrigger(arg0 context.Context, arg1 *pbgo.CaptureTriggerRequest) (*pbgo.CaptureTriggerResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DogstatsdCaptureTrigger", arg0, arg1)
	ret0, _ := ret[0].(*pbgo.CaptureTriggerResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DogstatsdCaptureTrigger indicates an expected call of DogstatsdCaptureTrigger.
func (mr *MockAgentSecureServerMockRecorder) DogstatsdCaptureTrigger(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DogstatsdCaptureTrigger", reflect.TypeOf((*MockAgentSecureServer)(nil).DogstatsdCaptureTrigger), arg0, arg1)
}

// DogstatsdSetTaggerState mocks base method.
func (m *MockAgentSecureServer) DogstatsdSetTaggerState(arg0 context.Context, arg1 *pbgo.TaggerState) (*pbgo.TaggerStateResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DogstatsdSetTaggerState", arg0, arg1)
	ret0, _ := ret[0].(*pbgo.TaggerStateResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DogstatsdSetTaggerState indicates an expected call of DogstatsdSetTaggerState.
func (mr *MockAgentSecureServerMockRecorder) DogstatsdSetTaggerState(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DogstatsdSetTaggerState", reflect.TypeOf((*MockAgentSecureServer)(nil).DogstatsdSetTaggerState), arg0, arg1)
}

// GetConfigState mocks base method.
func (m *MockAgentSecureServer) GetConfigState(arg0 context.Context, arg1 *emptypb.Empty) (*pbgo.GetStateConfigResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetConfigState", arg0, arg1)
	ret0, _ := ret[0].(*pbgo.GetStateConfigResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetConfigState indicates an expected call of GetConfigState.
func (mr *MockAgentSecureServerMockRecorder) GetConfigState(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetConfigState", reflect.TypeOf((*MockAgentSecureServer)(nil).GetConfigState), arg0, arg1)
}

// TaggerFetchEntity mocks base method.
func (m *MockAgentSecureServer) TaggerFetchEntity(arg0 context.Context, arg1 *pbgo.FetchEntityRequest) (*pbgo.FetchEntityResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TaggerFetchEntity", arg0, arg1)
	ret0, _ := ret[0].(*pbgo.FetchEntityResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// TaggerFetchEntity indicates an expected call of TaggerFetchEntity.
func (mr *MockAgentSecureServerMockRecorder) TaggerFetchEntity(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TaggerFetchEntity", reflect.TypeOf((*MockAgentSecureServer)(nil).TaggerFetchEntity), arg0, arg1)
}

// TaggerStreamEntities mocks base method.
func (m *MockAgentSecureServer) TaggerStreamEntities(arg0 *pbgo.StreamTagsRequest, arg1 pbgo.AgentSecure_TaggerStreamEntitiesServer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TaggerStreamEntities", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// TaggerStreamEntities indicates an expected call of TaggerStreamEntities.
func (mr *MockAgentSecureServerMockRecorder) TaggerStreamEntities(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TaggerStreamEntities", reflect.TypeOf((*MockAgentSecureServer)(nil).TaggerStreamEntities), arg0, arg1)
}

// MockAgentSecure_TaggerStreamEntitiesServer is a mock of AgentSecure_TaggerStreamEntitiesServer interface.
type MockAgentSecure_TaggerStreamEntitiesServer struct {
	ctrl     *gomock.Controller
	recorder *MockAgentSecure_TaggerStreamEntitiesServerMockRecorder
}

// MockAgentSecure_TaggerStreamEntitiesServerMockRecorder is the mock recorder for MockAgentSecure_TaggerStreamEntitiesServer.
type MockAgentSecure_TaggerStreamEntitiesServerMockRecorder struct {
	mock *MockAgentSecure_TaggerStreamEntitiesServer
}

// NewMockAgentSecure_TaggerStreamEntitiesServer creates a new mock instance.
func NewMockAgentSecure_TaggerStreamEntitiesServer(ctrl *gomock.Controller) *MockAgentSecure_TaggerStreamEntitiesServer {
	mock := &MockAgentSecure_TaggerStreamEntitiesServer{ctrl: ctrl}
	mock.recorder = &MockAgentSecure_TaggerStreamEntitiesServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAgentSecure_TaggerStreamEntitiesServer) EXPECT() *MockAgentSecure_TaggerStreamEntitiesServerMockRecorder {
	return m.recorder
}

// Context mocks base method.
func (m *MockAgentSecure_TaggerStreamEntitiesServer) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context.
func (mr *MockAgentSecure_TaggerStreamEntitiesServerMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockAgentSecure_TaggerStreamEntitiesServer)(nil).Context))
}

// RecvMsg mocks base method.
func (m_2 *MockAgentSecure_TaggerStreamEntitiesServer) RecvMsg(m interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "RecvMsg", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecvMsg indicates an expected call of RecvMsg.
func (mr *MockAgentSecure_TaggerStreamEntitiesServerMockRecorder) RecvMsg(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecvMsg", reflect.TypeOf((*MockAgentSecure_TaggerStreamEntitiesServer)(nil).RecvMsg), m)
}

// Send mocks base method.
func (m *MockAgentSecure_TaggerStreamEntitiesServer) Send(arg0 *pbgo.StreamTagsResponse) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Send", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Send indicates an expected call of Send.
func (mr *MockAgentSecure_TaggerStreamEntitiesServerMockRecorder) Send(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockAgentSecure_TaggerStreamEntitiesServer)(nil).Send), arg0)
}

// SendHeader mocks base method.
func (m *MockAgentSecure_TaggerStreamEntitiesServer) SendHeader(arg0 metadata.MD) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendHeader", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendHeader indicates an expected call of SendHeader.
func (mr *MockAgentSecure_TaggerStreamEntitiesServerMockRecorder) SendHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendHeader", reflect.TypeOf((*MockAgentSecure_TaggerStreamEntitiesServer)(nil).SendHeader), arg0)
}

// SendMsg mocks base method.
func (m_2 *MockAgentSecure_TaggerStreamEntitiesServer) SendMsg(m interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "SendMsg", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMsg indicates an expected call of SendMsg.
func (mr *MockAgentSecure_TaggerStreamEntitiesServerMockRecorder) SendMsg(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMsg", reflect.TypeOf((*MockAgentSecure_TaggerStreamEntitiesServer)(nil).SendMsg), m)
}

// SetHeader mocks base method.
func (m *MockAgentSecure_TaggerStreamEntitiesServer) SetHeader(arg0 metadata.MD) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetHeader", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetHeader indicates an expected call of SetHeader.
func (mr *MockAgentSecure_TaggerStreamEntitiesServerMockRecorder) SetHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetHeader", reflect.TypeOf((*MockAgentSecure_TaggerStreamEntitiesServer)(nil).SetHeader), arg0)
}

// SetTrailer mocks base method.
func (m *MockAgentSecure_TaggerStreamEntitiesServer) SetTrailer(arg0 metadata.MD) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetTrailer", arg0)
}

// SetTrailer indicates an expected call of SetTrailer.
func (mr *MockAgentSecure_TaggerStreamEntitiesServerMockRecorder) SetTrailer(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetTrailer", reflect.TypeOf((*MockAgentSecure_TaggerStreamEntitiesServer)(nil).SetTrailer), arg0)
}
