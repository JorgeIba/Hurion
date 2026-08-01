package data

import (
	"database/sql"
	_ "embed"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/logger"
	sqlite3 "github.com/mattn/go-sqlite3"
)

type UpdateHookCallback func(op int, dbName string, tableName string, rowid int64)

var (
	updateHookMu        sync.RWMutex
	updateHookCallbacks []UpdateHookCallback
)

// RegisterUpdateHookCallback adds a callback that is invoked when writes occur on SQLite.
// Callbacks are run asynchronously in a separate goroutine so they do not block the active database connection.
func RegisterUpdateHookCallback(callback UpdateHookCallback) {
	updateHookMu.Lock()
	defer updateHookMu.Unlock()
	updateHookCallbacks = append(updateHookCallbacks, callback)
}

func safeExecuteCallback(cb UpdateHookCallback, op int, dbName string, tableName string, rowid int64) {
	defer func() {
		if r := recover(); r != nil {
			if logger.Log != nil {
				logger.Log.Errorf("panic recovered in update hook callback: %v", r)
			}
		}
	}()
	cb(op, dbName, tableName, rowid)
}

func triggerUpdateHook(op int, dbName string, tableName string, rowid int64) {
	updateHookMu.RLock()
	defer updateHookMu.RUnlock()
	for _, cb := range updateHookCallbacks {
		go safeExecuteCallback(cb, op, dbName, tableName, rowid)
	}
}

var registerOnce sync.Once

func init() {
	registerOnce.Do(func() {
		sql.Register("sqlite3_with_hook", &sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				conn.RegisterUpdateHook(func(op int, dbName string, tableName string, rowid int64) {
					triggerUpdateHook(op, dbName, tableName, rowid)
				})
				return nil
			},
		})
	})
}

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
		// 0o755: owner=rwx, group=r-x, others=r-x (standard dir permissions)
		if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
			return nil, dataerr.NewInitializationError("could not create DB directory", err)
		}
	}

	params := url.Values{}
	params.Add("_foreign_keys", "on")

	dsn := fmt.Sprintf("%s?%s", cfg.DBPath, params.Encode())
	conn, err := sql.Open("sqlite3_with_hook", dsn)
	if err != nil {
		return nil, dataerr.NewInitializationError("could not open DB", err)
	}

	if err = conn.Ping(); err != nil {
		return nil, dataerr.NewInitializationError("could not ping DB", err)
	}

	if cfg.DBType == Test {
		conn.SetMaxOpenConns(1)
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
