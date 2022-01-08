// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/project-radius/radius/pkg/health/handlers (interfaces: HealthHandler)

// Package handlers is a generated GoMock package.
package handlers

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockHealthHandler is a mock of HealthHandler interface.
type MockHealthHandler struct {
	ctrl     *gomock.Controller
	recorder *MockHealthHandlerMockRecorder
}

// MockHealthHandlerMockRecorder is the mock recorder for MockHealthHandler.
type MockHealthHandlerMockRecorder struct {
	mock *MockHealthHandler
}

// NewMockHealthHandler creates a new mock instance.
func NewMockHealthHandler(ctrl *gomock.Controller) *MockHealthHandler {
	mock := &MockHealthHandler{ctrl: ctrl}
	mock.recorder = &MockHealthHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockHealthHandler) EXPECT() *MockHealthHandlerMockRecorder {
	return m.recorder
}

// GetHealthState mocks base method.
func (m *MockHealthHandler) GetHealthState(arg0 context.Context, arg1 HealthRegistration, arg2 Options) HealthState {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHealthState", arg0, arg1, arg2)
	ret0, _ := ret[0].(HealthState)
	return ret0
}

// GetHealthState indicates an expected call of GetHealthState.
func (mr *MockHealthHandlerMockRecorder) GetHealthState(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHealthState", reflect.TypeOf((*MockHealthHandler)(nil).GetHealthState), arg0, arg1, arg2)
}
