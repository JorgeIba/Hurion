package data

import (
	"os"
	"testing"
)

func TestDBConnection(t *testing.T) {

	prodConfig := &DBConfig{DBPath: "./db.sqlite", SchemaPath: "./schema.sql", DBType: Prod}
	testConfig := &DBConfig{DBPath: ":memory:", SchemaPath: "./schema.sql", DBType: Test}

	// Test for production database connection
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
		if err := os.Remove(prodConfig.DBPath); err != nil {
			t.Fatal("Someting wrong, db test file could not be deleted", err)
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
