// Code generated by MockGen. DO NOT EDIT.
// Source: ./az-client.go
//
// Generated by this command:
//
//	mockgen -source=./az-client.go -destination=./mock/az-client.go .
//

// Package mock_providers is a generated GoMock package.
package mock_providers

import (
	reflect "reflect"

	runtime "github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	armsubscription "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	applications "github.com/microsoftgraph/msgraph-sdk-go/applications"
	gomock "go.uber.org/mock/gomock"
)

// MockazTenantClient is a mock of azTenantClient interface.
type MockazTenantClient struct {
	ctrl     *gomock.Controller
	recorder *MockazTenantClientMockRecorder
}

// MockazTenantClientMockRecorder is the mock recorder for MockazTenantClient.
type MockazTenantClientMockRecorder struct {
	mock *MockazTenantClient
}

// NewMockazTenantClient creates a new mock instance.
func NewMockazTenantClient(ctrl *gomock.Controller) *MockazTenantClient {
	mock := &MockazTenantClient{ctrl: ctrl}
	mock.recorder = &MockazTenantClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockazTenantClient) EXPECT() *MockazTenantClientMockRecorder {
	return m.recorder
}

// NewListPager mocks base method.
func (m *MockazTenantClient) NewListPager(options *armsubscription.TenantsClientListOptions) *runtime.Pager[armsubscription.TenantsClientListResponse] {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewListPager", options)
	ret0, _ := ret[0].(*runtime.Pager[armsubscription.TenantsClientListResponse])
	return ret0
}

// NewListPager indicates an expected call of NewListPager.
func (mr *MockazTenantClientMockRecorder) NewListPager(options any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewListPager", reflect.TypeOf((*MockazTenantClient)(nil).NewListPager), options)
}

// MockGraphClient is a mock of GraphClient interface.
type MockGraphClient struct {
	ctrl     *gomock.Controller
	recorder *MockGraphClientMockRecorder
}

// MockGraphClientMockRecorder is the mock recorder for MockGraphClient.
type MockGraphClientMockRecorder struct {
	mock *MockGraphClient
}

// NewMockGraphClient creates a new mock instance.
func NewMockGraphClient(ctrl *gomock.Controller) *MockGraphClient {
	mock := &MockGraphClient{ctrl: ctrl}
	mock.recorder = &MockGraphClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockGraphClient) EXPECT() *MockGraphClientMockRecorder {
	return m.recorder
}

// Applications mocks base method.
func (m *MockGraphClient) Applications() *applications.ApplicationsRequestBuilder {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Applications")
	ret0, _ := ret[0].(*applications.ApplicationsRequestBuilder)
	return ret0
}

// Applications indicates an expected call of Applications.
func (mr *MockGraphClientMockRecorder) Applications() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Applications", reflect.TypeOf((*MockGraphClient)(nil).Applications))
}
