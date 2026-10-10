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

type DataChangeCallback func()

var (
	dataChangeMu        sync.RWMutex
	dataChangeCallbacks []DataChangeCallback
)

// RegisterDataChangeCallback adds a callback that is invoked when data changes.
// Callbacks are run asynchronously in a separate goroutine so they do not block the active database connection.
func RegisterDataChangeCallback(callback DataChangeCallback) {
	dataChangeMu.Lock()
	defer dataChangeMu.Unlock()
	dataChangeCallbacks = append(dataChangeCallbacks, callback)
}

func safeExecuteDataChangeCallback(callback DataChangeCallback) {
	defer func() {
		if r := recover(); r != nil {
			if logger.Log != nil {
				logger.Log.Errorf("panic recovered in data change callback: %v", r)
			}
		}
	}()
	callback()
}

// notifyDataChanged tells observers to refresh data that may have changed.
// Transactional writes call it again after commit so observers can read the committed state.
func notifyDataChanged() {
	dataChangeMu.RLock()
	defer dataChangeMu.RUnlock()
	for _, callback := range dataChangeCallbacks {
		go safeExecuteDataChangeCallback(callback)
	}
}

var registerOnce sync.Once

func init() {
	registerOnce.Do(func() {
		sql.Register("sqlite3_with_hook", &sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				conn.RegisterUpdateHook(func(int, string, string, int64) {
					notifyDataChanged()
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
