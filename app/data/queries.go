package data

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/mattn/go-sqlite3"
)

// EnsureUser ensures that a user with the given ID and display name exists in the database.
func (db *DB) EnsureUser(user User) error {
	// Update displayname if the user already exists, but only if the new display name is not empty and not "Unknown"
	query := `
		INSERT INTO users (user_id, display_name)
		VALUES (?1, CASE WHEN ?2 = '' THEN 'Unknown' ELSE ?2 END)
		ON CONFLICT(user_id) DO UPDATE SET 
			display_name = CASE 
				WHEN excluded.display_name <> 'Unknown' THEN excluded.display_name 
				ELSE users.display_name 
			END;
	`

	if _, err := db.conn.Exec(query, user.UserID, user.DisplayName); err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not ensure user %s", user.UserID), err)
	}

	return nil
}

func (db *DB) AddProblem(problem Problem) (int64, error) {
	query := `
		INSERT INTO problems (name, url, class_id, difficulty, topic, priority)
		VALUES (?, ?, ?, ?, ?, ?);
	`
	response, err := db.conn.Exec(query, problem.Name, problem.URL, problem.ClassID, problem.Difficulty, problem.Topic, problem.Priority)
	if err != nil {
		return 0, dataerr.NewQueryError(fmt.Sprintf("could not add the problem %s into the class id %d", problem.Name, problem.ClassID), err)
	}

	if rowsAffected, err := response.RowsAffected(); err != nil {
		return 0, dataerr.NewQueryError(fmt.Sprintf("could not determine rows affected when inserting problem %s into class %d", problem.Name, problem.ClassID), err)
	} else if rowsAffected == 0 {
		return 0, dataerr.NewQueryError(fmt.Sprintf("problem %s was not inserted into class %d", problem.Name, problem.ClassID), nil)
	}

	problemID, err := response.LastInsertId()
	if err != nil {
		return 0, dataerr.NewQueryError("could not fetch last insert id", err)
	}

	return problemID, nil
}

// AddProblemsBatch inserts the problems in order within a transaction and
// returns their IDs only after a successful commit.
func (db *DB) AddProblemsBatch(problems []Problem) ([]int64, error) {
	if len(problems) == 0 {
		return []int64{}, nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return nil, dataerr.NewQueryError("could not begin problem batch", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO problems (name, url, class_id, difficulty, topic, priority)
		VALUES (?, ?, ?, ?, ?, ?);
	`

	problemIDs := make([]int64, 0, len(problems))
	for index, problem := range problems {
		result, err := tx.Exec(query, problem.Name, problem.URL, problem.ClassID, problem.Difficulty, problem.Topic, problem.Priority)
		if err != nil {
			return nil, dataerr.NewQueryError(fmt.Sprintf("could not insert problem batch entry %d", index+1), err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return nil, dataerr.NewQueryError(fmt.Sprintf("could not determine rows affected for problem batch entry %d", index+1), err)
		}

		if rowsAffected == 0 {
			return nil, dataerr.NewQueryError(fmt.Sprintf("problem batch entry %d was not inserted", index+1), nil)
		}

		problemID, err := result.LastInsertId()
		if err != nil {
			return nil, dataerr.NewQueryError(fmt.Sprintf("could not fetch ID for problem batch entry %d", index+1), err)
		}

		problemIDs = append(problemIDs, problemID)
	}

	if err := tx.Commit(); err != nil {
		return nil, dataerr.NewQueryError("could not commit problem batch", err)
	}

	// The SQLite update hook runs before commit; notify again so observers read committed data.
	notifyDataChanged()

	return problemIDs, nil
}

func (db *DB) RemoveProblemByID(problemID int64) (int64, error) {
	query := "DELETE FROM problems WHERE id = ?"
	if _, err := db.conn.Exec(query, problemID); err != nil {
		return 0, dataerr.NewQueryError(fmt.Sprintf("could not remove problem %d", problemID), err)
	}

	return 1, nil
}

func (db *DB) AddAC(userID string, problemID int64) error {
	query := `
		INSERT INTO solved_problems (user_id, problem_id, solved_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, problem_id) DO NOTHING;
	`
	_, err := db.conn.Exec(query, userID, problemID)
	if err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not AC the problem %d for user %s", problemID, userID), err)
	}

	return nil
}

// AddACBatch ensures users and records ACs atomically. The returned count includes
// only inserted pairs; existing and repeated pairs leave their timestamps intact.
func (db *DB) AddACBatch(targets []ACKey, users []User) (int, error) {
	if len(targets) == 0 {
		return 0, nil
	}

	// Carry verified Discord names into the same transaction as the solves.
	userNames := make(map[string]string, len(users))
	for _, user := range users {
		userNames[user.UserID] = strings.TrimSpace(user.DisplayName)
	}

	// Bind submitted pairs and names once for the two bulk statements.
	pairPlaceholders := strings.TrimSuffix(strings.Repeat("(?, ?, ?),", len(targets)), ",")
	queryArgs := make([]any, 0, 3*len(targets))
	for _, target := range targets {
		name := userNames[target.UserID]
		if name == "" {
			name = "Unknown"
		}
		queryArgs = append(queryArgs, target.UserID, target.ProblemID, name)
	}
	requestedPairs := "WITH requested(user_id, problem_id, display_name) AS (VALUES " + pairPlaceholders + ") "

	// Keep user creation and solve registration in the same transaction.
	tx, err := db.conn.Begin()
	if err != nil {
		return 0, dataerr.NewQueryError("could not begin AC batch", err)
	}
	defer tx.Rollback()

	// Register users and refresh valid Discord names, matching EnsureUser.
	// WHERE true disambiguates SELECT from the following ON CONFLICT in SQLite.
	ensureUsers := requestedPairs + `
		INSERT INTO users (user_id, display_name)
		SELECT DISTINCT user_id, display_name FROM requested WHERE true
		ON CONFLICT(user_id) DO UPDATE SET display_name = excluded.display_name
		WHERE excluded.display_name <> 'Unknown'
	`

	if _, err := tx.Exec(ensureUsers, queryArgs...); err != nil {
		return 0, dataerr.NewQueryError("could not ensure AC batch users", err)
	}

	// Register solved-problem records, preserving existing ACs and their timestamps.
	insertACs := requestedPairs + `
		INSERT INTO solved_problems (user_id, problem_id, solved_at)
		SELECT user_id, problem_id, CURRENT_TIMESTAMP FROM requested WHERE true
		ON CONFLICT(user_id, problem_id) DO NOTHING
	`

	insertResult, err := tx.Exec(insertACs, queryArgs...)
	if err != nil {
		return 0, dataerr.NewQueryError("could not insert AC batch", err)
	}

	insertedCount, err := insertResult.RowsAffected()
	if err != nil {
		return 0, dataerr.NewQueryError("could not count inserted ACs", err)
	}

	// Report acknowledged registrations only after commit succeeds.
	if err := tx.Commit(); err != nil {
		return 0, dataerr.NewQueryError("could not commit AC batch", err)
	}
	// The SQLite update hook runs before commit; notify again so observers read committed data.
	notifyDataChanged()

	return int(insertedCount), nil
}

func (db *DB) RemoveAC(userID string, problemID int64) error {
	query := `
		DELETE FROM solved_problems
		WHERE user_id = ? AND problem_id = ?
	`
	_, err := db.conn.Exec(query, userID, problemID)
	if err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not remove the AC of problem %d for user %s", problemID, userID), err)
	}

	return nil
}

func (db *DB) GetProblemsByClass(classID int64) ([]Problem, error) {
	query := `
		SELECT id, name, url, difficulty, topic, priority, created_at
		FROM problems
		WHERE class_id = ?
	`

	rows, err := db.conn.Query(query, classID)
	if err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("could not fetch problems from class id %d", classID), err)
	}
	defer rows.Close()

	problems := make([]Problem, 0)

	for rows.Next() {
		var problem Problem
		if err := rows.Scan(&problem.ID, &problem.Name, &problem.URL, &problem.Difficulty, &problem.Topic, &problem.Priority, &problem.CreatedAt); err != nil {
			return nil, dataerr.NewQueryError(fmt.Sprintf("could not scan problem when fetching problems from class id %d", classID), err)
		}

		problem.ClassID = classID
		problems = append(problems, problem)
	}

	if err := rows.Err(); err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("error during rows iteration when fetching problems from class id %d", classID), err)
	}

	return problems, nil
}

func (db *DB) GetSolvedProblemsByUserAndClass(userID string, classID int64) ([]SolvedProblem, error) {
	query := `
		SELECT sp.problem_id, sp.solved_at
		FROM solved_problems sp
		JOIN problems p ON sp.problem_id = p.id
		WHERE sp.user_id = ? AND p.class_id = ?
	`

	rows, err := db.conn.Query(query, userID, classID)
	if err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("could not get solved problems for user %s in class %d", userID, classID), err)
	}
	defer rows.Close()

	solvedProblems := make([]SolvedProblem, 0)

	for rows.Next() {
		var problemID int64
		var solvedAt time.Time

		if err := rows.Scan(&problemID, &solvedAt); err != nil {
			return nil, dataerr.NewQueryError(fmt.Sprintf("could not scan row for problems by user %s in class %d", userID, classID), err)
		}

		solvedProblems = append(solvedProblems, SolvedProblem{
			ProblemID: problemID,
			UserID:    userID,
			SolvedAt:  solvedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("error iterating over rows for problems by user %s in class %d", userID, classID), err)
	}

	return solvedProblems, nil
}

func (db *DB) GetAllSolvedProblemsByClass(classID int64) (map[User][]SolvedProblem, error) {
	query := `
		SELECT 
			sp.user_id,
			u.display_name,
			sp.problem_id,
			sp.solved_at
		FROM solved_problems sp
		JOIN problems p ON sp.problem_id = p.id
		JOIN users u ON sp.user_id = u.user_id
		WHERE p.class_id = ?
	`
	rows, err := db.conn.Query(query, classID)
	if err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("could not get solved problems for the class %d", classID), err)
	}
	defer rows.Close()

	solvedProblemsIDByUser := make(map[User][]SolvedProblem)

	for rows.Next() {
		var userID, displayUserID string
		var problemID int64
		var solvedAt time.Time

		if err := rows.Scan(&userID, &displayUserID, &problemID, &solvedAt); err != nil {
			return nil, dataerr.NewQueryError("could not scan row", err)
		}

		user := User{
			UserID:      userID,
			DisplayName: displayUserID,
		}
		solvedProblemsIDByUser[user] = append(solvedProblemsIDByUser[user], SolvedProblem{UserID: userID, ProblemID: problemID, SolvedAt: solvedAt})
	}

	if err := rows.Err(); err != nil {
		return nil, dataerr.NewQueryError("error iterating over rows", err)
	}

	return solvedProblemsIDByUser, nil
}

func (db *DB) ClearClassContents(classID int64) error {
	query := "DELETE FROM problems WHERE class_id = ?"
	if _, err := db.conn.Exec(query, classID); err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not clear class contents for %d", classID), err)
	}

	return nil
}

func (db *DB) DeleteClass(classID int64) error {
	query := "DELETE FROM classes WHERE id = ?"
	if _, err := db.conn.Exec(query, classID); err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not delete class %d", classID), err)
	}

	return nil
}

func (db *DB) CreateClass(class Class) (int64, error) {
	query := `
		INSERT INTO classes (class_code, name) 
		VALUES (?, ?)
		ON CONFLICT(class_code) DO NOTHING;
	`

	res, err := db.conn.Exec(query, class.ClassCode, class.Name)
	if err != nil {
		return 0, dataerr.NewQueryError(fmt.Sprintf("could not create class %s with name %s", class.ClassCode, class.Name), err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, dataerr.NewQueryError("could not check affected rows", err)
	}

	if rowsAffected == 0 {
		return 0, dataerr.NewRowConflictError(fmt.Sprintf("class with code %s already exists", class.ClassCode), nil)
	}

	classID, err := res.LastInsertId()
	if err != nil {
		return 0, dataerr.NewQueryError("could not get last insert ID", err)
	}

	return classID, nil
}

func (db *DB) AddClassroom(classroom Classroom) error {
	query := `
		INSERT INTO classrooms (room_id, class_code)
		VALUES (?, ?)
		ON CONFLICT(room_id) DO UPDATE SET class_code = EXCLUDED.class_code;
	`

	if _, err := db.conn.Exec(query, classroom.RoomID, classroom.ClassCode); err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not add classroom %s for class %s", classroom.RoomID, classroom.ClassCode), err)
	}

	return nil
}

func (db *DB) RemoveClassroomByRoomID(roomID string) error {
	query := `
		DELETE FROM classrooms
		WHERE room_id = ?;
	`

	if _, err := db.conn.Exec(query, roomID); err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not remove classroom %s", roomID), err)
	}

	return nil
}

func (db *DB) AddClassAccessRole(classRole ClassAccessRole) error {
	query := `
		INSERT INTO class_access_roles (class_code, role_id)
		VALUES (?, ?)
		ON CONFLICT(class_code, role_id) DO NOTHING;
	`

	if _, err := db.conn.Exec(query, classRole.ClassCode, classRole.RoleID); err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not add role %s for class %s", classRole.RoleID, classRole.ClassCode), err)
	}

	return nil
}

func (db *DB) RemoveClassAccessRole(classRole ClassAccessRole) error {
	query := `
		DELETE FROM class_access_roles
		WHERE class_code = ? AND role_id = ?;
	`

	if _, err := db.conn.Exec(query, classRole.ClassCode, classRole.RoleID); err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not remove role %s from class %s", classRole.RoleID, classRole.ClassCode), err)
	}

	return nil
}

func (db *DB) GetClassCodeByRoomID(roomID string) (string, error) {
	query := `
		SELECT class_code
		FROM classrooms
		WHERE room_id = ?;
	`

	var classCode string
	err := db.conn.QueryRow(query, roomID).Scan(&classCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", dataerr.NewNotFoundError(fmt.Sprintf("classroom with id %s not found", roomID), err)
		}
		return "", dataerr.NewQueryError(fmt.Sprintf("could not get class code by room id %s", roomID), err)
	}

	return classCode, nil
}

func (db *DB) IsRoomRegistered(roomID string) (string, bool, error) {
	query := `
		SELECT class_code
		FROM classrooms
		WHERE room_id = ?;
	`

	var classCode string
	err := db.conn.QueryRow(query, roomID).Scan(&classCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, dataerr.NewQueryError(fmt.Sprintf("could not check if room %s is registered", roomID), err)
	}

	return classCode, true, nil
}

func (db *DB) HasClassAccessRole(classCode, roleID string) (bool, error) {
	query := `
		SELECT 1
		FROM class_access_roles
		WHERE class_code = ? AND role_id = ?;
	`

	var exists int
	err := db.conn.QueryRow(query, classCode, roleID).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, dataerr.NewQueryError(
			fmt.Sprintf("could not check if role %s is linked to class %s", roleID, classCode),
			err,
		)
	}

	return true, nil
}

func (db *DB) GetClassroomsByClassCode(classCode string) ([]Classroom, error) {
	query := `
		SELECT room_id, class_code
		FROM classrooms
		WHERE class_code = ?
		ORDER BY room_id;
	`

	rows, err := db.conn.Query(query, classCode)
	if err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("could not get classrooms for class %s", classCode), err)
	}
	defer rows.Close()

	classrooms := make([]Classroom, 0)
	for rows.Next() {
		var classroom Classroom
		if err := rows.Scan(&classroom.RoomID, &classroom.ClassCode); err != nil {
			return nil, dataerr.NewQueryError(fmt.Sprintf("could not scan classroom for class %s", classCode), err)
		}
		classrooms = append(classrooms, classroom)
	}

	if err := rows.Err(); err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("error iterating classrooms for class %s", classCode), err)
	}

	return classrooms, nil
}

func (db *DB) GetClassAccessRolesByClassCode(classCode string) ([]ClassAccessRole, error) {
	query := `
		SELECT class_code, role_id
		FROM class_access_roles
		WHERE class_code = ?
		ORDER BY role_id;
	`

	rows, err := db.conn.Query(query, classCode)
	if err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("could not get class access roles for class %s", classCode), err)
	}
	defer rows.Close()

	roles := make([]ClassAccessRole, 0)
	for rows.Next() {
		var classRole ClassAccessRole
		if err := rows.Scan(&classRole.ClassCode, &classRole.RoleID); err != nil {
			return nil, dataerr.NewQueryError(fmt.Sprintf("could not scan class access role for class %s", classCode), err)
		}
		roles = append(roles, classRole)
	}

	if err := rows.Err(); err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("error iterating class access roles for class %s", classCode), err)
	}

	return roles, nil
}

func (db *DB) GetUserByUserID(userID string) (User, error) {
	query := `
		SELECT user_id, display_name
		FROM users
		WHERE user_id = ?
	`

	var user User
	err := db.conn.QueryRow(query, userID).Scan(&user.UserID, &user.DisplayName)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, dataerr.NewNotFoundError(fmt.Sprintf("no user found with id %s", userID), err)
		}
		return User{}, dataerr.NewQueryError(fmt.Sprintf("could not get user for user id %s", userID), err)
	}

	return user, nil
}

func (db *DB) GetClassByClassCode(classCode string) (Class, error) {
	query := `SELECT id, class_code, name FROM classes WHERE class_code = ?`

	var class Class
	err := db.conn.QueryRow(query, classCode).Scan(&class.ID, &class.ClassCode, &class.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Class{}, dataerr.NewNotFoundError(fmt.Sprintf("class with code %s not found", classCode), err)
		}
		return Class{}, dataerr.NewQueryError(fmt.Sprintf("could not get class ID from class code %s", classCode), err)
	}
	return class, nil
}

func (db *DB) GetClassesByClassCodes(classCodes []string) ([]Class, error) {
	if len(classCodes) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(classCodes)), ",")
	args := make([]any, len(classCodes))
	for index, classCode := range classCodes {
		args[index] = classCode
	}

	query := "SELECT id, class_code, name FROM classes WHERE class_code IN (" + placeholders + ")"
	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, dataerr.NewQueryError("could not get classes by class codes", err)
	}
	defer rows.Close()

	var classes []Class
	for rows.Next() {
		var class Class
		if err := rows.Scan(&class.ID, &class.ClassCode, &class.Name); err != nil {
			return nil, dataerr.NewQueryError("could not scan class by class codes", err)
		}
		classes = append(classes, class)
	}

	if err := rows.Err(); err != nil {
		return nil, dataerr.NewQueryError("could not iterate classes by class codes", err)
	}
	return classes, nil
}

func (db *DB) GetClassByClassID(classID int64) (Class, error) {
	query := `SELECT id, class_code, name FROM classes WHERE id = ?`

	var class Class
	err := db.conn.QueryRow(query, classID).Scan(&class.ID, &class.ClassCode, &class.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Class{}, dataerr.NewNotFoundError(fmt.Sprintf("class with ID %d not found", classID), err)
		}
		return Class{}, dataerr.NewQueryError(fmt.Sprintf("could not get class with ID %d", classID), err)
	}
	return class, nil
}

func (db *DB) AddPendingConfirmation(confirmation PendingConfirmation) error {
	query := `
		INSERT INTO pending_confirmations (message_id, user_id, problem_id)
		VALUES (?, ?, ?)
		ON CONFLICT (user_id, problem_id) DO UPDATE SET message_id = EXCLUDED.message_id;
	`
	_, err := db.conn.Exec(query, confirmation.MessageID, confirmation.UserID, confirmation.ProblemID)
	if err != nil {
		sqliteErr, ok := errors.AsType[sqlite3.Error](err)
		if ok && sqliteErr.ExtendedCode == sqlite3.ErrConstraintForeignKey {
			return dataerr.NewForeignKeyViolationError(
				fmt.Sprintf("could not add pending confirmation: user %s or problem %d is missing", confirmation.UserID, confirmation.ProblemID),
				err,
			)
		}
		return dataerr.NewQueryError(fmt.Sprintf("could not add or update pending confirmation for message id %s, user id %s and problem %d", confirmation.MessageID, confirmation.UserID, confirmation.ProblemID), err)
	}

	return nil
}

func (db *DB) ExistsPendingConfirmation(messageID string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM pending_confirmations
		WHERE message_id = ?;
	`
	var count int

	err := db.conn.QueryRow(query, messageID).Scan(&count)
	if err != nil {
		return false, dataerr.NewQueryError(fmt.Sprintf("could not check pending confirmation existence for message id %s", messageID), err)
	}
	return count > 0, nil
}

func (db *DB) GetPendingConfirmation(messageID string) (string, int64, error) {
	query := `
		SELECT user_id, problem_id FROM pending_confirmations
		WHERE message_id = ?;
	`
	var userID string
	var problemID int64

	err := db.conn.QueryRow(query, messageID).Scan(&userID, &problemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", -1, dataerr.NewNotFoundError(fmt.Sprintf("no pending confirmation with message id %s", messageID), err)
		}
		return "", 0, dataerr.NewQueryError(fmt.Sprintf("could not get pending confirmation from message id %s", messageID), err)
	}
	return userID, problemID, nil
}

func (db *DB) RemovePendingConfirmation(messageID string) error {
	query := `
		DELETE FROM pending_confirmations WHERE message_id = ?;
	`
	_, err := db.conn.Exec(query, messageID)
	if err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not remove pending confirmation for message id %s", messageID), err)
	}
	return nil
}

func (db *DB) GetProblemByID(problemID int64) (Problem, error) {
	query := `
		SELECT name, url, class_id, difficulty, topic, priority, created_at
		FROM problems
		WHERE id = ?;
	`
	var problem Problem

	err := db.conn.QueryRow(query, problemID).Scan(&problem.Name, &problem.URL, &problem.ClassID, &problem.Difficulty, &problem.Topic, &problem.Priority, &problem.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Problem{}, dataerr.NewNotFoundError(fmt.Sprintf("ProblemID %d not found", problemID), err)
		}
		return Problem{}, dataerr.NewQueryError(fmt.Sprintf("could not get problem info for problem id %d", problemID), err)
	}

	problem.ID = problemID
	return problem, nil
}

func (db *DB) GetProblemsByIDs(problemIDs []int64) ([]Problem, error) {
	if len(problemIDs) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(problemIDs)), ",")
	args := make([]any, len(problemIDs))
	for index, problemID := range problemIDs {
		args[index] = problemID
	}

	query := "SELECT id, name, url, class_id, difficulty, topic, priority, created_at FROM problems WHERE id IN (" + placeholders + ")"
	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, dataerr.NewQueryError("could not get problems by IDs", err)
	}
	defer rows.Close()

	var problems []Problem
	for rows.Next() {
		var problem Problem
		if err := rows.Scan(
			&problem.ID,
			&problem.Name,
			&problem.URL,
			&problem.ClassID,
			&problem.Difficulty,
			&problem.Topic,
			&problem.Priority,
			&problem.CreatedAt,
		); err != nil {
			return nil, dataerr.NewQueryError("could not scan problem by IDs", err)
		}
		problems = append(problems, problem)
	}
	if err := rows.Err(); err != nil {
		return nil, dataerr.NewQueryError("could not iterate problems by IDs", err)
	}
	return problems, nil
}

func (db *DB) GetActiveUsersByClassID(classID int64) ([]User, error) {
	query := `
		SELECT DISTINCT
			u.user_id,
			u.display_name
		FROM solved_problems sp
		JOIN problems p ON sp.problem_id = p.id
		JOIN users u ON sp.user_id = u.user_id
		WHERE p.class_id = ?
		ORDER BY u.display_name ASC, u.user_id ASC
	`
	rows, err := db.conn.Query(query, classID)
	if err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("could not get active users from class id %d", classID), err)
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.UserID, &user.DisplayName); err != nil {
			return nil, dataerr.NewQueryError(fmt.Sprintf("could not scan user from class id %d", classID), err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("error iterating over active users from class id %d", classID), err)
	}

	return users, nil
}

func (db *DB) GetAllClasses() ([]Class, error) {
	query := `
		SELECT id, class_code, name from classes
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, dataerr.NewQueryError("could not run query to get info for all classes", err)
	}
	defer rows.Close()

	var classes []Class
	for rows.Next() {
		var classID int64
		var classCode, className string
		if err := rows.Scan(&classID, &classCode, &className); err != nil {
			return nil, dataerr.NewQueryError("could not scan info for one of the classes", err)
		}
		classes = append(classes, Class{
			ID:        classID,
			ClassCode: classCode,
			Name:      className,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, dataerr.NewQueryError("error iterating over rows to get info from all classes", err)
	}

	return classes, nil
}

func (db *DB) ReplaceProblemByProblemID(problemID int64, problem Problem) error {
	query := `
		UPDATE problems
		SET name = ?, url = ?, class_id = ?, difficulty = ?, topic = ?, priority = ?
		WHERE id = ?;
	`

	if result, err := db.conn.Exec(query, problem.Name, problem.URL, problem.ClassID, problem.Difficulty, problem.Topic, problem.Priority, problemID); err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not update problem with ID %d", problemID), err)
	} else if rowsAffected, err := result.RowsAffected(); err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not determine rows affected when updating problem with ID %d", problemID), err)
	} else if rowsAffected == 0 {
		return dataerr.NewNotFoundError(fmt.Sprintf("problem with ID %d not found", problemID), nil)
	}

	return nil
}

func (db *DB) PatchProblemByProblemID(problemID int64, patch ProblemPatch) error {
	var (
		sets []string
		args []any
	)
	if patch.Name.Set {
		sets = append(sets, "name = ?")
		args = append(args, patch.Name.Value)
	}
	if patch.URL.Set {
		sets = append(sets, "url = ?")
		args = append(args, patch.URL.Value)
	}
	if patch.ClassID.Set {
		sets = append(sets, "class_id = ?")
		args = append(args, patch.ClassID.Value)
	}
	if patch.Difficulty.Set {
		sets = append(sets, "difficulty = ?")
		args = append(args, patch.Difficulty.Value)
	}
	if patch.Topic.Set {
		sets = append(sets, "topic = ?")
		args = append(args, patch.Topic.Value)
	}
	if patch.Priority.Set {
		sets = append(sets, "priority = ?")
		args = append(args, patch.Priority.Value)
	}
	if len(sets) == 0 {
		return dataerr.NewQueryError("could not patch problem without fields", errors.New("problem patch is empty"))
	}

	args = append(args, problemID)
	query := "UPDATE problems SET " + strings.Join(sets, ", ") + " WHERE id = ?;"
	result, err := db.conn.Exec(query, args...)
	if err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not patch problem with ID %d", problemID), err)
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not determine rows affected when patching problem with ID %d", problemID), err)
	} else if rowsAffected == 0 {
		return dataerr.NewNotFoundError(fmt.Sprintf("problem with ID %d not found", problemID), nil)
	}
	return nil
}

func (db *DB) HasUserSolvedProblem(userID string, problemID int64) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM solved_problems
			WHERE user_id = ? AND problem_id = ?
		)
	`

	var exists bool
	if err := db.conn.QueryRow(query, userID, problemID).Scan(&exists); err != nil {
		return false, dataerr.NewQueryError(fmt.Sprintf("could not check if user %s has solved problem %d", userID, problemID), err)
	}

	return exists, nil
}

func (db *DB) ExterminateUser(userID string) error {
	query := "DELETE FROM users WHERE user_id = ?"
	if _, err := db.conn.Exec(query, userID); err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not exterminate user %s", userID), err)
	}

	return nil
}

func (db *DB) GetClassesWhereUserIsActive(userID string) ([]Class, error) {
	query := `
		SELECT DISTINCT c.id, c.class_code, c.name
		FROM solved_problems sp
		JOIN problems p ON sp.problem_id = p.id
		JOIN classes c ON p.class_id = c.id
		WHERE sp.user_id = ?
		ORDER BY c.class_code ASC;
	`

	rows, err := db.conn.Query(query, userID)
	if err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("could not get classes where user %s is active", userID), err)
	}
	defer rows.Close()

	var classes []Class
	for rows.Next() {
		var class Class
		if err := rows.Scan(&class.ID, &class.ClassCode, &class.Name); err != nil {
			return nil, dataerr.NewQueryError(fmt.Sprintf("could not scan class for active user %s", userID), err)
		}
		classes = append(classes, class)
	}

	if err := rows.Err(); err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("error iterating over classes for active user %s", userID), err)
	}

	return classes, nil
}

func (db *DB) GetClassesByAccessRoles(roleIDs []string) ([]Class, error) {
	if len(roleIDs) == 0 {
		return []Class{}, nil
	}

	// Build placeholders for IN clause idiomatic way
	placeholdersArr := make([]string, len(roleIDs))
	args := make([]any, len(roleIDs))
	for i, id := range roleIDs {
		placeholdersArr[i] = "?"
		args[i] = id
	}
	placeholders := strings.Join(placeholdersArr, ",")

	query := fmt.Sprintf(`
		SELECT DISTINCT c.id, c.class_code, c.name
		FROM classes c
		JOIN class_access_roles car ON c.class_code = car.class_code
		WHERE car.role_id IN (%s)
		ORDER BY c.class_code ASC
	`, placeholders)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, dataerr.NewQueryError("could not get classes by access roles", err)
	}
	defer rows.Close()

	var classes []Class
	for rows.Next() {
		var class Class
		if err := rows.Scan(&class.ID, &class.ClassCode, &class.Name); err != nil {
			return nil, dataerr.NewQueryError("could not scan class from access roles", err)
		}
		classes = append(classes, class)
	}

	if err := rows.Err(); err != nil {
		return nil, dataerr.NewQueryError("error iterating over classes from access roles", err)
	}

	return classes, nil
}

// GetACValidationData reads the validation data for every requested pair in one query.
// Missing targets remain in the result with nil problem or class pointers.
func (db *DB) GetACValidationData(targets []ACKey) (map[ACKey]ACValidationData, error) {
	validationDataByAC := make(map[ACKey]ACValidationData)
	if len(targets) == 0 {
		return validationDataByAC, nil
	}

	pairPlaceholders := strings.TrimSuffix(strings.Repeat("(?, ?),", len(targets)), ",")
	queryArgs := make([]any, 0, 2*len(targets))
	for _, target := range targets {
		queryArgs = append(queryArgs, target.UserID, target.ProblemID)
	}

	query := `
		WITH requested(user_id, problem_id) AS (VALUES ` + pairPlaceholders + `)
		SELECT DISTINCT
			requested.user_id, requested.problem_id,
			p.id, p.name, p.url, p.class_id, p.difficulty, p.topic, p.priority, p.created_at,
			c.id, c.class_code, c.name,
			ar.role_id, sp.problem_id IS NOT NULL
		FROM requested
		LEFT JOIN problems p ON p.id = requested.problem_id
		LEFT JOIN classes c ON c.id = p.class_id
		LEFT JOIN class_access_roles ar ON ar.class_code = c.class_code
		LEFT JOIN solved_problems sp
			ON sp.user_id = requested.user_id AND sp.problem_id = requested.problem_id
	`

	rows, err := db.conn.Query(query, queryArgs...)
	if err != nil {
		return nil, dataerr.NewQueryError("could not read AC validation data", err)
	}

	defer rows.Close()
	for rows.Next() {
		var target ACKey
		var problem Problem
		var problemID, problemClassID, classID sql.NullInt64
		var problemName, problemURL, classCode, className, roleID sql.NullString
		var createdAt sql.NullTime
		var alreadyRecorded bool

		if err := rows.Scan(
			&target.UserID, &target.ProblemID,
			&problemID, &problemName, &problemURL, &problemClassID,
			&problem.Difficulty, &problem.Topic, &problem.Priority, &createdAt,
			&classID, &classCode, &className,
			&roleID, &alreadyRecorded,
		); err != nil {
			return nil, dataerr.NewQueryError("could not scan AC validation data", err)
		}

		// Join rows repeat problem/class values for each role; assemble each pair once.
		validationData, found := validationDataByAC[target]
		if !found {
			if problemID.Valid {
				problem.ID, problem.ClassID = problemID.Int64, problemClassID.Int64
				problem.Name, problem.URL, problem.CreatedAt = problemName.String, problemURL.String, createdAt.Time
				validationData.Problem = &problem
			}
			if classID.Valid {
				validationData.Class = &Class{
					ID:        classID.Int64,
					ClassCode: classCode.String,
					Name:      className.String,
				}
			}
			validationData.AlreadyRecorded = alreadyRecorded
		}

		if roleID.Valid {
			validationData.AccessRoleIDs = append(validationData.AccessRoleIDs, roleID.String)
		}

		validationDataByAC[target] = validationData
	}

	if err := rows.Err(); err != nil {
		return nil, dataerr.NewQueryError("could not iterate AC validation data", err)
	}
	return validationDataByAC, nil
}
