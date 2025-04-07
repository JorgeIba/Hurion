package data

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type DBTypeEnum int

const (
	Prod DBTypeEnum = iota
	Test
)

type DBConfig struct {
	DBPath     string
	SchemaPath string
	DBType     DBTypeEnum
}

type DB struct {
	conn   *sql.DB
	Config *DBConfig
}

func (db *DB) CloseDB() {
	if db.conn != nil {
		db.conn.Close()
	}
}

func NewDB(cfg *DBConfig) (*DB, error) {
	conn, err := sql.Open("sqlite3", cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("could not open DB: %v", err)
	}

	if err = conn.Ping(); err != nil {
		return nil, fmt.Errorf("could not ping DB: %v", err)
	}

	if err := applySchema(conn, cfg.SchemaPath); err != nil {
		return nil, fmt.Errorf("could not get the schema: %v", err)
	}

	return &DB{conn: conn, Config: cfg}, nil
}

func applySchema(conn *sql.DB, schemaPath string) error {

	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("could not read schema file: %v", err)
	}

	if _, err = conn.Exec(string(schema)); err != nil {
		return fmt.Errorf("could not apply schema: %v", err)
	}

	fmt.Println("Schema applied successfully")
	return nil
}
