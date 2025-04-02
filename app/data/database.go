package data

import (
	"fmt"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)


type DBType int

const (
	Prod DBType = iota
	Test	
)

type DB struct {
	conn *sql.DB
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

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("could not ping DB: %v", err)
	}

	return &DB{conn: db}, nil
}