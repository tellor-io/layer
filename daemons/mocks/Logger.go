// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import (
	log "cosmossdk.io/log"
	mock "github.com/stretchr/testify/mock"
)

// Logger is an autogenerated mock type for the Logger type
type Logger struct {
	mock.Mock
}

// Debug provides a mock function with given fields: msg, keyVals
func (_m *Logger) Debug(msg string, keyVals ...interface{}) {
	var _ca []interface{}
	_ca = append(_ca, msg)
	_ca = append(_ca, keyVals...)
	_m.Called(_ca...)
}

// Error provides a mock function with given fields: msg, keyVals
func (_m *Logger) Error(msg string, keyVals ...interface{}) {
	var _ca []interface{}
	_ca = append(_ca, msg)
	_ca = append(_ca, keyVals...)
	_m.Called(_ca...)
}

// Impl provides a mock function with no fields
func (_m *Logger) Impl() interface{} {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Impl")
	}

	var r0 interface{}
	if rf, ok := ret.Get(0).(func() interface{}); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(interface{})
		}
	}

	return r0
}

// Info provides a mock function with given fields: msg, keyVals
func (_m *Logger) Info(msg string, keyVals ...interface{}) {
	var _ca []interface{}
	_ca = append(_ca, msg)
	_ca = append(_ca, keyVals...)
	_m.Called(_ca...)
}

// Warn provides a mock function with given fields: msg, keyVals
func (_m *Logger) Warn(msg string, keyVals ...interface{}) {
	var _ca []interface{}
	_ca = append(_ca, msg)
	_ca = append(_ca, keyVals...)
	_m.Called(_ca...)
}

// With provides a mock function with given fields: keyVals
func (_m *Logger) With(keyVals ...interface{}) log.Logger {
	var _ca []interface{}
	_ca = append(_ca, keyVals...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for With")
	}

	var r0 log.Logger
	if rf, ok := ret.Get(0).(func(...interface{}) log.Logger); ok {
		r0 = rf(keyVals...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(log.Logger)
		}
	}

	return r0
}

// NewLogger creates a new instance of Logger. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewLogger(t interface {
	mock.TestingT
	Cleanup(func())
}) *Logger {
	mock := &Logger{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
