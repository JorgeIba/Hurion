package data

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr"
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
		return nil, customerr.NewInitializationError("could not open DB", err)
	}

	if err = conn.Ping(); err != nil {
		return nil, customerr.NewInitializationError("could not ping DB", err)
	}

	if err := applySchema(conn, cfg.SchemaPath); err != nil {
		return nil, customerr.NewInitializationError("could not get the schema", err)
	}

	return &DB{conn: conn, Config: cfg}, nil
}

func applySchema(conn *sql.DB, schemaPath string) error {

	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return customerr.NewInitializationError("could not read schema file", err)
	}

	if _, err = conn.Exec(string(schema)); err != nil {
		return customerr.NewInitializationError("could not apply schema", err)
	}

	fmt.Println("Schema applied successfully")
	return nil
}
