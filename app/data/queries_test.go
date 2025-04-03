package data

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

// Bind
func TestBindUserNew(t *testing.T) {
	db, err := newDB(Test) // Open DB in memory
	assert.NoError(t, err, "Unable to create a DB")
	defer db.closeDB()

	const discordID = "12345"
	const nameQuery = "Jorge"
	const nameExpected = "Jorge"

	err = db.BindUser(discordID, nameQuery)
	assert.NoError(t, err, "Error inserting")

	var userName string
	err = db.conn.QueryRow("SELECT user_name FROM users WHERE discord_id = ?", discordID).Scan(&userName)
	assert.NoError(t, err, "User not found")
	assert.Equal(t, nameExpected, userName, "Username doesn't match")
}

func TestBindUserUpdate(t *testing.T) {
	db, err := newDB(Test) // Open DB in memory
	assert.NoError(t, err, "Unable to create a DB")
	defer db.closeDB()

	const discordID = "12345"
	const nameFirst = "Jorge"
	const nameUpdated = "JorgeLuis"
	const nameExpected = "JorgeLuis"

	// Create a new entry
	err = db.BindUser(discordID, nameFirst)
	assert.NoError(t, err, "Error inserting")

	// Update entry
	err = db.BindUser(discordID, nameUpdated)
	assert.NoError(t, err, "Error updating")

	// Verify
	var userName string
	err = db.conn.QueryRow("SELECT user_name FROM users WHERE discord_id = ?", discordID).Scan(&userName)
	assert.NoError(t, err, "User not found")
	assert.Equal(t, nameExpected, userName, "Username doesn't match")
}