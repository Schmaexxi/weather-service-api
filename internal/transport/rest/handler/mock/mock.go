// Code generated by MockGen. DO NOT EDIT.
// Source: handlers.go

// Package mock_handler is a generated GoMock package.
package mock_handler

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	model "github.com/katiamach/weather-service-api/internal/model"
)

// MockWeatherService is a mock of WeatherService interface.
type MockWeatherService struct {
	ctrl     *gomock.Controller
	recorder *MockWeatherServiceMockRecorder
}

// MockWeatherServiceMockRecorder is the mock recorder for MockWeatherService.
type MockWeatherServiceMockRecorder struct {
	mock *MockWeatherService
}

// NewMockWeatherService creates a new mock instance.
func NewMockWeatherService(ctrl *gomock.Controller) *MockWeatherService {
	mock := &MockWeatherService{ctrl: ctrl}
	mock.recorder = &MockWeatherServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockWeatherService) EXPECT() *MockWeatherServiceMockRecorder {
	return m.recorder
}

// GetWindInfo mocks base method.
func (m *MockWeatherService) GetWindInfo(ctx context.Context, req *model.WindRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWindInfo", ctx, req)
	ret0, _ := ret[0].(error)
	return ret0
}

// GetWindInfo indicates an expected call of GetWindInfo.
func (mr *MockWeatherServiceMockRecorder) GetWindInfo(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWindInfo", reflect.TypeOf((*MockWeatherService)(nil).GetWindInfo), ctx, req)
}
