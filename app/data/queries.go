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