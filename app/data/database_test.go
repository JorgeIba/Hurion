package data

import (
	"testing"
)

func TestDBConnection(t *testing.T) {
	// Test for production database connection

	prodConfig := &DBConfig{DBPath: "./db.sqlite", SchemaPath: "./schema.sql", DBType: Prod}
	testConfig := &DBConfig{DBPath: ":memory:", SchemaPath: "./schema.sql", DBType: Test}

	t.Run("Prod", func(t *testing.T) {
		db, err := NewDB(prodConfig)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if db.conn == nil {
			t.Fatal("Expected valid connection, got nil")
		}

		// Test for successful ping
		if err := db.conn.Ping(); err != nil {
			t.Fatalf("Expected no error during ping, got %v", err)
		}
	})

	// Test for in-memory database connection
	t.Run("Test", func(t *testing.T) {
		db, err := NewDB(testConfig)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if db.conn == nil {
			t.Fatal("Expected valid connection, got nil")
		}

		// Test for successful ping
		if err := db.conn.Ping(); err != nil {
			t.Fatalf("Expected no error during ping, got %v", err)
		}
	})
}
