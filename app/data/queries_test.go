package data

import (
	"testing"
	"github.com/stretchr/testify/assert"
)


func TestBindUserNew(t *testing.T) {
	db, err := newDB(Test) // Open DB in memory
	assert.NoError(t, err, "Unable to create a DB")
	defer db.closeDB()

	const nameQuery = "Jorge"
	const nameExpected = "Jorge"

	err = db.BindUser("12345", nameQuery)
	assert.NoError(t, err, "Error inserting")

	var userName string
	err = db.conn.QueryRow("SELECT user_name FROM users WHERE discord_id = ?", "12345").Scan(&userName)
	assert.NoError(t, err, "User not found")
	assert.Equal(t, nameExpected, userName, "Username doesn't match")
}

func TestBindUserUpdate(t *testing.T) {
	db, err := newDB(Test) // Open DB in memory
	assert.NoError(t, err, "Unable to create a DB")
	defer db.closeDB()

	const nameFirst = "Jorge"
	const nameUpdated = "JorgeLuis"
	const nameExpected = "JorgeLuis"

	// Create a new entry
	err = db.BindUser("12345", nameFirst)
	assert.NoError(t, err, "Error inserting")

	// Update entry
	err = db.BindUser("12345", nameUpdated)
	assert.NoError(t, err, "Error updating")

	// Verify
	var userName string
	err = db.conn.QueryRow("SELECT user_name FROM users WHERE discord_id = ?", "12345").Scan(&userName)
	assert.NoError(t, err, "User not found")
	assert.Equal(t, nameExpected, userName, "Username doesn't match")
}