// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/pmdcosta/crawler/internal/worker (interfaces: Backend)

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
	url "net/url"
	reflect "reflect"
)

// MockWorkerBackend is a mock of Backend interface
type MockWorkerBackend struct {
	ctrl     *gomock.Controller
	recorder *MockWorkerBackendMockRecorder
}

// MockWorkerBackendMockRecorder is the mock recorder for MockWorkerBackend
type MockWorkerBackendMockRecorder struct {
	mock *MockWorkerBackend
}

// NewMockWorkerBackend creates a new mock instance
func NewMockWorkerBackend(ctrl *gomock.Controller) *MockWorkerBackend {
	mock := &MockWorkerBackend{ctrl: ctrl}
	mock.recorder = &MockWorkerBackendMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockWorkerBackend) EXPECT() *MockWorkerBackendMockRecorder {
	return m.recorder
}

// Do mocks base method
func (m *MockWorkerBackend) Do(arg0 *url.URL) ([]byte, error) {
	ret := m.ctrl.Call(m, "Do", arg0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Do indicates an expected call of Do
func (mr *MockWorkerBackendMockRecorder) Do(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Do", reflect.TypeOf((*MockWorkerBackend)(nil).Do), arg0)
}
