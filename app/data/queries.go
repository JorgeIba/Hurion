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