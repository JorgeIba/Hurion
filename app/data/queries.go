package data

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
)

// Binds an existing (or not) discord id to a single user name.
func (db *DB) BindUser(discordId, userName string) error {
	query := `
		INSERT INTO users (discord_id, user_name)
		VALUES (?, ?)
		ON CONFLICT(discord_id) DO UPDATE SET user_name = excluded.user_name;
	`

	if _, err := db.conn.Exec(query, discordId, userName); err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not bind user %s with user name %s", discordId, userName), err)
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
		return dataerr.NewQueryError(fmt.Sprintf("could not register the user %s into class id %d", discordID, classID), err)
	}
	return nil
}

func (db *DB) DeregisterUserFromClass(discordID string, classID int64) error {
	query := `
		DELETE FROM enrollments
		WHERE user_id = ? AND class_id = ?;
	`

	if _, err := db.conn.Exec(query, discordID, classID); err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not deregister the user %s from class id %d", discordID, classID), err)
	}
	return nil
}

func (db *DB) AddProblem(problemName, url string, classID int64) (int64, error) {
	query := `
		INSERT INTO problems (name, url, class_id)
		VALUES (?,?,?)
		ON CONFLICT(name, class_id) DO NOTHING;
	`
	response, err := db.conn.Exec(query, problemName, url, classID)
	if err != nil {
		return 0, dataerr.NewQueryError(fmt.Sprintf("could not add the problem %s into the class id %d", problemName, classID), err)
	}

	problemID, err := response.LastInsertId()
	if err != nil {
		return 0, dataerr.NewQueryError("could not fetch last insert id", err)
	}

	if rowsAffected, err := response.RowsAffected(); err != nil {
		return 0, dataerr.NewQueryError(fmt.Sprintf("could not determine rows affected when inserting problem %s into class %d", problemName, classID), err)
	} else if rowsAffected == 0 {
		return problemID, dataerr.NewRowConflictError(fmt.Sprintf("Problem %s already exists on class %d with ID %d", problemName, classID, problemID), nil)
	}

	return problemID, nil
}

func (db *DB) RemoveProblemByNameAndClassID(problemName string, classID int64) (int64, error) {
	query := `
		DELETE FROM problems
		WHERE name = ? AND class_id = ?
	`
	response, err := db.conn.Exec(query, problemName, classID)
	if err != nil {
		return 0, dataerr.NewQueryError(fmt.Sprintf("could not remove the problem %s from the class id %d", problemName, classID), err)
	}

	return response.RowsAffected()
}

func (db *DB) RemoveProblemByID(problemID int64) (int64, error) {
	query := `
		DELETE FROM problems
		WHERE id = ?
	`
	response, err := db.conn.Exec(query, problemID)
	if err != nil {
		return 0, dataerr.NewQueryError(fmt.Sprintf("could not remove %d the problem", problemID), err)
	}

	return response.RowsAffected()
}

func (db *DB) AddAC(discordID string, problemID int64) error {
	query := `
		INSERT INTO solved_problems (user_id, problem_id, solved_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, problem_id) DO NOTHING;
	`
	_, err := db.conn.Exec(query, discordID, problemID)
	if err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not AC the problem for the user %s in the class %d", discordID, problemID), err)
	}

	return nil
}

func (db *DB) RemoveAC(discordID string, problemID int64) error {
	query := `
		DELETE FROM solved_problems
		WHERE user_id = ? AND problem_id = ?
	`
	_, err := db.conn.Exec(query, discordID, problemID)
	if err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not remove the AC of problem for the user %s from the class %d", discordID, problemID), err)
	}

	return nil
}

func (db *DB) GetProblemsByClass(classID int64) ([]Problem, error) {
	query := `
		SELECT id, name, url
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
		var problemID int64
		var problemName string
		var url string
		if err := rows.Scan(&problemID, &problemName, &url); err != nil {
			return nil, dataerr.NewQueryError(fmt.Sprintf("could not scan problem name when fetching problems from class id %d", classID), err)
		}
		problems = append(problems, Problem{
			ID:      problemID,
			Name:    problemName,
			ClassID: classID,
			URL:     url,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("error during rows iteration when fetching problems from class id %d", classID), err)
	}

	return problems, nil
}

func (db *DB) GetSolvedProblemsByUserAndClass(discordID string, classID int64) ([]SolvedProblem, error) {
	query := `
		SELECT sp.problem_id, sp.solved_at
		FROM solved_problems sp
		JOIN problems p ON sp.problem_id = p.id
		WHERE sp.user_id = ? AND p.class_id = ?
	`

	rows, err := db.conn.Query(query, discordID, classID)
	if err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("could not get solved problems for user %s in class %d", discordID, classID), err)
	}
	defer rows.Close()

	solvedProblems := make([]SolvedProblem, 0)

	for rows.Next() {
		var problemID int64
		var solvedAt time.Time

		if err := rows.Scan(&problemID, &solvedAt); err != nil {
			return nil, dataerr.NewQueryError(fmt.Sprintf("could not scan row for problems by user %s in class %d", discordID, classID), err)
		}

		solvedProblems = append(solvedProblems, SolvedProblem{
			ProblemID: problemID,
			UserID:    discordID,
			SolvedAt:  solvedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("error iterating over rows for problems by user %s in class %d", discordID, classID), err)
	}

	return solvedProblems, nil
}

func (db *DB) GetAllSolvedProblemsByClass(classID int64) (map[User][]SolvedProblem, error) {
	query := `
		SELECT 
			sp.user_id,
			COALESCE(u.user_name, 'Unknown') AS user_name,
			sp.problem_id,
			sp.solved_at
		FROM solved_problems sp
		LEFT JOIN users u ON sp.user_id = u.discord_id
		JOIN problems p ON sp.problem_id = p.id
		WHERE p.class_id = ?
	`

	rows, err := db.conn.Query(query, classID)
	if err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("could not get solved problems for the class %d", classID), err)
	}
	defer rows.Close()

	solvedProblemsIDByUser := make(map[User][]SolvedProblem)

	for rows.Next() {
		var userID, userName string
		var problemID int64
		var solvedAt time.Time

		if err := rows.Scan(&userID, &userName, &problemID, &solvedAt); err != nil {
			return nil, dataerr.NewQueryError("could not scan row", err)
		}

		user := User{
			DiscordID: userID,
			UserName:  userName,
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

func (db *DB) CreateClass(classCode, className string) (int64, error) {
	query := `
		INSERT INTO classes (class_code, name) 
		VALUES (?, ?)
		ON CONFLICT(class_code) DO NOTHING;
	`

	res, err := db.conn.Exec(query, classCode, className)
	if err != nil {
		return 0, dataerr.NewQueryError(fmt.Sprintf("could not create class %s with name %s", classCode, className), err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, dataerr.NewQueryError("could not check affected rows", err)
	}

	if rowsAffected == 0 {
		return 0, dataerr.NewRowConflictError(fmt.Sprintf("class with code %s already exists", classCode), nil)
	}

	classID, err := res.LastInsertId()
	if err != nil {
		return 0, dataerr.NewQueryError("could not get last insert ID", err)
	}

	return classID, nil
}

func (db *DB) GetDiscordIDByUserName(userName string) (string, error) {
	query := `SELECT discord_id FROM users WHERE user_name = ?`

	var discordID string
	err := db.conn.QueryRow(query, userName).Scan(&discordID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", dataerr.NewNotFoundError(fmt.Sprintf("no user found with name: %s", userName), nil)
		}
		return "", dataerr.NewQueryError(fmt.Sprintf("could not get discord ID from user %s", userName), err)
	}

	return discordID, nil
}

func (db *DB) GetClassIDByClassCode(classCode string) (int64, error) {
	query := `SELECT id FROM classes WHERE class_code = ?`

	var classID int64
	err := db.conn.QueryRow(query, classCode).Scan(&classID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, dataerr.NewNotFoundError(fmt.Sprintf("no class found with code: %s", classCode), nil)
		}
		return 0, dataerr.NewQueryError(fmt.Sprintf("could not get class ID from class code %s", classCode), err)
	}

	return classID, nil
}

func (db *DB) GetClassByClassID(classID int64) (Class, error) {
	query := `SELECT class_code, name FROM classes WHERE id = ?`

	var classCode, className string

	err := db.conn.QueryRow(query, classID).Scan(&classCode, &className)
	if err != nil {
		if err == sql.ErrNoRows {
			return Class{}, dataerr.NewNotFoundError(fmt.Sprintf("no class found with id: %d", classID), nil)
		}
		return Class{}, dataerr.NewQueryError(fmt.Sprintf("could not get class from class id %d", classID), err)
	}

	return Class{ID: classID, ClassCode: classCode, Name: className}, nil
}

func (db *DB) GetClassByClassCode(classCode string) (Class, error) {
	query := `SELECT id, name FROM classes WHERE class_code = ?`

	var classID int64
	var className string

	err := db.conn.QueryRow(query, classCode).Scan(&classID, &className)
	if err != nil {
		if err == sql.ErrNoRows {
			return Class{}, dataerr.NewNotFoundError(fmt.Sprintf("no class found with code: %s", classCode), nil)
		}
		return Class{}, dataerr.NewQueryError(fmt.Sprintf("could not get class ID from class code %s", classCode), err)
	}

	return Class{ID: classID, ClassCode: classCode, Name: className}, nil
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

func (db *DB) IsUserBinded(userID string) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM users
		WHERE discord_id = ?;
	`

	var count int
	err := db.conn.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return false, dataerr.NewQueryError(fmt.Sprintf("error checking user binding for user %s", userID), err)
	}

	return count > 0, nil
}

func (db *DB) AddPendingConfirmation(messageID, userID string, problemID int64) error {
	query := `
		INSERT INTO pending_confirmations (message_id, user_id, problem_id)
		VALUES (?, ?, ?)
		ON CONFLICT (user_id, problem_id) DO UPDATE SET message_id = EXCLUDED.message_id;
	`
	_, err := db.conn.Exec(query, messageID, userID, problemID)
	if err != nil {
		return dataerr.NewQueryError(fmt.Sprintf("could not add or update pending confirmation for message id %s, user id %s and problem %d", messageID, userID, problemID), err)
	}
	return nil
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
			return "", -1, dataerr.NewNotFoundError(fmt.Sprintf("not pending confimation with message id %s", messageID), nil)
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

func (db *DB) GetUsernameByDiscordID(discordID string) (string, error) {
	query := `
		SELECT user_name FROM users
		WHERE discord_id = ?;
	`
	var userName string

	err := db.conn.QueryRow(query, discordID).Scan(&userName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", dataerr.NewNotFoundError(fmt.Sprintf("discordId %s has not any username binded", discordID), nil)
		}
		return "", dataerr.NewQueryError(fmt.Sprintf("could not get user_name from discord id %s", discordID), err)
	}
	return userName, nil
}

func (db *DB) GetProblemByID(problemID int64) (Problem, error) {
	query := `
	SELECT name, url, class_id FROM problems
	WHERE id = ?;
	`
	var name string
	var url string
	var class_id int64

	err := db.conn.QueryRow(query, problemID).Scan(&name, &url, &class_id)
	if err != nil {
		if err == sql.ErrNoRows {
			return Problem{}, dataerr.NewNotFoundError(fmt.Sprintf("ProblemID %d not found", problemID), nil)
		}
		return Problem{}, dataerr.NewQueryError(fmt.Sprintf("could not get problem info for problem id %d", problemID), err)
	}
	return Problem{ID: problemID, URL: url, Name: name, ClassID: class_id}, nil
}

func (db *DB) GetAllUsersByClassID(classID int64) ([]User, error) {
	query := `
		SELECT 
			u.discord_id,
			COALESCE(u.user_name, 'Unknown') AS user_name
		FROM enrollments e
		JOIN users u ON e.user_id = u.discord_id
		WHERE e.class_id = ?
	`

	rows, err := db.conn.Query(query, classID)
	if err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("could not get users for class %d", classID), err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.DiscordID, &user.UserName); err != nil {
			return nil, dataerr.NewQueryError(fmt.Sprintf("could not scan user row for class %d", classID), err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, dataerr.NewQueryError(fmt.Sprintf("error iterating over user rows for class %d", classID), err)
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

func (db *DB) SetURLByProblemID(problemID int64, url string) error {
	query := `
		UPDATE problems
		SET url = ?
		where id = ?
	`
	_, err := db.conn.Exec(query, url, problemID)
	if err != nil {
		return dataerr.NewQueryError("Could not set the url %s for problem %d", err)
	}

	return nil

}
