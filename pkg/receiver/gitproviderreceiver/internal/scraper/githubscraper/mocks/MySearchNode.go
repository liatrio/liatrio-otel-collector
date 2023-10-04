// Code generated by mockery v2.34.2. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// MySearchNode is an autogenerated mock type for the MySearchNode type
type MySearchNode struct {
	mock.Mock
	Typename string `json:"__typename"`
	Id       string `json:"id"`
	// The name of the repository.
	Name string `json:"name"`

}

// GetTypename provides a mock function with given fields:
func (_m *MySearchNode) GetTypename() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// implementsGraphQLInterfaceMySearchNode provides a mock function with given fields:
func (_m *MySearchNode) ImplementsGraphQLInterfaceSearchNode() {
}

// NewMySearchNode creates a new instance of MySearchNode. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMySearchNode(t interface {
	mock.TestingT
	Cleanup(func())
}) *MySearchNode {
	mock := &MySearchNode{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
