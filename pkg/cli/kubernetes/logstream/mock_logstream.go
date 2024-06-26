// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/radius-project/radius/pkg/cli/kubernetes/logstream (interfaces: Interface)
//
// Generated by this command:
//
//	mockgen -destination=./mock_logstream.go -package=logstream -self_package github.com/radius-project/radius/pkg/cli/kubernetes/logstream github.com/radius-project/radius/pkg/cli/kubernetes/logstream Interface
//

// Package logstream is a generated GoMock package.
package logstream

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockInterface is a mock of Interface interface.
type MockInterface struct {
	ctrl     *gomock.Controller
	recorder *MockInterfaceMockRecorder
}

// MockInterfaceMockRecorder is the mock recorder for MockInterface.
type MockInterfaceMockRecorder struct {
	mock *MockInterface
}

// NewMockInterface creates a new mock instance.
func NewMockInterface(ctrl *gomock.Controller) *MockInterface {
	mock := &MockInterface{ctrl: ctrl}
	mock.recorder = &MockInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockInterface) EXPECT() *MockInterfaceMockRecorder {
	return m.recorder
}

// Stream mocks base method.
func (m *MockInterface) Stream(arg0 context.Context, arg1 Options) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stream", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Stream indicates an expected call of Stream.
func (mr *MockInterfaceMockRecorder) Stream(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stream", reflect.TypeOf((*MockInterface)(nil).Stream), arg0, arg1)
}
