package data

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/mattn/go-sqlite3"
)

// EnsureUser ensures that a user with the given ID and display name exists in the database.
func (db *DB) EnsureUser(user User) error {
	query := `
		INSERT INTO users (user_id, display_name)
		VALUES (?, ?)
		ON CONFLICT(user_id) DO UPDATE SET display_name = excluded.display_name;
	`

	if _, err := db.conn.Exec(query, user.UserID, user.DisplayName); err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not ensure user %s", user.UserID), err)
	}

	return nil
}
func (db *DB) RegisterUserIntoClass(userID string, classID int64) error {
	query := `
		INSERT INTO enrollments (user_id, class_id)
		VALUES (?,?)
		ON CONFLICT(user_id, class_id) DO NOTHING;
	`

	if _, err := db.conn.Exec(query, userID, classID); err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not register the user %s into class id %d", userID, classID), err)
	}
	return nil
}

func (db *DB) DeregisterUserFromClass(userID string, classID int64) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return dataerr.NewTransactionError(fmt.Sprintf("could not begin transaction to deregister user %s from class %d", userID, classID), err)
	}

	// Delete in order: pending confirmations -> solved problems -> enrollment
	queries := []struct {
		query string
		note  string
	}{
		{
			query: "DELETE FROM pending_confirmations WHERE user_id = ? AND problem_id IN (SELECT id FROM problems WHERE class_id = ?)",
			note:  "pending confirmations",
		},
		{
			query: "DELETE FROM solved_problems WHERE user_id = ? AND problem_id IN (SELECT id FROM problems WHERE class_id = ?)",
			note:  "solved problems",
		},
		{
			query: "DELETE FROM enrollments WHERE user_id = ? AND class_id = ?",
			note:  "enrollment",
		},
	}

	for _, q := range queries {
		if _, err := tx.Exec(q.query, userID, classID); err != nil {
			tx.Rollback()
			return dataerr.NewQueryError(fmt.Sprintf("could not delete %s for user %s from class %d", q.note, userID, classID), err)
		}
	}

	if err := tx.Commit(); err != nil {
		return dataerr.NewTransactionError(fmt.Sprintf("could not commit transaction to deregister user %s from class %d", userID, classID), err)
	}

	return nil
}

func (db *DB) AddProblem(problem Problem) (int64, error) {
	query := `
		INSERT INTO problems (name, url, class_id, difficulty, topic)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(name, class_id) DO NOTHING;
	`
	response, err := db.conn.Exec(query, problem.Name, problem.URL, problem.ClassID, problem.Difficulty, problem.Topic)
	if err != nil {
		return 0, dataerr.NewQueryError(fmt.Sprintf("could not add the problem %s into the class id %d", problem.Name, problem.ClassID), err)
	}

	if rowsAffected, err := response.RowsAffected(); err != nil {
		return 0, dataerr.NewQueryError(fmt.Sprintf("could not determine rows affected when inserting problem %s into class %d", problem.Name, problem.ClassID), err)
	} else if rowsAffected == 0 {
		return 0, dataerr.NewRowConflictError(fmt.Sprintf("Problem %s already exists in class %d", problem.Name, problem.ClassID), nil)
	}

	problemID, err := response.LastInsertId()
	if err != nil {
		return 0, dataerr.NewQueryError("could not fetch last insert id", err)
	}

	return problemID, nil
}

func (db *DB) RemoveProblemByID(problemID int64) (int64, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return 0, dataerr.NewTransactionError(fmt.Sprintf("could not begin transaction to remove problem %d", problemID), err)
	}

	// Delete in order: pending confirmations -> solved problems -> problem itself
	queries := []string{
		"DELETE FROM pending_confirmations WHERE problem_id = ?",
		"DELETE FROM solved_problems WHERE problem_id = ?",
		"DELETE FROM problems WHERE id = ?",
	}

	for _, query := range queries {
		if _, err := tx.Exec(query, problemID); err != nil {
			tx.Rollback()
			return 0, dataerr.NewQueryError(fmt.Sprintf("could not remove problem %d", problemID), err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, dataerr.NewTransactionError(fmt.Sprintf("could not commit transaction to remove problem %d", problemID), err)
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
		SELECT id, name, url, difficulty, topic
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
		if err := rows.Scan(&problem.ID, &problem.Name, &problem.URL, &problem.Difficulty, &problem.Topic); err != nil {
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

func (db *DB) ClearClass(classID int64) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return dataerr.NewTransactionError(fmt.Sprintf("could not begin transaction to clear class %d", classID), err)
	}

	queries := []string{
		"DELETE FROM pending_confirmations WHERE problem_id IN (SELECT id FROM problems WHERE class_id = ?)",
		"DELETE FROM solved_problems WHERE problem_id IN (SELECT id FROM problems WHERE class_id = ?)",
		"DELETE FROM enrollments WHERE class_id = ?",
		"DELETE FROM problems WHERE class_id = ?",
		"DELETE FROM classes WHERE id = ?",
	}

	for _, query := range queries {
		if _, err := tx.Exec(query, classID); err != nil {
			tx.Rollback()
			return dataerr.NewQueryError(fmt.Sprintf("could not execute query to clear class %d", classID), err)
		}
	}

	if err := tx.Commit(); err != nil {
		return dataerr.NewTransactionError(fmt.Sprintf("could not commit transaction to clear class %d", classID), err)
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

func (db *DB) GetUserByUserID(userID string) (User, error) {
	query := `
		SELECT user_id, display_name
		FROM users
		WHERE user_id = ?
	`

	var user User
	err := db.conn.QueryRow(query, userID).Scan(&user.UserID, &user.DisplayName)

	if err != nil {
		if err == sql.ErrNoRows {
			return User{}, dataerr.NewNotFoundError(fmt.Sprintf("no user found with id %s", userID), nil)
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
		if err == sql.ErrNoRows {
			return Class{}, dataerr.NewNotFoundError(fmt.Sprintf("class with code %s not found", classCode), nil)
		}
		return Class{}, dataerr.NewQueryError(fmt.Sprintf("could not get class ID from class code %s", classCode), err)
	}
	return class, nil
}

func (db *DB) IsUserRegisteredInClass(userID string, classID int64) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM enrollments
		WHERE user_id = ? AND class_id = ?;
	`

	var count int
	err := db.conn.QueryRow(query, userID, classID).Scan(&count)
	if err != nil {
		return false, dataerr.NewQueryError(fmt.Sprintf("error checking user enrollment from user %s in class %d", userID, classID), err)
	}

	return count > 0, nil
}

func (db *DB) AddPendingConfirmation(confirmation PendingConfirmation) error {
	query := `
		INSERT INTO pending_confirmations (message_id, user_id, problem_id)
		VALUES (?, ?, ?)
		ON CONFLICT (user_id, problem_id) DO UPDATE SET message_id = EXCLUDED.message_id;
	`
	_, err := db.conn.Exec(query, confirmation.MessageID, confirmation.UserID, confirmation.ProblemID)
	if err != nil {
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
		if err == sql.ErrNoRows {
			return "", -1, dataerr.NewNotFoundError(fmt.Sprintf("no pending confirmation with message id %s", messageID), nil)
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
		SELECT name, url, class_id, difficulty, topic 
		FROM problems
		WHERE id = ?;
	`
	var problem Problem

	err := db.conn.QueryRow(query, problemID).Scan(&problem.Name, &problem.URL, &problem.ClassID, &problem.Difficulty, &problem.Topic)
	if err != nil {
		if err == sql.ErrNoRows {
			return Problem{}, dataerr.NewNotFoundError(fmt.Sprintf("ProblemID %d not found", problemID), nil)
		}
		return Problem{}, dataerr.NewQueryError(fmt.Sprintf("could not get problem info for problem id %d", problemID), err)
	}

	problem.ID = problemID
	return problem, nil
}

func (db *DB) GetAllUsersByClassID(classID int64) ([]User, error) {
	query := `
		SELECT 
			u.user_id,
			u.display_name
		FROM enrollments e
		JOIN users u ON e.user_id = u.user_id
		WHERE e.class_id = ?
	`
	rows, err := db.conn.Query(query, classID)
	if err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("could not get users from class id %d", classID), err)
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
		return nil, dataerr.NewQueryError(fmt.Sprintf("error iterating over users from class id %d", classID), err)
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

func (db *DB) UpdateProblemByProblemID(problemID int64, problem Problem) error {
	query := `
		UPDATE problems
		SET name = ?, url = ?, class_id = ?, difficulty = ?, topic = ?
		WHERE id = ?;
	`

	if result, err := db.conn.Exec(query, problem.Name, problem.URL, problem.ClassID, problem.Difficulty, problem.Topic, problemID); err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return dataerr.NewRowConflictError(fmt.Sprintf("problem %q already exists in class %d", problem.Name, problem.ClassID), err)
		}
		return dataerr.NewQueryError(fmt.Sprintf("could not update problem with ID %d", problemID), err)
	} else if rowsAffected, err := result.RowsAffected(); err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not determine rows affected when updating problem with ID %d", problemID), err)
	} else if rowsAffected == 0 {
		return dataerr.NewNotFoundError(fmt.Sprintf("problem with ID %d not found", problemID), nil)
	}

	return nil
}

func (db *DB) GetClassesByUserID(userID string) ([]Class, error) {
	query := `
		SELECT c.id, c.class_code, c.name
		FROM enrollments e
		JOIN classes c ON e.class_id = c.id
		WHERE e.user_id = ?;
	`

	rows, err := db.conn.Query(query, userID)
	if err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("could not get classes for user %s", userID), err)
	}
	defer rows.Close()

	var classes []Class
	for rows.Next() {
		var class Class
		if err := rows.Scan(&class.ID, &class.ClassCode, &class.Name); err != nil {
			return nil, dataerr.NewQueryError(fmt.Sprintf("could not scan class for user %s", userID), err)
		}
		classes = append(classes, class)
	}

	if err := rows.Err(); err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("error iterating over classes for user %s", userID), err)
	}

	return classes, nil
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
	tx, err := db.conn.Begin()
	if err != nil {
		return dataerr.NewTransactionError(fmt.Sprintf("could not begin transaction to exterminate user %s", userID), err)
	}

	queries := []string{
		"DELETE FROM pending_confirmations WHERE user_id = ?",
		"DELETE FROM solved_problems WHERE user_id = ?",
		"DELETE FROM enrollments WHERE user_id = ?",
		"DELETE FROM users WHERE user_id = ?",
	}

	for _, query := range queries {
		if _, err := tx.Exec(query, userID); err != nil {
			tx.Rollback()
			return dataerr.NewQueryError(fmt.Sprintf("could not execute query to exterminate user %s", userID), err)
		}
	}

	if err := tx.Commit(); err != nil {
		return dataerr.NewTransactionError(fmt.Sprintf("could not commit transaction to exterminate user %s", userID), err)
	}

	return nil
}
