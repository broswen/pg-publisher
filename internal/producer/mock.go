package producer

import (
	"github.com/stretchr/testify/mock"
)

type MockProducer struct {
	mock.Mock
}

func (m *MockProducer) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockProducer) Submit(row map[string]interface{}) error {
	args := m.Called(row)
	return args.Error(0)
}
