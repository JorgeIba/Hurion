package data

import (
	"database/sql"
	"fmt"
)

// Binds an existing (or not) discord id to a single user name.
func (db *DB) BindUser(discordId, userName string) error {
	query := `
		INSERT INTO users (discord_id, user_name)
		VALUES (?, ?)
		ON CONFLICT(discord_id) DO UPDATE SET user_name = excluded.user_name;
	`

	if _, err := db.conn.Exec(query, discordId, userName); err != nil {
		return fmt.Errorf("could not bind user: %v", err)
	}

	return nil
}

func (db *DB) RegisterUserIntoClass(discordID string, classID int64) error {
	query := `
		INSERT INTO enrollments (user_id, class_id)
		VALUES (?,?)
		ON CONFLICT(user_id, class_id) DO NOTHING;
	`

	if _, err := db.conn.Exec(query, discordID, classID); err != nil {
		return fmt.Errorf("could not register the user into class: %v", err)
	}
	return nil
}

func (db *DB) DeregisterUserFromClass(discordID string, classID int64) error {
	query := `
		DELETE FROM enrollments
		WHERE user_id = ? AND class_id = ?;
	`

	if _, err := db.conn.Exec(query, discordID, classID); err != nil {
		return fmt.Errorf("could not deregister the user into class: %v", err)
	}
	return nil
}

func (db *DB) AddProblem(problemName string, classID int64) (int64, error) {
	query := `
		INSERT INTO problems (name, class_id)
		VALUES (?,?)
		ON CONFLICT(name, class_id) DO NOTHING;
	`
	response, err := db.conn.Exec(query, problemName, classID)
	if err != nil {
		return 0, fmt.Errorf("could not add the problem into the class: %v", err)
	}

	return response.LastInsertId()
}

func (db *DB) RemoveProblem(problemName string, classID int64) (int64, error) {
	query := `
		DELETE FROM problems
		WHERE name = ? AND class_id = ?
	`
	response, err := db.conn.Exec(query, problemName, classID)
	if err != nil {
		return 0, fmt.Errorf("could not remove the problem from the class: %v", err)
	}

	return response.RowsAffected()
}

func (db *DB) AddAC(discordID string, problemID, classID int64) error {
	query := `
		INSERT INTO solved_problems (user_id, problem_id, class_id, solved_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, problem_id, class_id) DO NOTHING;
	`
	_, err := db.conn.Exec(query, discordID, problemID, classID)
	if err != nil {
		return fmt.Errorf("could not AC the problem for the user in the class: %v", err)
	}

	return nil
}

func (db *DB) RemoveAC(discordID string, problemID, classID int64) error {
	query := `
		DELETE FROM solved_problems
		WHERE user_id = ? AND problem_id = ? AND class_id = ?
	`
	_, err := db.conn.Exec(query, discordID, problemID, classID)
	if err != nil {
		return fmt.Errorf("could not remove the AC of problem for the user in the class: %v", err)
	}

	return nil
}

func (db *DB) GetProblemsByClass(classID int64) ([]string, error) {
	query := `
		SELECT name
		FROM problems
		WHERE class_id = ?
	`

	rows, err := db.conn.Query(query, classID)
	if err != nil {
		return nil, fmt.Errorf("could not fetch problems: %v", err)
	}
	defer rows.Close()

	var problems []string
	for rows.Next() {
		var problemName string
		if err := rows.Scan(&problemName); err != nil {
			return nil, fmt.Errorf("could not scan problem name: %v", err)
		}
		problems = append(problems, problemName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %v", err)
	}

	return problems, nil
}

func (db *DB) GetAllSolvedProblemsByUser(discordID string, classID int64) ([]int64, error) {
	query := `
		SELECT problem_id
		FROM solved_problems
		WHERE user_id = ? AND class_id = ?
	`

	rows, err := db.conn.Query(query, discordID, classID)
	if err != nil {
		return nil, fmt.Errorf("could not get solved problems for user %s and class %d: %v", discordID, classID, err)
	}
	defer rows.Close()

	solvedProblems := make([]int64, 0)

	for rows.Next() {
		var problemID int64
		if err := rows.Scan(&problemID); err != nil {
			return nil, fmt.Errorf("could not scan row: %v", err)
		}
		solvedProblems = append(solvedProblems, problemID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %v", err)
	}

	return solvedProblems, nil
}

func (db *DB) GetAllSolvedProblemsByClass(classID int64) (map[string][]int64, error) {
	query := `
		SELECT user_id, problem_id
		FROM solved_problems
		WHERE class_id = ?
	`

	rows, err := db.conn.Query(query, classID)
	if err != nil {
		return nil, fmt.Errorf("could not get solved problems for the class %d: %v", classID, err)
	}
	defer rows.Close()

	solvedProblemsIDByUser := make(map[string][]int64)

	for rows.Next() {
		var userID string
		var problemID int64

		if err := rows.Scan(&userID, &problemID); err != nil {
			return nil, fmt.Errorf("could not scan row: %v", err)
		}

		solvedProblemsIDByUser[userID] = append(solvedProblemsIDByUser[userID], problemID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %v", err)
	}

	return solvedProblemsIDByUser, nil
}

func (db *DB) ClearClass(classID int) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("could not begin transaction: %v", err)
	}

	queries := []string{
		"DELETE FROM solved_problems WHERE class_id = ?",
		"DELETE FROM enrollments WHERE class_id = ?",
		"DELETE FROM problems WHERE class_id = ?",
		"DELETE FROM classes WHERE id = ?",
	}

	for _, query := range queries {
		if _, err := tx.Exec(query, classID); err != nil {
			tx.Rollback()
			return fmt.Errorf("could not execute query: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction: %v", err)
	}

	return nil
}

func (db *DB) CreateClass(classCode, className string) (int64, error) {
	query := `
		INSERT INTO classes (class_code, name) 
		VALUES (?, ?)
		ON CONFLICT(class_code) DO NOTHING;
	`

	res, err := db.conn.Exec(query, classCode, className)
	if err != nil {
		return 0, fmt.Errorf("could not create class: %v", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("could not check affected rows: %v", err)
	}

	if rowsAffected == 0 {
		return 0, fmt.Errorf("class with code %s already exists", classCode)
	}

	classID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("could not get last insert ID: %v", err)
	}

	return classID, nil
}

func (db *DB) GetDiscordIDByUserName(userName string) (string, error) {
	query := `SELECT discord_id FROM users WHERE user_name = ?`

	var discordID string
	err := db.conn.QueryRow(query, userName).Scan(&discordID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("no user found with name: %s", userName)
		}
		return "", fmt.Errorf("could not get discord ID: %v", err)
	}

	return discordID, nil
}

func (db *DB) GetClassIDByClassCode(classCode string) (int64, error) {
	query := `SELECT id FROM classes WHERE class_code = ?`

	var classID int64
	err := db.conn.QueryRow(query, classCode).Scan(&classID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("no class found with code: %s", classCode)
		}
		return 0, fmt.Errorf("could not get class ID: %v", err)
	}

	return classID, nil
}
