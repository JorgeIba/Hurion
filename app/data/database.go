package data

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
)

type DBType int

const (
	Prod DBType = iota
	Test
)

type DB struct {
	conn *sql.DB
}

func (db *DB) closeDB() {
	if db.conn != nil {
		db.conn.Close()
	}
}

func newDB(t DBType) (*DB, error) {
	var dbPath string

	if t == Prod {
		dbPath = "./db.sqlite"
	} else if t == Test {
		dbPath = ":memory:"
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("could not open DB: %v", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("could not ping DB: %v", err)
	}

	if err := applySchema(db); err != nil {
		return nil, fmt.Errorf("could not get the schema: %v", err)
	}

	return &DB{conn: db}, nil
}

func applySchema(db *sql.DB) error {
	schemaPath := "./schema.sql"

	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("could not read schema file: %v", err)
	}

	if _, err = db.Exec(string(schema)); err != nil {
		return fmt.Errorf("could not apply schema: %v", err)
	}

	fmt.Println("Schema applied successfully")
	return nil
}
