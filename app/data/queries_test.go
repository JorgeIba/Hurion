package data

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupDB returns a brand new, isolated in-memory database for testing.
// Every call creates a fresh instance with the schema applied.
func setupDB(t *testing.T) *DB {
	t.Helper()
	cfg := &DBConfig{DBPath: ":memory:", DBType: Test}
	db, err := NewDB(cfg)
	require.NoError(t, err, "failed to create test db")
	return db
}

func ptrStr(value string) *string {
	return &value
}

var batchLookupQueryCount atomic.Int64
var batchWriteQueryCount atomic.Int64

type queryCountingDriver struct {
	driver.Driver
	queries *atomic.Int64
}

func (counting queryCountingDriver) Open(name string) (driver.Conn, error) {
	conn, err := counting.Driver.Open(name)
	if err != nil {
		return nil, err
	}
	return queryCountingConn{Conn: conn, queries: counting.queries}, nil
}

type queryCountingConn struct {
	driver.Conn
	queries *atomic.Int64
}

func (counting queryCountingConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	batchWriteQueryCount.Add(1)
	execer, ok := counting.Conn.(driver.ExecerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	return execer.ExecContext(ctx, query, args)
}

func (counting queryCountingConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	counting.countRead(query)
	queryer, ok := counting.Conn.(driver.QueryerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	return queryer.QueryContext(ctx, query, args)
}

func (counting queryCountingConn) countRead(query string) {
	countBatchLookupRead(query, counting.queries)
}

func countBatchLookupRead(query string, queries *atomic.Int64) {
	query = strings.ToUpper(strings.TrimSpace(query))
	if strings.HasPrefix(query, "SELECT") || strings.HasPrefix(query, "WITH") {
		queries.Add(1)
	}
}

func (counting queryCountingConn) Prepare(query string) (driver.Stmt, error) {
	return counting.Conn.Prepare(query)
}

func init() {
	sql.Register("sqlite3_batch_lookup_count", queryCountingDriver{Driver: &sqlite3.SQLiteDriver{}, queries: &batchLookupQueryCount})
}

func TestBatchProblemLookupsUseOneSQLReadForOneAndOneHundredTargets(t *testing.T) {
	conn, err := sql.Open("sqlite3_batch_lookup_count", ":memory:")
	require.NoError(t, err)
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	require.NoError(t, applySchema(conn))
	db := &DB{conn: conn}

	codes := make([]string, 100)
	problemIDs := make([]int64, 100)
	for index := range 100 {
		codes[index] = fmt.Sprintf("class-%d", index)
		classID, err := db.CreateClass(Class{ClassCode: codes[index], Name: codes[index]})
		require.NoError(t, err)
		problemIDs[index], err = db.AddProblem(Problem{Name: fmt.Sprintf("problem-%d", index), URL: "https://example.com", ClassID: classID})
		require.NoError(t, err)
	}

	for _, count := range []int{1, 100} {
		batchLookupQueryCount.Store(0)
		classes, err := db.GetClassesByClassCodes(codes[:count])
		require.NoError(t, err)
		assert.Len(t, classes, count)
		assert.EqualValues(t, 1, batchLookupQueryCount.Load())

		batchLookupQueryCount.Store(0)
		problems, err := db.GetProblemsByIDs(problemIDs[:count])
		require.NoError(t, err)
		assert.Len(t, problems, count)
		assert.EqualValues(t, 1, batchLookupQueryCount.Load())
	}
}

func TestEnsureUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		user       User
		assertFunc func(t *testing.T, db *DB, err error)
	}{
		{
			name: "should insert new user",
			user: User{UserID: "123", DisplayName: "John Doe"},
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.NoError(t, err)
				var name string
				err = db.conn.QueryRow("SELECT display_name FROM users WHERE user_id = '123'").Scan(&name)
				assert.NoError(t, err)
				assert.Equal(t, "John Doe", name)
			},
		},
		{
			name: "should preserve name when new name is empty",
			setup: func(t *testing.T, db *DB) {
				err := db.EnsureUser(User{UserID: "123", DisplayName: "Original Name"})
				require.NoError(t, err)
			},
			user: User{UserID: "123", DisplayName: ""},
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.NoError(t, err)
				var name string
				err = db.conn.QueryRow("SELECT display_name FROM users WHERE user_id = '123'").Scan(&name)
				assert.NoError(t, err)
				assert.Equal(t, "Original Name", name, "Logic check: empty name should not overwrite")
			},
		},
		{
			name: "should update name if not empty",
			setup: func(t *testing.T, db *DB) {
				err := db.EnsureUser(User{UserID: "123", DisplayName: "Old Name"})
				require.NoError(t, err)
			},
			user: User{UserID: "123", DisplayName: "New Name"},
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.NoError(t, err)
				var name string
				err = db.conn.QueryRow("SELECT display_name FROM users WHERE user_id = '123'").Scan(&name)
				assert.NoError(t, err)
				assert.Equal(t, "New Name", name)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			user: User{UserID: "123", DisplayName: "John Doe"},
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			err := db.EnsureUser(tt.user)
			tt.assertFunc(t, db, err)
		})
	}
}

func TestAddProblem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		problem    Problem
		assertFunc func(t *testing.T, db *DB, id int64, err error)
	}{
		{
			name: "should successfully add problem",
			setup: func(t *testing.T, db *DB) {
				_, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
			},
			problem: Problem{Name: "p1", URL: "u1", ClassID: 1, Priority: ptrStr("recommended")},
			assertFunc: func(t *testing.T, db *DB, id int64, err error) {
				assert.NoError(t, err)
				assert.Greater(t, id, int64(0))
				problem, err := db.GetProblemByID(id)
				require.NoError(t, err)
				assert.Equal(t, "p1", problem.Name)
				if assert.NotNil(t, problem.Priority) {
					assert.Equal(t, "recommended", *problem.Priority)
				}
				assert.False(t, problem.CreatedAt.IsZero())
			},
		},
		{
			name: "stores priority values without database-level validation",
			setup: func(t *testing.T, db *DB) {
				_, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
			},
			problem: Problem{Name: "p1", URL: "u1", ClassID: 1, Priority: ptrStr("top_tier")},
			assertFunc: func(t *testing.T, db *DB, id int64, err error) {
				assert.NoError(t, err)
				problem, err := db.GetProblemByID(id)
				require.NoError(t, err)
				if assert.NotNil(t, problem.Priority) {
					assert.Equal(t, "top_tier", *problem.Priority)
				}
			},
		},
		{
			name: "should allow another problem with the same name in class",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				_, err = db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})
				require.NoError(t, err)
			},
			problem: Problem{Name: "p1", URL: "u2", ClassID: 1},
			assertFunc: func(t *testing.T, db *DB, id int64, err error) {
				assert.NoError(t, err)
				assert.Greater(t, id, int64(0))
				var count int
				err = db.conn.QueryRow("SELECT COUNT(*) FROM problems WHERE name = ? AND class_id = ?", "p1", 1).Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 2, count)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			problem: Problem{Name: "p1", URL: "u1", ClassID: 1},
			assertFunc: func(t *testing.T, db *DB, id int64, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			id, err := db.AddProblem(tt.problem)
			tt.assertFunc(t, db, id, err)
		})
	}
}

func TestRemoveProblemByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		problemID  int64
		assertFunc func(t *testing.T, db *DB, affected int64, err error)
	}{
		{
			name: "should remove problem and its dependencies",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				pID, err := db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})
				require.NoError(t, err)
				err = db.EnsureUser(User{UserID: "u1", DisplayName: "n1"})
				require.NoError(t, err)
				err = db.AddAC("u1", pID)
				require.NoError(t, err)
			},
			problemID: 1,
			assertFunc: func(t *testing.T, db *DB, affected int64, err error) {
				assert.NoError(t, err)
				assert.Equal(t, int64(1), affected)
				var count int
				err = db.conn.QueryRow("SELECT COUNT(*) FROM problems WHERE id = 1").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count)

				err = db.conn.QueryRow("SELECT COUNT(*) FROM solved_problems WHERE problem_id = 1").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			problemID: 1,
			assertFunc: func(t *testing.T, db *DB, affected int64, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			affected, err := db.RemoveProblemByID(tt.problemID)
			tt.assertFunc(t, db, affected, err)
		})
	}
}

func TestAddAC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		userID     string
		problemID  int64
		assertFunc func(t *testing.T, db *DB, err error)
	}{
		{
			name: "should successfully add AC",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				_, err = db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})
				require.NoError(t, err)
				err = db.EnsureUser(User{UserID: "u1", DisplayName: "n1"})
				require.NoError(t, err)
			},
			userID:    "u1",
			problemID: 1,
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.NoError(t, err)
				var count int
				err = db.conn.QueryRow("SELECT COUNT(*) FROM solved_problems WHERE user_id = 'u1' AND problem_id = 1").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 1, count)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			userID:    "u1",
			problemID: 1,
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			err := db.AddAC(tt.userID, tt.problemID)
			tt.assertFunc(t, db, err)
		})
	}
}

func TestRemoveAC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		userID     string
		problemID  int64
		assertFunc func(t *testing.T, db *DB, err error)
	}{
		{
			name: "should remove AC",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				pID, err := db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})
				require.NoError(t, err)
				err = db.EnsureUser(User{UserID: "u1", DisplayName: "n1"})
				require.NoError(t, err)
				err = db.AddAC("u1", pID)
				require.NoError(t, err)
			},
			userID:    "u1",
			problemID: 1,
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.NoError(t, err)
				var count int
				err = db.conn.QueryRow("SELECT COUNT(*) FROM solved_problems WHERE user_id = 'u1' AND problem_id = 1").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			userID:    "u1",
			problemID: 1,
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			err := db.RemoveAC(tt.userID, tt.problemID)
			tt.assertFunc(t, db, err)
		})
	}
}

func TestGetProblemsByClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		classID    int64
		assertFunc func(t *testing.T, db *DB, problems []Problem, err error)
	}{
		{
			name: "should get problems for class",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				_, err = db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})
				require.NoError(t, err)
				_, err = db.AddProblem(Problem{Name: "p2", URL: "u2", ClassID: cID})
				require.NoError(t, err)
			},
			classID: 1,
			assertFunc: func(t *testing.T, db *DB, problems []Problem, err error) {
				assert.NoError(t, err)
				assert.Len(t, problems, 2)
				assert.Equal(t, "p1", problems[0].Name)
				assert.Equal(t, "p2", problems[1].Name)
				assert.Nil(t, problems[0].Priority)
				assert.False(t, problems[0].CreatedAt.IsZero())
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			classID: 1,
			assertFunc: func(t *testing.T, db *DB, problems []Problem, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			problems, err := db.GetProblemsByClass(tt.classID)
			tt.assertFunc(t, db, problems, err)
		})
	}
}

func TestGetSolvedProblemsByUserAndClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		userID     string
		classID    int64
		assertFunc func(t *testing.T, db *DB, solved []SolvedProblem, err error)
	}{
		{
			name: "should return solved problems",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				pID, err := db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})
				require.NoError(t, err)
				err = db.EnsureUser(User{UserID: "u1", DisplayName: "n1"})
				require.NoError(t, err)
				err = db.AddAC("u1", pID)
				require.NoError(t, err)
			},
			userID:  "u1",
			classID: 1,
			assertFunc: func(t *testing.T, db *DB, solved []SolvedProblem, err error) {
				assert.NoError(t, err)
				assert.Len(t, solved, 1)
				assert.Equal(t, int64(1), solved[0].ProblemID)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			userID:  "u1",
			classID: 1,
			assertFunc: func(t *testing.T, db *DB, solved []SolvedProblem, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			solved, err := db.GetSolvedProblemsByUserAndClass(tt.userID, tt.classID)
			tt.assertFunc(t, db, solved, err)
		})
	}
}

func TestGetAllSolvedProblemsByClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		classID    int64
		assertFunc func(t *testing.T, db *DB, solved map[User][]SolvedProblem, err error)
	}{
		{
			name: "should group solved problems by user",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				pID1, err := db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})
				require.NoError(t, err)
				pID2, err := db.AddProblem(Problem{Name: "p2", URL: "u2", ClassID: cID})
				require.NoError(t, err)
				err = db.EnsureUser(User{UserID: "u1", DisplayName: "User 1"})
				require.NoError(t, err)
				err = db.EnsureUser(User{UserID: "u2", DisplayName: "User 2"})
				require.NoError(t, err)
				err = db.AddAC("u1", pID1)
				require.NoError(t, err)
				err = db.AddAC("u1", pID2)
				require.NoError(t, err)
				err = db.AddAC("u2", pID1)
				require.NoError(t, err)
			},
			classID: 1,
			assertFunc: func(t *testing.T, db *DB, solved map[User][]SolvedProblem, err error) {
				assert.NoError(t, err)
				assert.Len(t, solved, 2)
				user1 := User{UserID: "u1", DisplayName: "User 1"}
				assert.Len(t, solved[user1], 2)
				user2 := User{UserID: "u2", DisplayName: "User 2"}
				assert.Len(t, solved[user2], 1)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			classID: 1,
			assertFunc: func(t *testing.T, db *DB, solved map[User][]SolvedProblem, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			solved, err := db.GetAllSolvedProblemsByClass(tt.classID)
			tt.assertFunc(t, db, solved, err)
		})
	}
}

func TestClearClassContents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		classID    int64
		assertFunc func(t *testing.T, db *DB, err error)
	}{
		{
			name: "should wipe problems and cascaded score data while keeping class settings",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				err = db.AddClassroom(Classroom{RoomID: "ch-1", ClassCode: "c1"})
				require.NoError(t, err)
				err = db.AddClassAccessRole(ClassAccessRole{ClassCode: "c1", RoleID: "r1"})
				require.NoError(t, err)
				pID, err := db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})
				require.NoError(t, err)
				err = db.EnsureUser(User{UserID: "u1", DisplayName: "n1"})
				require.NoError(t, err)
				err = db.AddAC("u1", pID)
				require.NoError(t, err)
				err = db.AddPendingConfirmation(PendingConfirmation{MessageID: "m1", UserID: "u1", ProblemID: pID})
				require.NoError(t, err)
			},
			classID: 1,
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.NoError(t, err)
				var count int
				err = db.conn.QueryRow("SELECT COUNT(*) FROM classes WHERE id = 1").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 1, count)
				err = db.conn.QueryRow("SELECT COUNT(*) FROM problems WHERE class_id = 1").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count)
				err = db.conn.QueryRow("SELECT COUNT(*) FROM solved_problems").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count)
				err = db.conn.QueryRow("SELECT COUNT(*) FROM pending_confirmations").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count)
				err = db.conn.QueryRow("SELECT COUNT(*) FROM classrooms WHERE class_code = 'c1'").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 1, count)
				err = db.conn.QueryRow("SELECT COUNT(*) FROM class_access_roles WHERE class_code = 'c1'").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 1, count)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			classID: 1,
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			err := db.ClearClassContents(tt.classID)
			tt.assertFunc(t, db, err)
		})
	}
}

func TestDeleteClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		classID    int64
		assertFunc func(t *testing.T, db *DB, err error)
	}{
		{
			name: "should wipe class and all its associations",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				err = db.AddClassroom(Classroom{RoomID: "ch-1", ClassCode: "c1"})
				require.NoError(t, err)
				err = db.AddClassAccessRole(ClassAccessRole{ClassCode: "c1", RoleID: "r1"})
				require.NoError(t, err)
				pID, err := db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})
				require.NoError(t, err)
				err = db.EnsureUser(User{UserID: "u1", DisplayName: "n1"})
				require.NoError(t, err)
				err = db.AddAC("u1", pID)
				require.NoError(t, err)
			},
			classID: 1,
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.NoError(t, err)
				var count int
				err = db.conn.QueryRow("SELECT COUNT(*) FROM classes WHERE id = 1").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count)
				err = db.conn.QueryRow("SELECT COUNT(*) FROM problems WHERE class_id = 1").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count)
				err = db.conn.QueryRow("SELECT COUNT(*) FROM classrooms WHERE class_code = 'c1'").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count)
				err = db.conn.QueryRow("SELECT COUNT(*) FROM class_access_roles WHERE class_code = 'c1'").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			classID: 1,
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			err := db.DeleteClass(tt.classID)
			tt.assertFunc(t, db, err)
		})
	}
}

func TestCreateClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		class      Class
		assertFunc func(t *testing.T, db *DB, id int64, err error)
	}{
		{
			name:  "should successfully create class",
			class: Class{ClassCode: "basicos", Name: "Basicos 2025"},
			assertFunc: func(t *testing.T, db *DB, id int64, err error) {
				assert.NoError(t, err)
				assert.Greater(t, id, int64(0))
				var code string
				err = db.conn.QueryRow("SELECT class_code FROM classes WHERE id = ?", id).Scan(&code)
				assert.NoError(t, err)
				assert.Equal(t, "basicos", code)
			},
		},
		{
			name: "should return error if class code already exists",
			setup: func(t *testing.T, db *DB) {
				_, err := db.CreateClass(Class{ClassCode: "basicos", Name: "Basicos 2025"})
				require.NoError(t, err)
			},
			class: Class{ClassCode: "basicos", Name: "Duplicate"},
			assertFunc: func(t *testing.T, db *DB, id int64, err error) {
				assert.Error(t, err)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			class: Class{ClassCode: "any", Name: "any"},
			assertFunc: func(t *testing.T, db *DB, id int64, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			id, err := db.CreateClass(tt.class)
			tt.assertFunc(t, db, id, err)
		})
	}
}

func TestAddClassroom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		classroom  Classroom
		assertFunc func(t *testing.T, db *DB, err error)
	}{
		{
			name: "should upsert classroom to latest class",
			setup: func(t *testing.T, db *DB) {
				_, err := db.CreateClass(Class{ClassCode: "c1", Name: "Class 1"})
				require.NoError(t, err)
				_, err = db.CreateClass(Class{ClassCode: "c2", Name: "Class 2"})
				require.NoError(t, err)
				err = db.AddClassroom(Classroom{RoomID: "ch-1", ClassCode: "c1"})
				require.NoError(t, err)
			},
			classroom: Classroom{RoomID: "ch-1", ClassCode: "c2"},
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.NoError(t, err)
				classCode, getErr := db.GetClassCodeByRoomID("ch-1")
				assert.NoError(t, getErr)
				assert.Equal(t, "c2", classCode)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			classroom: Classroom{RoomID: "ch-1", ClassCode: "c1"},
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			err := db.AddClassroom(tt.classroom)
			tt.assertFunc(t, db, err)
		})
	}
}

func TestGetClassCodeByRoomID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		roomID     string
		assertFunc func(t *testing.T, db *DB, classCode string, err error)
	}{
		{
			name: "should get class code for mapped room",
			setup: func(t *testing.T, db *DB) {
				_, err := db.CreateClass(Class{ClassCode: "c1", Name: "Class 1"})
				require.NoError(t, err)
				err = db.AddClassroom(Classroom{RoomID: "ch-1", ClassCode: "c1"})
				require.NoError(t, err)
			},
			roomID: "ch-1",
			assertFunc: func(t *testing.T, db *DB, classCode string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "c1", classCode)
			},
		},
		{
			name:   "should return error for unknown room",
			roomID: "missing",
			assertFunc: func(t *testing.T, db *DB, classCode string, err error) {
				assert.Error(t, err)
				assert.Equal(t, "", classCode)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			roomID: "ch-1",
			assertFunc: func(t *testing.T, db *DB, classCode string, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			classCode, err := db.GetClassCodeByRoomID(tt.roomID)
			tt.assertFunc(t, db, classCode, err)
		})
	}
}

func TestGetClassroomsByClassCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		classCode  string
		assertFunc func(t *testing.T, db *DB, classrooms []Classroom, err error)
	}{
		{
			name: "should return classrooms sorted by room_id",
			setup: func(t *testing.T, db *DB) {
				_, err := db.CreateClass(Class{ClassCode: "c1", Name: "Class 1"})
				require.NoError(t, err)
				err = db.AddClassroom(Classroom{RoomID: "ch-2", ClassCode: "c1"})
				require.NoError(t, err)
				err = db.AddClassroom(Classroom{RoomID: "ch-1", ClassCode: "c1"})
				require.NoError(t, err)
			},
			classCode: "c1",
			assertFunc: func(t *testing.T, db *DB, classrooms []Classroom, err error) {
				assert.NoError(t, err)
				assert.Len(t, classrooms, 2)
				assert.Equal(t, "ch-1", classrooms[0].RoomID)
				assert.Equal(t, "ch-2", classrooms[1].RoomID)
			},
		},
		{
			name:      "should return empty list for class without classrooms",
			classCode: "c1",
			assertFunc: func(t *testing.T, db *DB, classrooms []Classroom, err error) {
				assert.NoError(t, err)
				assert.Len(t, classrooms, 0)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			classCode: "c1",
			assertFunc: func(t *testing.T, db *DB, classrooms []Classroom, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			classrooms, err := db.GetClassroomsByClassCode(tt.classCode)
			tt.assertFunc(t, db, classrooms, err)
		})
	}
}

func TestRemoveClassroomByRoomID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		roomID     string
		assertFunc func(t *testing.T, db *DB, err error)
	}{
		{
			name: "should remove classroom mapping",
			setup: func(t *testing.T, db *DB) {
				_, err := db.CreateClass(Class{ClassCode: "c1", Name: "Class 1"})
				require.NoError(t, err)
				err = db.AddClassroom(Classroom{RoomID: "ch-1", ClassCode: "c1"})
				require.NoError(t, err)
			},
			roomID: "ch-1",
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.NoError(t, err)
				_, getErr := db.GetClassCodeByRoomID("ch-1")
				assert.Error(t, getErr)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			roomID: "ch-1",
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			err := db.RemoveClassroomByRoomID(tt.roomID)
			tt.assertFunc(t, db, err)
		})
	}
}

func TestAddClassAccessRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		classRole  ClassAccessRole
		assertFunc func(t *testing.T, db *DB, err error)
	}{
		{
			name: "should ignore duplicate role and keep one row",
			setup: func(t *testing.T, db *DB) {
				_, err := db.CreateClass(Class{ClassCode: "c1", Name: "Class 1"})
				require.NoError(t, err)
				err = db.AddClassAccessRole(ClassAccessRole{ClassCode: "c1", RoleID: "r1"})
				require.NoError(t, err)
			},
			classRole: ClassAccessRole{ClassCode: "c1", RoleID: "r1"},
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.NoError(t, err)
				roles, getErr := db.GetClassAccessRolesByClassCode("c1")
				assert.NoError(t, getErr)
				assert.Len(t, roles, 1)
				assert.Equal(t, "r1", roles[0].RoleID)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			classRole: ClassAccessRole{ClassCode: "c1", RoleID: "r1"},
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			err := db.AddClassAccessRole(tt.classRole)
			tt.assertFunc(t, db, err)
		})
	}
}

func TestGetClassAccessRolesByClassCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		classCode  string
		assertFunc func(t *testing.T, db *DB, roles []ClassAccessRole, err error)
	}{
		{
			name: "should return roles sorted by role_id",
			setup: func(t *testing.T, db *DB) {
				_, err := db.CreateClass(Class{ClassCode: "c1", Name: "Class 1"})
				require.NoError(t, err)
				err = db.AddClassAccessRole(ClassAccessRole{ClassCode: "c1", RoleID: "r2"})
				require.NoError(t, err)
				err = db.AddClassAccessRole(ClassAccessRole{ClassCode: "c1", RoleID: "r1"})
				require.NoError(t, err)
			},
			classCode: "c1",
			assertFunc: func(t *testing.T, db *DB, roles []ClassAccessRole, err error) {
				assert.NoError(t, err)
				assert.Len(t, roles, 2)
				assert.Equal(t, "r1", roles[0].RoleID)
				assert.Equal(t, "r2", roles[1].RoleID)
			},
		},
		{
			name:      "should return empty list for class without roles",
			classCode: "c1",
			assertFunc: func(t *testing.T, db *DB, roles []ClassAccessRole, err error) {
				assert.NoError(t, err)
				assert.Len(t, roles, 0)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			classCode: "c1",
			assertFunc: func(t *testing.T, db *DB, roles []ClassAccessRole, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			roles, err := db.GetClassAccessRolesByClassCode(tt.classCode)
			tt.assertFunc(t, db, roles, err)
		})
	}
}

func TestHasClassAccessRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		classCode  string
		roleID     string
		assertFunc func(t *testing.T, exists bool, err error)
	}{
		{
			name: "should return true when role is linked to class",
			setup: func(t *testing.T, db *DB) {
				_, err := db.CreateClass(Class{ClassCode: "c1", Name: "Class 1"})
				require.NoError(t, err)
				err = db.AddClassAccessRole(ClassAccessRole{ClassCode: "c1", RoleID: "r1"})
				require.NoError(t, err)
			},
			classCode: "c1",
			roleID:    "r1",
			assertFunc: func(t *testing.T, exists bool, err error) {
				assert.NoError(t, err)
				assert.True(t, exists)
			},
		},
		{
			name: "should return false when role is not linked to class",
			setup: func(t *testing.T, db *DB) {
				_, err := db.CreateClass(Class{ClassCode: "c1", Name: "Class 1"})
				require.NoError(t, err)
				err = db.AddClassAccessRole(ClassAccessRole{ClassCode: "c1", RoleID: "r1"})
				require.NoError(t, err)
			},
			classCode: "c1",
			roleID:    "missing",
			assertFunc: func(t *testing.T, exists bool, err error) {
				assert.NoError(t, err)
				assert.False(t, exists)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			classCode: "c1",
			roleID:    "r1",
			assertFunc: func(t *testing.T, exists bool, err error) {
				assert.Error(t, err)
				assert.False(t, exists)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			exists, err := db.HasClassAccessRole(tt.classCode, tt.roleID)
			tt.assertFunc(t, exists, err)
		})
	}
}

func TestRemoveClassAccessRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		classRole  ClassAccessRole
		assertFunc func(t *testing.T, db *DB, err error)
	}{
		{
			name: "should remove class access role mapping",
			setup: func(t *testing.T, db *DB) {
				_, err := db.CreateClass(Class{ClassCode: "c1", Name: "Class 1"})
				require.NoError(t, err)
				err = db.AddClassAccessRole(ClassAccessRole{ClassCode: "c1", RoleID: "r1"})
				require.NoError(t, err)
			},
			classRole: ClassAccessRole{ClassCode: "c1", RoleID: "r1"},
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.NoError(t, err)
				roles, getErr := db.GetClassAccessRolesByClassCode("c1")
				assert.NoError(t, getErr)
				assert.Len(t, roles, 0)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			classRole: ClassAccessRole{ClassCode: "c1", RoleID: "r1"},
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			err := db.RemoveClassAccessRole(tt.classRole)
			tt.assertFunc(t, db, err)
		})
	}
}

func TestGetUserByUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		userID     string
		assertFunc func(t *testing.T, db *DB, user User, err error)
	}{
		{
			name: "should successfully get user",
			setup: func(t *testing.T, db *DB) {
				err := db.EnsureUser(User{UserID: "u1", DisplayName: "n1"})
				require.NoError(t, err)
			},
			userID: "u1",
			assertFunc: func(t *testing.T, db *DB, user User, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "u1", user.UserID)
				assert.Equal(t, "n1", user.DisplayName)
			},
		},
		{
			name:   "should return error if user not found",
			userID: "missing",
			assertFunc: func(t *testing.T, db *DB, user User, err error) {
				assert.Error(t, err)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			userID: "u1",
			assertFunc: func(t *testing.T, db *DB, user User, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			user, err := db.GetUserByUserID(tt.userID)
			tt.assertFunc(t, db, user, err)
		})
	}
}

func TestGetClassByClassCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		classCode  string
		assertFunc func(t *testing.T, db *DB, class Class, err error)
	}{
		{
			name: "should successfully get class",
			setup: func(t *testing.T, db *DB) {
				_, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
			},
			classCode: "c1",
			assertFunc: func(t *testing.T, db *DB, class Class, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "c1", class.ClassCode)
			},
		},
		{
			name:      "should return error if class not found",
			classCode: "missing",
			assertFunc: func(t *testing.T, db *DB, class Class, err error) {
				if !assert.Error(t, err) {
					return
				}
				_, ok := errors.AsType[*dataerr.NotFoundError](err)
				assert.True(t, ok, "expected data NotFoundError in error chain")
				assert.ErrorIs(t, err, sql.ErrNoRows)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			classCode: "c1",
			assertFunc: func(t *testing.T, db *DB, class Class, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			class, err := db.GetClassByClassCode(tt.classCode)
			tt.assertFunc(t, db, class, err)
		})
	}
}

func TestAddPendingConfirmation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		setup        func(t *testing.T, db *DB)
		confirmation PendingConfirmation
		assertFunc   func(t *testing.T, db *DB, err error)
	}{
		{
			name: "should successfully add confirmation",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				_, err = db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})
				require.NoError(t, err)
				err = db.EnsureUser(User{UserID: "u1", DisplayName: "n1"})
				require.NoError(t, err)
			},
			confirmation: PendingConfirmation{MessageID: "m1", UserID: "u1", ProblemID: 1},
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.NoError(t, err)
				var count int
				err = db.conn.QueryRow("SELECT COUNT(*) FROM pending_confirmations WHERE message_id = 'm1'").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 1, count)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			confirmation: PendingConfirmation{MessageID: "m1", UserID: "u1", ProblemID: 1},
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			err := db.AddPendingConfirmation(tt.confirmation)
			tt.assertFunc(t, db, err)
		})
	}
}

func TestAddPendingConfirmationClassifiesForeignKeyFailuresAccurately(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		setup        func(t *testing.T, db *DB)
		confirmation PendingConfirmation
	}{
		{
			name: "missing user",
			setup: func(t *testing.T, db *DB) {
				classID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				_, err = db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: classID})
				require.NoError(t, err)
			},
			confirmation: PendingConfirmation{MessageID: "m1", UserID: "missing", ProblemID: 1},
		},
		{
			name: "missing problem",
			setup: func(t *testing.T, db *DB) {
				err := db.EnsureUser(User{UserID: "u1", DisplayName: "n1"})
				require.NoError(t, err)
			},
			confirmation: PendingConfirmation{MessageID: "m1", UserID: "u1", ProblemID: 999},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			tt.setup(t, db)

			err := db.AddPendingConfirmation(tt.confirmation)
			require.Error(t, err)
			_, ok := errors.AsType[*dataerr.ForeignKeyViolationError](err)
			assert.True(t, ok, "expected ForeignKeyViolationError in error chain")
			_, userMissing := errors.AsType[*dataerr.UserMissingFromDatabaseError](err)
			assert.False(t, userMissing, "a generic foreign-key failure must not be mislabeled as a missing user")
		})
	}
}

func TestExistsPendingConfirmation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		messageID  string
		assertFunc func(t *testing.T, db *DB, exists bool, err error)
	}{
		{
			name: "should return true if exists",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				pID, err := db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})
				require.NoError(t, err)
				err = db.EnsureUser(User{UserID: "u1", DisplayName: "n1"})
				require.NoError(t, err)
				err = db.AddPendingConfirmation(PendingConfirmation{MessageID: "m1", UserID: "u1", ProblemID: pID})
				require.NoError(t, err)
			},
			messageID: "m1",
			assertFunc: func(t *testing.T, db *DB, exists bool, err error) {
				assert.NoError(t, err)
				assert.True(t, exists)
			},
		},
		{
			name:      "should return false if not exists",
			messageID: "missing",
			assertFunc: func(t *testing.T, db *DB, exists bool, err error) {
				assert.NoError(t, err)
				assert.False(t, exists)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			messageID: "m1",
			assertFunc: func(t *testing.T, db *DB, exists bool, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			exists, err := db.ExistsPendingConfirmation(tt.messageID)
			tt.assertFunc(t, db, exists, err)
		})
	}
}

func TestGetPendingConfirmation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		messageID  string
		assertFunc func(t *testing.T, db *DB, userID string, problemID int64, err error)
	}{
		{
			name: "should successfully get confirmation details",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				pID, err := db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})
				require.NoError(t, err)
				err = db.EnsureUser(User{UserID: "u1", DisplayName: "n1"})
				require.NoError(t, err)
				err = db.AddPendingConfirmation(PendingConfirmation{MessageID: "m1", UserID: "u1", ProblemID: pID})
				require.NoError(t, err)
			},
			messageID: "m1",
			assertFunc: func(t *testing.T, db *DB, userID string, problemID int64, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "u1", userID)
				assert.Equal(t, int64(1), problemID)
			},
		},
		{
			name:      "should return error if not found",
			messageID: "missing",
			assertFunc: func(t *testing.T, db *DB, userID string, problemID int64, err error) {
				assert.Error(t, err)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			messageID: "m1",
			assertFunc: func(t *testing.T, db *DB, userID string, problemID int64, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			userID, problemID, err := db.GetPendingConfirmation(tt.messageID)
			tt.assertFunc(t, db, userID, problemID, err)
		})
	}
}

func TestRemovePendingConfirmation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		messageID  string
		assertFunc func(t *testing.T, db *DB, err error)
	}{
		{
			name: "should successfully remove confirmation",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				pID, err := db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})
				require.NoError(t, err)
				err = db.EnsureUser(User{UserID: "u1", DisplayName: "n1"})
				require.NoError(t, err)
				err = db.AddPendingConfirmation(PendingConfirmation{MessageID: "m1", UserID: "u1", ProblemID: pID})
				require.NoError(t, err)
			},
			messageID: "m1",
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.NoError(t, err)
				var count int
				err = db.conn.QueryRow("SELECT COUNT(*) FROM pending_confirmations WHERE message_id = 'm1'").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			messageID: "m1",
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			err := db.RemovePendingConfirmation(tt.messageID)
			tt.assertFunc(t, db, err)
		})
	}
}

func TestGetProblemByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		problemID  int64
		assertFunc func(t *testing.T, db *DB, problem Problem, err error)
	}{
		{
			name: "should successfully get problem",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				_, err = db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})
				require.NoError(t, err)
			},
			problemID: 1,
			assertFunc: func(t *testing.T, db *DB, problem Problem, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "p1", problem.Name)
			},
		},
		{
			name:      "should return error if problem not found",
			problemID: 99,
			assertFunc: func(t *testing.T, db *DB, problem Problem, err error) {
				assert.Error(t, err)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			problemID: 1,
			assertFunc: func(t *testing.T, db *DB, problem Problem, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			problem, err := db.GetProblemByID(tt.problemID)
			tt.assertFunc(t, db, problem, err)
		})
	}
}

func TestGetActiveUsersByClassID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		classID    int64
		assertFunc func(t *testing.T, db *DB, users []User, err error)
	}{
		{
			name: "should only return users with at least one AC in the class",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				pID, err := db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})
				require.NoError(t, err)

				// User 1: Active (has AC)
				err = db.EnsureUser(User{UserID: "u1", DisplayName: "Active User"})
				require.NoError(t, err)
				err = db.AddAC("u1", pID)
				require.NoError(t, err)

				// User 2: Inactive (no AC in this class)
				err = db.EnsureUser(User{UserID: "u2", DisplayName: "Inactive User"})
				require.NoError(t, err)

				// User 3: Active in DIFFERENT class
				cID2, _ := db.CreateClass(Class{ClassCode: "c2", Name: "n2"})
				pID2, _ := db.AddProblem(Problem{Name: "p2", URL: "u2", ClassID: cID2})
				err = db.EnsureUser(User{UserID: "u3", DisplayName: "Other Class User"})
				require.NoError(t, err)
				err = db.AddAC("u3", pID2)
				require.NoError(t, err)
			},
			classID: 1,
			assertFunc: func(t *testing.T, db *DB, users []User, err error) {
				assert.NoError(t, err)
				assert.Len(t, users, 1)
				assert.Equal(t, "u1", users[0].UserID)
			},
		},
		{
			name: "should return empty if no users have solved anything",
			setup: func(t *testing.T, db *DB) {
				db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
			},
			classID: 1,
			assertFunc: func(t *testing.T, db *DB, users []User, err error) {
				assert.NoError(t, err)
				assert.Len(t, users, 0)
			},
		},
		{
			name: "should return users sorted by display name then user id",
			setup: func(t *testing.T, db *DB) {
				cID, _ := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				pID, _ := db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})

				// Insert unsorted
				_ = db.EnsureUser(User{UserID: "u3", DisplayName: "Zebra"})
				_ = db.EnsureUser(User{UserID: "u2", DisplayName: "Alpha"})
				_ = db.EnsureUser(User{UserID: "u1", DisplayName: "Alpha"})

				_ = db.AddAC("u3", pID)
				_ = db.AddAC("u2", pID)
				_ = db.AddAC("u1", pID)
			},
			classID: 1,
			assertFunc: func(t *testing.T, db *DB, users []User, err error) {
				assert.NoError(t, err)
				assert.Len(t, users, 3)
				// Alpha / u1 -> Alpha / u2 -> Zebra / u3
				assert.Equal(t, "u1", users[0].UserID)
				assert.Equal(t, "u2", users[1].UserID)
				assert.Equal(t, "u3", users[2].UserID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			users, err := db.GetActiveUsersByClassID(tt.classID)
			tt.assertFunc(t, db, users, err)
		})
	}
}

func TestGetAllClasses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		assertFunc func(t *testing.T, db *DB, classes []Class, err error)
	}{
		{
			name: "should return all classes",
			setup: func(t *testing.T, db *DB) {
				_, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				_, err = db.CreateClass(Class{ClassCode: "c2", Name: "n2"})
				require.NoError(t, err)
			},
			assertFunc: func(t *testing.T, db *DB, classes []Class, err error) {
				assert.NoError(t, err)
				assert.Len(t, classes, 2)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			assertFunc: func(t *testing.T, db *DB, classes []Class, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			classes, err := db.GetAllClasses()
			tt.assertFunc(t, db, classes, err)
		})
	}
}

func TestUpdateProblemByProblemID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		setup                   func(t *testing.T, db *DB)
		problemID               int64
		problem                 Problem
		checkCreatedAtUnchanged bool
		assertFunc              func(t *testing.T, db *DB, err error)
	}{
		{
			name: "should update problem details",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				_, err = db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})
				require.NoError(t, err)
			},
			problemID: 1,
			problem:   Problem{Name: "p2", URL: "u2", ClassID: 1},
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.NoError(t, err)
				var name string
				err = db.conn.QueryRow("SELECT name FROM problems WHERE id = 1").Scan(&name)
				assert.NoError(t, err)
				assert.Equal(t, "p2", name)
			},
		},
		{
			name: "should update priority without changing creation timestamp",
			setup: func(t *testing.T, db *DB) {
				classID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				_, err = db.AddProblem(Problem{
					Name:     "p1",
					URL:      "u1",
					ClassID:  classID,
					Priority: ptrStr("essential"),
				})
				require.NoError(t, err)
			},
			problemID: 1,
			problem: Problem{
				Name:      "p2",
				URL:       "u2",
				ClassID:   1,
				Priority:  ptrStr("discouraged"),
				CreatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			},
			checkCreatedAtUnchanged: true,
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.NoError(t, err)
				updated, err := db.GetProblemByID(1)
				if !assert.NoError(t, err) {
					return
				}
				assert.Equal(t, "p2", updated.Name)
				if assert.NotNil(t, updated.Priority) {
					assert.Equal(t, "discouraged", *updated.Priority)
				}
			},
		},
		{
			name:      "should return NotFoundError if problem not found",
			problemID: 99,
			problem:   Problem{Name: "p2", URL: "u2", ClassID: 1},
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.Error(t, err)
			},
		},
		{
			name: "should allow updating to a name already used in class",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				_, err = db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})
				require.NoError(t, err)
				_, err = db.AddProblem(Problem{Name: "p2", URL: "u2", ClassID: cID})
				require.NoError(t, err)
			},
			problemID: 1,
			problem:   Problem{Name: "p2", URL: "u3", ClassID: 1},
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.NoError(t, err)
				var count int
				err = db.conn.QueryRow("SELECT COUNT(*) FROM problems WHERE name = ? AND class_id = ?", "p2", 1).Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 2, count)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			problemID: 1,
			problem:   Problem{Name: "p2", URL: "u2", ClassID: 1},
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			var originalCreatedAt time.Time
			if tt.checkCreatedAtUnchanged {
				original, err := db.GetProblemByID(tt.problemID)
				if !assert.NoError(t, err) {
					return
				}
				originalCreatedAt = original.CreatedAt
			}

			err := db.UpdateProblemByProblemID(tt.problemID, tt.problem)
			tt.assertFunc(t, db, err)

			if tt.checkCreatedAtUnchanged {
				updated, err := db.GetProblemByID(tt.problemID)
				if assert.NoError(t, err) {
					assert.Equal(t, originalCreatedAt, updated.CreatedAt)
				}
			}
		})
	}
}

func TestGetClassesWhereUserIsActive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		userID     string
		assertFunc func(t *testing.T, db *DB, classes []Class, err error)
	}{
		{
			name: "should only return classes where the user has an AC",
			setup: func(t *testing.T, db *DB) {
				cID1, _ := db.CreateClass(Class{ClassCode: "c1", Name: "Class 1"})
				cID2, _ := db.CreateClass(Class{ClassCode: "c2", Name: "Class 2"})
				_ = db.EnsureUser(User{UserID: "u1", DisplayName: "User 1"})

				pID1, _ := db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID1})
				_, _ = db.AddProblem(Problem{Name: "p2", URL: "u2", ClassID: cID2})

				// User 1 solved a problem in Class 1, but NOT in Class 2
				_ = db.AddAC("u1", pID1)
			},
			userID: "u1",
			assertFunc: func(t *testing.T, db *DB, classes []Class, err error) {
				assert.NoError(t, err)
				assert.Len(t, classes, 1)
				assert.Equal(t, "c1", classes[0].ClassCode)
			},
		},
		{
			name: "should return empty if user has no solves anywhere",
			setup: func(t *testing.T, db *DB) {
				_, _ = db.CreateClass(Class{ClassCode: "c1", Name: "Class 1"})
				_ = db.EnsureUser(User{UserID: "u1", DisplayName: "User 1"})
			},
			userID: "u1",
			assertFunc: func(t *testing.T, db *DB, classes []Class, err error) {
				assert.NoError(t, err)
				assert.Len(t, classes, 0)
			},
		},
		{
			name: "should return classes sorted by class code",
			setup: func(t *testing.T, db *DB) {
				cID3, _ := db.CreateClass(Class{ClassCode: "c3", Name: "Class 3"})
				cID1, _ := db.CreateClass(Class{ClassCode: "c1", Name: "Class 1"})
				cID2, _ := db.CreateClass(Class{ClassCode: "c2", Name: "Class 2"})
				_ = db.EnsureUser(User{UserID: "u1", DisplayName: "User 1"})

				pID3, _ := db.AddProblem(Problem{Name: "p3", URL: "u3", ClassID: cID3})
				pID1, _ := db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID1})
				pID2, _ := db.AddProblem(Problem{Name: "p2", URL: "u2", ClassID: cID2})

				// Insert unsorted
				_ = db.AddAC("u1", pID3)
				_ = db.AddAC("u1", pID1)
				_ = db.AddAC("u1", pID2)
			},
			userID: "u1",
			assertFunc: func(t *testing.T, db *DB, classes []Class, err error) {
				assert.NoError(t, err)
				assert.Len(t, classes, 3)
				assert.Equal(t, "c1", classes[0].ClassCode)
				assert.Equal(t, "c2", classes[1].ClassCode)
				assert.Equal(t, "c3", classes[2].ClassCode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			classes, err := db.GetClassesWhereUserIsActive(tt.userID)
			tt.assertFunc(t, db, classes, err)
		})
	}
}

func TestHasUserSolvedProblem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		userID     string
		problemID  int64
		assertFunc func(t *testing.T, db *DB, solved bool, err error)
	}{
		{
			name: "should return true if solved",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				pID, err := db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})
				require.NoError(t, err)
				err = db.EnsureUser(User{UserID: "u1", DisplayName: "n1"})
				require.NoError(t, err)
				err = db.AddAC("u1", pID)
				require.NoError(t, err)
			},
			userID:    "u1",
			problemID: 1,
			assertFunc: func(t *testing.T, db *DB, solved bool, err error) {
				assert.NoError(t, err)
				assert.True(t, solved)
			},
		},
		{
			name:      "should return false if not solved",
			userID:    "u1",
			problemID: 1,
			assertFunc: func(t *testing.T, db *DB, solved bool, err error) {
				assert.NoError(t, err)
				assert.False(t, solved)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			userID:    "u1",
			problemID: 1,
			assertFunc: func(t *testing.T, db *DB, solved bool, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			solved, err := db.HasUserSolvedProblem(tt.userID, tt.problemID)
			tt.assertFunc(t, db, solved, err)
		})
	}
}

func TestExterminateUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		userID     string
		assertFunc func(t *testing.T, db *DB, err error)
	}{
		{
			name: "should completely wipe user and data",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				pID, err := db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})
				require.NoError(t, err)
				err = db.EnsureUser(User{UserID: "u1", DisplayName: "n1"})
				require.NoError(t, err)
				err = db.AddAC("u1", pID)
				require.NoError(t, err)
			},
			userID: "u1",
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.NoError(t, err)
				var count int
				err = db.conn.QueryRow("SELECT COUNT(*) FROM users WHERE user_id = 'u1'").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count)
				err = db.conn.QueryRow("SELECT COUNT(*) FROM solved_problems WHERE user_id = 'u1'").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count)
			},
		},
		{
			name: "should return error if db closed (transaction fail)",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			userID: "u1",
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			err := db.ExterminateUser(tt.userID)
			tt.assertFunc(t, db, err)
		})
	}
}

func TestGetClassesByAccessRoles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		roles      []string
		assertFunc func(t *testing.T, db *DB, classes []Class, err error)
	}{
		{
			name: "should return empty if no roles provided",
			setup: func(t *testing.T, db *DB) {
				_, _ = db.CreateClass(Class{ClassCode: "c1", Name: "Class 1"})
				_ = db.AddClassAccessRole(ClassAccessRole{ClassCode: "c1", RoleID: "r1"})
			},
			roles: []string{},
			assertFunc: func(t *testing.T, db *DB, classes []Class, err error) {
				assert.NoError(t, err)
				assert.Len(t, classes, 0)
			},
		},
		{
			name: "should return classes sorted by class code for multiple roles",
			setup: func(t *testing.T, db *DB) {
				_, _ = db.CreateClass(Class{ClassCode: "c3", Name: "Class 3"})
				_, _ = db.CreateClass(Class{ClassCode: "c1", Name: "Class 1"})
				_, _ = db.CreateClass(Class{ClassCode: "c2", Name: "Class 2"})

				_ = db.AddClassAccessRole(ClassAccessRole{ClassCode: "c3", RoleID: "r2"})
				_ = db.AddClassAccessRole(ClassAccessRole{ClassCode: "c1", RoleID: "r1"})
				_ = db.AddClassAccessRole(ClassAccessRole{ClassCode: "c2", RoleID: "r1"})
			},
			roles: []string{"r1", "r2"},
			assertFunc: func(t *testing.T, db *DB, classes []Class, err error) {
				assert.NoError(t, err)
				assert.Len(t, classes, 3)
				assert.Equal(t, "c1", classes[0].ClassCode)
				assert.Equal(t, "c2", classes[1].ClassCode)
				assert.Equal(t, "c3", classes[2].ClassCode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			defer db.CloseDB()

			if tt.setup != nil {
				tt.setup(t, db)
			}

			classes, err := db.GetClassesByAccessRoles(tt.roles)
			tt.assertFunc(t, db, classes, err)
		})
	}
}

func TestACValidationDataUsesOneReadAndPreservesRequestedPairs(t *testing.T) {
	conn, err := sql.Open("sqlite3_batch_lookup_count", ":memory:")
	require.NoError(t, err)
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	require.NoError(t, applySchema(conn))
	db := &DB{conn: conn}
	targets := make([]ACKey, 100)
	expectedProblems := make(map[ACKey]Problem)
	for index := range 100 {
		code := fmt.Sprintf("ac-class-%d", index)
		classID, err := db.CreateClass(Class{ClassCode: code, Name: code})
		require.NoError(t, err)
		require.NoError(t, db.AddClassAccessRole(ClassAccessRole{ClassCode: code, RoleID: "student"}))
		id, err := db.AddProblem(Problem{Name: code, URL: "https://example.com", ClassID: classID, Difficulty: ptrStr("easy"), Topic: ptrStr("Arrays")})
		require.NoError(t, err)
		userID := fmt.Sprint(index + 1)
		require.NoError(t, db.EnsureUser(User{UserID: userID}))
		require.NoError(t, db.AddAC(userID, id))
		targets[index] = ACKey{UserID: userID, ProblemID: id}
		expectedProblems[targets[index]], err = db.GetProblemByID(id)
		require.NoError(t, err)
	}
	for _, size := range []int{1, 100} {
		batchLookupQueryCount.Store(0)
		facts, err := db.GetACValidationData(targets[:size])
		require.NoError(t, err)
		require.Len(t, facts, size)
		assert.EqualValues(t, 1, batchLookupQueryCount.Load())
		for index, target := range targets[:size] {
			fact := facts[target]
			require.NotNil(t, fact.Problem)
			assert.Equal(t, expectedProblems[target], *fact.Problem)
			require.NotNil(t, fact.Class)
			assert.Equal(t, fmt.Sprintf("ac-class-%d", index), fact.Class.ClassCode)
			assert.Equal(t, fact.Problem.ClassID, fact.Class.ID)
			assert.Equal(t, []string{"student"}, fact.AccessRoleIDs)
			assert.True(t, fact.AlreadyRecorded)
		}
	}
	require.NoError(t, db.AddClassAccessRole(ClassAccessRole{ClassCode: "ac-class-0", RoleID: "second-role"}))
	crossPair := ACKey{UserID: targets[0].UserID, ProblemID: targets[1].ProblemID}
	missingProblem := ACKey{UserID: "absent", ProblemID: 999999}
	facts, err := db.GetACValidationData([]ACKey{targets[0], targets[0], crossPair, missingProblem})
	require.NoError(t, err)
	require.Len(t, facts, 3)
	assert.ElementsMatch(t, []string{"student", "second-role"}, facts[targets[0]].AccessRoleIDs)
	assert.True(t, facts[targets[0]].AlreadyRecorded)
	assert.False(t, facts[crossPair].AlreadyRecorded, "existing solves must match the exact pair")
	assert.Equal(t, []string{"student"}, facts[crossPair].AccessRoleIDs, "roles belong to the problem's class")
	assert.Nil(t, facts[missingProblem].Problem)
	assert.Nil(t, facts[missingProblem].Class)
	assert.Empty(t, facts[missingProblem].AccessRoleIDs)
	assert.False(t, facts[missingProblem].AlreadyRecorded)

	// Optional metadata and classes without access roles remain valid read results.
	classID, err := db.CreateClass(Class{ClassCode: "no-roles", Name: "No roles"})
	require.NoError(t, err)
	id, err := db.AddProblem(Problem{Name: "No metadata", URL: "https://example.com", ClassID: classID})
	require.NoError(t, err)
	target := ACKey{UserID: "new-student", ProblemID: id}
	facts, err = db.GetACValidationData([]ACKey{target})
	require.NoError(t, err)
	require.NotNil(t, facts[target].Problem)
	assert.Nil(t, facts[target].Problem.Difficulty)
	assert.Nil(t, facts[target].Problem.Topic)
	assert.Nil(t, facts[target].Problem.Priority)
	assert.Empty(t, facts[target].AccessRoleIDs)
	assert.False(t, facts[target].AlreadyRecorded)

	// This raw test connection allows an orphaned class reference for legacy-data coverage.
	orphanID, err := db.AddProblem(Problem{Name: "Orphan", URL: "https://example.com", ClassID: 999999})
	require.NoError(t, err)
	orphan := ACKey{UserID: "123", ProblemID: orphanID}
	facts, err = db.GetACValidationData([]ACKey{orphan})
	require.NoError(t, err)
	require.NotNil(t, facts[orphan].Problem)
	assert.Equal(t, int64(999999), facts[orphan].Problem.ClassID)
	assert.Nil(t, facts[orphan].Class)

	batchLookupQueryCount.Store(0)
	facts, err = db.GetACValidationData(nil)
	require.NoError(t, err)
	assert.Empty(t, facts)
	assert.Zero(t, batchLookupQueryCount.Load())
	require.NoError(t, conn.Close())
	facts, err = db.GetACValidationData(targets[:1])
	assert.Error(t, err)
	assert.Nil(t, facts, "query failures must not expose partial facts")
}

func TestAddACBatchUsesTwoWritesAndPreservesExistingSolves(t *testing.T) {
	conn, err := sql.Open("sqlite3_batch_lookup_count", ":memory:")
	require.NoError(t, err)
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	require.NoError(t, applySchema(conn))
	db := &DB{conn: conn}
	classID, err := db.CreateClass(Class{ClassCode: "batch", Name: "Batch"})
	require.NoError(t, err)
	problemID, err := db.AddProblem(Problem{Name: "Problem", ClassID: classID})
	require.NoError(t, err)

	for _, size := range []int{0, 1, 100} {
		targets := make([]ACKey, size)
		for index := range targets {
			targets[index] = ACKey{UserID: fmt.Sprintf("size-%d-user-%d", size, index), ProblemID: problemID}
		}
		batchWriteQueryCount.Store(0)
		inserted, err := db.AddACBatch(targets)
		require.NoError(t, err)
		assert.Equal(t, size, inserted)
		expectedWrites := int64(2)
		if size == 0 {
			expectedWrites = 0
		}
		assert.Equal(t, expectedWrites, batchWriteQueryCount.Load())
		for _, target := range targets {
			solved, err := db.HasUserSolvedProblem(target.UserID, target.ProblemID)
			require.NoError(t, err)
			assert.True(t, solved)
		}
	}

	// Repeats and existing solves must not overwrite student names or timestamps.
	require.NoError(t, db.EnsureUser(User{UserID: "existing", DisplayName: "Ana"}))
	require.NoError(t, db.AddAC("existing", problemID))
	_, err = conn.Exec("UPDATE solved_problems SET solved_at = '2000-01-01 00:00:00' WHERE user_id = ?", "existing")
	require.NoError(t, err)
	newTarget := ACKey{UserID: "new'; DROP TABLE users; --", ProblemID: problemID}
	inserted, err := db.AddACBatch([]ACKey{{UserID: "existing", ProblemID: problemID}, newTarget, newTarget})
	require.NoError(t, err)
	assert.Equal(t, 1, inserted)
	user, err := db.GetUserByUserID("existing")
	require.NoError(t, err)
	assert.Equal(t, "Ana", user.DisplayName)
	var solvedAt time.Time
	require.NoError(t, conn.QueryRow("SELECT solved_at FROM solved_problems WHERE user_id = ?", "existing").Scan(&solvedAt))
	assert.Equal(t, 2000, solvedAt.Year())
	_, err = db.GetUserByUserID(newTarget.UserID)
	require.NoError(t, err, "SQL-looking values must be stored literally")
}

func TestAddACBatchRollsBackUsersAndSolvesOnFailure(t *testing.T) {
	t.Parallel()
	for _, failure := range []string{"invalid problem", "user insert", "solve insert"} {
		t.Run(failure, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			t.Cleanup(db.CloseDB)
			classID, err := db.CreateClass(Class{ClassCode: "batch", Name: "Batch"})
			require.NoError(t, err)
			problemID, err := db.AddProblem(Problem{Name: "Problem", ClassID: classID})
			require.NoError(t, err)
			require.NoError(t, db.EnsureUser(User{UserID: "existing", DisplayName: "Ana"}))
			require.NoError(t, db.AddAC("existing", problemID))
			targets := []ACKey{{UserID: "first", ProblemID: problemID}, {UserID: "second", ProblemID: problemID}}
			switch failure {
			case "invalid problem":
				targets[1].ProblemID = -1
			case "user insert":
				_, err = db.conn.Exec(`CREATE TRIGGER reject_user BEFORE INSERT ON users WHEN NEW.user_id = 'second' BEGIN SELECT RAISE(ABORT, 'user failure'); END`)
				require.NoError(t, err)
			case "solve insert":
				_, err = db.conn.Exec(`CREATE TRIGGER reject_solve BEFORE INSERT ON solved_problems WHEN NEW.user_id = 'second' BEGIN SELECT RAISE(ABORT, 'solve failure'); END`)
				require.NoError(t, err)
			}
			inserted, err := db.AddACBatch(targets)
			require.Error(t, err)
			assert.Zero(t, inserted)
			for _, userID := range []string{"first", "second"} {
				_, err := db.GetUserByUserID(userID)
				assert.Error(t, err, "new users must roll back")
				solved, err := db.HasUserSolvedProblem(userID, problemID)
				require.NoError(t, err)
				assert.False(t, solved)
			}
			solved, err := db.HasUserSolvedProblem("existing", problemID)
			require.NoError(t, err)
			assert.True(t, solved, "existing solves must survive a failed batch")
		})
	}
}
