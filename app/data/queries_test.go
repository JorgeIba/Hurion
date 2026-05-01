package data

import (
	"testing"

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
			problem: Problem{Name: "p1", URL: "u1", ClassID: 1},
			assertFunc: func(t *testing.T, db *DB, id int64, err error) {
				assert.NoError(t, err)
				assert.Greater(t, id, int64(0))
				var name string
				err = db.conn.QueryRow("SELECT name FROM problems WHERE id = ?", id).Scan(&name)
				assert.NoError(t, err)
				assert.Equal(t, "p1", name)
			},
		},
		{
			name: "should fail if problem name exists in class",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				_, err = db.AddProblem(Problem{Name: "p1", URL: "u1", ClassID: cID})
				require.NoError(t, err)
			},
			problem: Problem{Name: "p1", URL: "u2", ClassID: 1},
			assertFunc: func(t *testing.T, db *DB, id int64, err error) {
				assert.Error(t, err)
				assert.Equal(t, int64(0), id)
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

func TestClearClass(t *testing.T) {
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
				_, err = db.conn.Exec("INSERT INTO enrollments (user_id, class_id) VALUES (?, ?)", "u1", cID)
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
				err = db.conn.QueryRow("SELECT COUNT(*) FROM enrollments WHERE class_id = 1").Scan(&count)
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

			err := db.ClearClass(tt.classID)
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
				assert.Error(t, err)
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
				_, err = db.conn.Exec("INSERT INTO enrollments (user_id, class_id) VALUES (?, ?)", "u1", cID)
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

func TestGetAllUsersByClassID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		classID    int64
		assertFunc func(t *testing.T, db *DB, users []User, err error)
	}{
		{
			name: "should get all users in class",
			setup: func(t *testing.T, db *DB) {
				cID, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				err = db.EnsureUser(User{UserID: "u1", DisplayName: "User 1"})
				require.NoError(t, err)
				err = db.EnsureUser(User{UserID: "u2", DisplayName: "User 2"})
				require.NoError(t, err)
				_, err = db.conn.Exec("INSERT INTO enrollments (user_id, class_id) VALUES (?, ?)", "u1", cID)
				require.NoError(t, err)
				_, err = db.conn.Exec("INSERT INTO enrollments (user_id, class_id) VALUES (?, ?)", "u2", cID)
				require.NoError(t, err)
			},
			classID: 1,
			assertFunc: func(t *testing.T, db *DB, users []User, err error) {
				assert.NoError(t, err)
				assert.Len(t, users, 2)
			},
		},
		{
			name: "should return error if db closed",
			setup: func(t *testing.T, db *DB) {
				db.CloseDB()
			},
			classID: 1,
			assertFunc: func(t *testing.T, db *DB, users []User, err error) {
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

			users, err := db.GetAllUsersByClassID(tt.classID)
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
		name       string
		setup      func(t *testing.T, db *DB)
		problemID  int64
		problem    Problem
		assertFunc func(t *testing.T, db *DB, err error)
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
			name:      "should return NotFoundError if problem not found",
			problemID: 99,
			problem:   Problem{Name: "p2", URL: "u2", ClassID: 1},
			assertFunc: func(t *testing.T, db *DB, err error) {
				assert.Error(t, err)
			},
		},
		{
			name: "should return RowConflictError if problem name exists in class",
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
				assert.Error(t, err)
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

			err := db.UpdateProblemByProblemID(tt.problemID, tt.problem)
			tt.assertFunc(t, db, err)
		})
	}
}

func TestGetClassesByUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, db *DB)
		userID     string
		assertFunc func(t *testing.T, db *DB, classes []Class, err error)
	}{
		{
			name: "should return classes user is enrolled in",
			setup: func(t *testing.T, db *DB) {
				cID1, err := db.CreateClass(Class{ClassCode: "c1", Name: "n1"})
				require.NoError(t, err)
				cID2, err := db.CreateClass(Class{ClassCode: "c2", Name: "n2"})
				require.NoError(t, err)
				err = db.EnsureUser(User{UserID: "u1", DisplayName: "n1"})
				require.NoError(t, err)
				_, err = db.conn.Exec("INSERT INTO enrollments (user_id, class_id) VALUES (?, ?)", "u1", cID1)
				require.NoError(t, err)
				_, err = db.conn.Exec("INSERT INTO enrollments (user_id, class_id) VALUES (?, ?)", "u1", cID2)
				require.NoError(t, err)
			},
			userID: "u1",
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
			userID: "u1",
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

			classes, err := db.GetClassesByUserID(tt.userID)
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
