// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	log "github.com/cometbft/cometbft/libs/log"
	mock "github.com/stretchr/testify/mock"
)

// Logger is an autogenerated mock type for the Logger type
type Logger struct {
	mock.Mock
}

// Debug provides a mock function with given fields: msg, keyvals
func (_m *Logger) Debug(msg string, keyvals ...interface{}) {
	var _ca []interface{}
	_ca = append(_ca, msg)
	_ca = append(_ca, keyvals...)
	_m.Called(_ca...)
}

// Error provides a mock function with given fields: msg, keyvals
func (_m *Logger) Error(msg string, keyvals ...interface{}) {
	var _ca []interface{}
	_ca = append(_ca, msg)
	_ca = append(_ca, keyvals...)
	_m.Called(_ca...)
}

// Info provides a mock function with given fields: msg, keyvals
func (_m *Logger) Info(msg string, keyvals ...interface{}) {
	var _ca []interface{}
	_ca = append(_ca, msg)
	_ca = append(_ca, keyvals...)
	_m.Called(_ca...)
}

// With provides a mock function with given fields: keyvals
func (_m *Logger) With(keyvals ...interface{}) log.Logger {
	var _ca []interface{}
	_ca = append(_ca, keyvals...)
	ret := _m.Called(_ca...)

	var r0 log.Logger
	if rf, ok := ret.Get(0).(func(...interface{}) log.Logger); ok {
		r0 = rf(keyvals...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(log.Logger)
		}
	}

	return r0
}

type mockConstructorTestingTNewLogger interface {
	mock.TestingT
	Cleanup(func())
}

// NewLogger creates a new instance of Logger. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewLogger(t mockConstructorTestingTNewLogger) *Logger {
	mock := &Logger{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
