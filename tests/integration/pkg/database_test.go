//go:build integration

package pkg_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"logistics/pkg/database"
	"logistics/tests/integration/testutil"
)

func TestPostgresDB_Connect(t *testing.T) {
	_ = testutil.RequirePostgres(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	cfg := testutil.PostgresConfig()

	db, err := database.NewPostgresDB(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	testutil.Cleanup(t, func() { db.Close() })

	if err := db.Ping(ctx); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestPostgresDB_HealthCheck(t *testing.T) {
	_ = testutil.RequirePostgres(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	db, err := database.NewPostgresDB(ctx, testutil.PostgresConfig())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	testutil.Cleanup(t, func() { db.Close() })

	if err := db.HealthCheck(ctx); err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}
}

func TestPostgresDB_ExecQuery(t *testing.T) {
	_ = testutil.RequirePostgres(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	db, err := database.NewPostgresDB(ctx, testutil.PostgresConfig())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	testutil.Cleanup(t, func() { db.Close() })

	tableName := "test_exec_" + testutil.RandomString(8)

	// Create table
	_, err = db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS `+tableName+` (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			value INT,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("CREATE TABLE failed: %v", err)
	}
	testutil.Cleanup(t, func() {
		db.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
	})

	// Insert
	_, err = db.Exec(ctx, "INSERT INTO "+tableName+" (name, value) VALUES ($1, $2)", "test", 42)
	if err != nil {
		t.Fatalf("INSERT failed: %v", err)
	}

	// QueryRow
	var name string
	var value int
	err = db.QueryRow(ctx, "SELECT name, value FROM "+tableName+" WHERE name = $1", "test").Scan(&name, &value)
	if err != nil {
		t.Fatalf("SELECT failed: %v", err)
	}
	if name != "test" || value != 42 {
		t.Errorf("got name=%s value=%d, want test, 42", name, value)
	}

	// Query multiple rows
	for i := 0; i < 5; i++ {
		db.Exec(ctx, "INSERT INTO "+tableName+" (name, value) VALUES ($1, $2)", "batch", i)
	}

	rows, err := db.Query(ctx, "SELECT value FROM "+tableName+" WHERE name = $1 ORDER BY value", "batch")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var v int
		rows.Scan(&v)
		if v != count {
			t.Errorf("value = %d, want %d", v, count)
		}
		count++
	}
	if count != 5 {
		t.Errorf("row count = %d, want 5", count)
	}
}

func TestPostgresDB_Transaction_Commit(t *testing.T) {
	_ = testutil.RequirePostgres(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	db, err := database.NewPostgresDB(ctx, testutil.PostgresConfig())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	testutil.Cleanup(t, func() { db.Close() })

	tableName := "test_tx_commit_" + testutil.RandomString(8)
	db.Exec(ctx, "CREATE TABLE "+tableName+" (id SERIAL, value INT)")
	testutil.Cleanup(t, func() {
		db.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
	})

	// Successful transaction
	err = database.WithTransaction(ctx, db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "INSERT INTO "+tableName+" (value) VALUES ($1)", 100)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "INSERT INTO "+tableName+" (value) VALUES ($1)", 200)
		return err
	})
	if err != nil {
		t.Errorf("transaction failed: %v", err)
	}

	// Verify
	var count int
	db.QueryRow(ctx, "SELECT COUNT(*) FROM "+tableName).Scan(&count)
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestPostgresDB_Transaction_Rollback(t *testing.T) {
	_ = testutil.RequirePostgres(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	db, err := database.NewPostgresDB(ctx, testutil.PostgresConfig())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	testutil.Cleanup(t, func() { db.Close() })

	tableName := "test_tx_rollback_" + testutil.RandomString(8)
	db.Exec(ctx, "CREATE TABLE "+tableName+" (id SERIAL, value INT)")
	db.Exec(ctx, "INSERT INTO "+tableName+" (value) VALUES (1)")
	testutil.Cleanup(t, func() {
		db.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
	})

	// Failed transaction
	err = database.WithTransaction(ctx, db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "INSERT INTO "+tableName+" (value) VALUES ($1)", 2)
		if err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	if err == nil {
		t.Error("expected error")
	}

	// Verify rollback - should have only 1 row
	var count int
	db.QueryRow(ctx, "SELECT COUNT(*) FROM "+tableName).Scan(&count)
	if count != 1 {
		t.Errorf("count = %d, want 1 (rollback failed)", count)
	}
}

func TestPostgresDB_Transaction_Panic(t *testing.T) {
	_ = testutil.RequirePostgres(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	db, err := database.NewPostgresDB(ctx, testutil.PostgresConfig())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	testutil.Cleanup(t, func() { db.Close() })

	tableName := "test_tx_panic_" + testutil.RandomString(8)
	db.Exec(ctx, "CREATE TABLE "+tableName+" (id SERIAL, value INT)")
	testutil.Cleanup(t, func() {
		db.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
	})

	// Transaction with panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic to propagate")
		}
	}()

	database.WithTransaction(ctx, db, func(tx pgx.Tx) error {
		tx.Exec(ctx, "INSERT INTO "+tableName+" (value) VALUES ($1)", 1)
		panic("test panic")
	})
}

func TestPostgresDB_WithTransactionResult(t *testing.T) {
	_ = testutil.RequirePostgres(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	db, err := database.NewPostgresDB(ctx, testutil.PostgresConfig())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	testutil.Cleanup(t, func() { db.Close() })

	tableName := "test_tx_result_" + testutil.RandomString(8)
	db.Exec(ctx, "CREATE TABLE "+tableName+" (id SERIAL PRIMARY KEY, value INT)")
	testutil.Cleanup(t, func() {
		db.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
	})

	// Transaction with result
	id, err := database.WithTransactionResult(ctx, db, func(tx pgx.Tx) (int64, error) {
		var id int64
		err := tx.QueryRow(ctx, "INSERT INTO "+tableName+" (value) VALUES ($1) RETURNING id", 42).Scan(&id)
		return id, err
	})
	if err != nil {
		t.Fatalf("transaction failed: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}
}

func TestPostgresDB_Stats(t *testing.T) {
	_ = testutil.RequirePostgres(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	db, err := database.NewPostgresDB(ctx, testutil.PostgresConfig())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	testutil.Cleanup(t, func() { db.Close() })

	// Make some queries
	for i := 0; i < 10; i++ {
		db.QueryRow(ctx, "SELECT 1")
	}

	stats := db.Stats()
	if stats == nil {
		t.Fatal("Stats() returned nil")
	}

	if stats.AcquireCount() < 10 {
		t.Errorf("AcquireCount = %d, expected >= 10", stats.AcquireCount())
	}
}

func TestPostgresDB_Pool(t *testing.T) {
	_ = testutil.RequirePostgres(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	db, err := database.NewPostgresDB(ctx, testutil.PostgresConfig())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	testutil.Cleanup(t, func() { db.Close() })

	pool := db.Pool()
	if pool == nil {
		t.Fatal("Pool() returned nil")
	}

	// Use pool directly
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	defer conn.Release()

	var result int
	err = conn.QueryRow(ctx, "SELECT 42").Scan(&result)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if result != 42 {
		t.Errorf("result = %d, want 42", result)
	}
}

func TestPostgresDB_Reconnect(t *testing.T) {
	_ = testutil.RequirePostgres(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	cfg := testutil.PostgresConfig()
	cfg.MaxOpenConns = 2
	cfg.MaxIdleConns = 1
	cfg.ConnMaxLifetime = 1 * time.Second

	db, err := database.NewPostgresDB(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	testutil.Cleanup(t, func() { db.Close() })

	// Make query
	var result int
	db.QueryRow(ctx, "SELECT 1").Scan(&result)

	// Wait for connection lifetime
	time.Sleep(1500 * time.Millisecond)

	// Should still work (reconnect)
	err = db.QueryRow(ctx, "SELECT 2").Scan(&result)
	if err != nil {
		t.Errorf("query after reconnect failed: %v", err)
	}
	if result != 2 {
		t.Errorf("result = %d, want 2", result)
	}
}
