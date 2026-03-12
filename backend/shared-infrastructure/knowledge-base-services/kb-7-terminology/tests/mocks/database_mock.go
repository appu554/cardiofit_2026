package mocks

import (
	"database/sql/driver"
	"github.com/stretchr/testify/mock"
)

// MockDB provides a mock implementation of database/sql.DB for testing
type MockDB struct {
	mock.Mock
}

// QueryRow mocks the QueryRow method
func (m *MockDB) QueryRow(query string, args ...interface{}) *MockRows {
	mockArgs := m.Called(query, args)
	return mockArgs.Get(0).(*MockRows)
}

// Query mocks the Query method
func (m *MockDB) Query(query string, args ...interface{}) (*MockRows, error) {
	mockArgs := m.Called(query, args)
	return mockArgs.Get(0).(*MockRows), mockArgs.Error(1)
}

// Exec mocks the Exec method
func (m *MockDB) Exec(query string, args ...interface{}) (MockResult, error) {
	mockArgs := m.Called(query, args)
	return mockArgs.Get(0).(MockResult), mockArgs.Error(1)
}

// Ping mocks the Ping method
func (m *MockDB) Ping() error {
	args := m.Called()
	return args.Error(0)
}

// Close mocks the Close method
func (m *MockDB) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Begin mocks the Begin method
func (m *MockDB) Begin() (*MockTx, error) {
	args := m.Called()
	return args.Get(0).(*MockTx), args.Error(1)
}

// MockRows provides a mock implementation of sql.Rows
type MockRows struct {
	mock.Mock
}

// NewMockRows creates a new MockRows instance
func NewMockRows() *MockRows {
	return &MockRows{}
}

// Scan mocks the Scan method
func (m *MockRows) Scan(dest ...interface{}) error {
	args := make([]interface{}, len(dest))
	for i, d := range dest {
		args[i] = d
	}
	mockArgs := m.Called(args...)
	return mockArgs.Error(0)
}

// Next mocks the Next method
func (m *MockRows) Next() bool {
	args := m.Called()
	return args.Bool(0)
}

// Close mocks the Close method
func (m *MockRows) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Err mocks the Err method
func (m *MockRows) Err() error {
	args := m.Called()
	return args.Error(0)
}

// MockResult provides a mock implementation of sql.Result
type MockResult struct {
	mock.Mock
}

// LastInsertId mocks the LastInsertId method
func (m MockResult) LastInsertId() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// RowsAffected mocks the RowsAffected method
func (m MockResult) RowsAffected() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// MockTx provides a mock implementation of sql.Tx
type MockTx struct {
	mock.Mock
}

// Commit mocks the Commit method
func (m *MockTx) Commit() error {
	args := m.Called()
	return args.Error(0)
}

// Rollback mocks the Rollback method
func (m *MockTx) Rollback() error {
	args := m.Called()
	return args.Error(0)
}

// QueryRow mocks the QueryRow method
func (m *MockTx) QueryRow(query string, args ...interface{}) *MockRows {
	mockArgs := m.Called(query, args)
	return mockArgs.Get(0).(*MockRows)
}

// Query mocks the Query method
func (m *MockTx) Query(query string, args ...interface{}) (*MockRows, error) {
	mockArgs := m.Called(query, args)
	return mockArgs.Get(0).(*MockRows), mockArgs.Error(1)
}

// Exec mocks the Exec method
func (m *MockTx) Exec(query string, args ...interface{}) (MockResult, error) {
	mockArgs := m.Called(query, args)
	return mockArgs.Get(0).(MockResult), mockArgs.Error(1)
}

// MockDriver provides a mock implementation of database/sql/driver.Driver
type MockDriver struct {
	mock.Mock
}

// Open mocks the Open method
func (m *MockDriver) Open(name string) (driver.Conn, error) {
	args := m.Called(name)
	return args.Get(0).(driver.Conn), args.Error(1)
}

// MockConn provides a mock implementation of database/sql/driver.Conn
type MockConn struct {
	mock.Mock
}

// Prepare mocks the Prepare method
func (m *MockConn) Prepare(query string) (driver.Stmt, error) {
	args := m.Called(query)
	return args.Get(0).(driver.Stmt), args.Error(1)
}

// Close mocks the Close method
func (m *MockConn) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Begin mocks the Begin method
func (m *MockConn) Begin() (driver.Tx, error) {
	args := m.Called()
	return args.Get(0).(driver.Tx), args.Error(1)
}