package data

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var embeddedSchema string

type DBTypeEnum int

const (
	Prod DBTypeEnum = iota
	Test
)

type DBConfig struct {
	DBPath string
	DBType DBTypeEnum
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
	if cfg.DBType == Prod {
		if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0755); err != nil {
			return nil, dataerr.NewInitializationError("could not create DB directory", err)
		}
	}

	conn, err := sql.Open("sqlite3", cfg.DBPath)
	if err != nil {
		return nil, dataerr.NewInitializationError("could not open DB", err)
	}

	if err = conn.Ping(); err != nil {
		return nil, dataerr.NewInitializationError("could not ping DB", err)
	}

	if err := applySchema(conn); err != nil {
		return nil, dataerr.NewInitializationError("could not get the schema", err)
	}

	return &DB{conn: conn, Config: cfg}, nil
}

func applySchema(conn *sql.DB) error {
	if _, err := conn.Exec(embeddedSchema); err != nil {
		return dataerr.NewInitializationError("could not apply schema", err)
	}

	fmt.Println("Schema applied successfully")
	return nil
}
