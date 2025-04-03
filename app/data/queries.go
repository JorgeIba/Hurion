package data

import (
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

func (db *DB) AddProblem(problemName string, classId int64) (int64, error) {
	query := `
		INSERT INTO problems (name, class_id)
		VALUES (?,?)
		ON CONFLICT(name, class_id) DO NOTHING;
	`
	response, err := db.conn.Exec(query, problemName, classId)
	if err != nil {
		return 0, fmt.Errorf("could not add the problem into the class: %v", err)
	}

	return response.LastInsertId()
}

func (db *DB) RemoveProblem(problemName string, classId int64) (int64, error) {
	query := `
		DELETE FROM problems
		WHERE name = ? AND class_id = ?
	`
	response, err := db.conn.Exec(query, problemName, classId)
	if err != nil {
		return 0, fmt.Errorf("could not remove the problem from the class: %v", err)
	}

	return response.RowsAffected()
}
