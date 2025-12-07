package database

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockDB struct {
	mock.Mock
}

func (m *MockDB) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	args := m.Called(ctx, txOptions)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(pgx.Tx), args.Error(1)
}

// Остальные методы интерфейса DB нам для теста WithTransaction не нужны,
// но интерфейс требует их реализации. Заглушки:
func (m *MockDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (m *MockDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (m *MockDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row { return nil }
func (m *MockDB) Close()                                                        {}
func (m *MockDB) Ping(ctx context.Context) error                                { return nil }

type MockTx struct {
	mock.Mock
}

func (m *MockTx) Commit(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}
func (m *MockTx) Rollback(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

// Заглушки остальных методов Tx
func (m *MockTx) Begin(ctx context.Context) (pgx.Tx, error) { return nil, nil }
func (m *MockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (m *MockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults { return nil }
func (m *MockTx) LargeObjects() pgx.LargeObjects                               { return pgx.LargeObjects{} }
func (m *MockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (m *MockTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (m *MockTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (m *MockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row { return nil }
func (m *MockTx) Conn() *pgx.Conn                                               { return nil }

// --- Tests ---

func TestWithTransaction_Commit(t *testing.T) {
	mockDB := new(MockDB)
	mockTx := new(MockTx)
	ctx := context.Background()

	// Ожидаем начало транзакции
	mockDB.On("BeginTx", ctx, mock.Anything).Return(mockTx, nil)
	// Ожидаем коммит
	mockTx.On("Commit", ctx).Return(nil)

	err := WithTransaction(ctx, mockDB, func(tx pgx.Tx) error {
		return nil
	})

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

func TestWithTransaction_RollbackOnError(t *testing.T) {
	mockDB := new(MockDB)
	mockTx := new(MockTx)
	ctx := context.Background()
	expectedErr := errors.New("db error")

	mockDB.On("BeginTx", ctx, mock.Anything).Return(mockTx, nil)
	// Ожидаем откат
	mockTx.On("Rollback", ctx).Return(nil)

	err := WithTransaction(ctx, mockDB, func(tx pgx.Tx) error {
		return expectedErr
	})

	assert.ErrorIs(t, err, expectedErr)
	mockDB.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

func TestWithTransaction_RollbackOnPanic(t *testing.T) {
	mockDB := new(MockDB)
	mockTx := new(MockTx)
	ctx := context.Background()

	mockDB.On("BeginTx", ctx, mock.Anything).Return(mockTx, nil)
	// При панике тоже должен быть откат
	mockTx.On("Rollback", ctx).Return(nil)

	assert.Panics(t, func() {
		_ = WithTransaction(ctx, mockDB, func(tx pgx.Tx) error {
			panic("unexpected")
		})
	})

	mockDB.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}
