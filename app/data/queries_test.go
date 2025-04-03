package data

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Bind
func TestBindUser_NewUser(t *testing.T) {
	db, err := newDB(Test)
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

func TestBindUser_UpdateUser(t *testing.T) {
	db, err := newDB(Test)
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

// Register
func TestRegisterUserIntoClass_NewUser(t *testing.T) {
	db, err := newDB(Test)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.closeDB()

	const discordID = "12345"
	const userName = "Jorge"

	// Preconditions
	response, err := db.conn.Exec("INSERT INTO classes (class_code, name) VALUES (?, ?)", "basicos", "Basicos 2025")
	assert.NoError(t, err, "Could not insert class")

	_, err = db.conn.Exec("INSERT INTO users (discord_id, user_name) VALUES (?, ?)", discordID, userName)
	assert.NoError(t, err, "Could not insert class")

	classID, _ := response.LastInsertId()

	// Register a user
	err = db.RegisterUserIntoClass(discordID, classID)
	assert.NoError(t, err, "Expected not error, but got: ")

	// Verify
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM enrollments WHERE user_id = ? AND class_id = ?", discordID, classID).Scan(&count)
	assert.NoError(t, err, "Could not query enrollments correctly")

	assert.Equal(t, 1, count, "User should be registered in the class")
}

func TestRegisterUserIntoClass_ExistingUser(t *testing.T) {
	db, err := newDB(Test)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.closeDB()

	const discordID = "12345"
	const userName = "Jorge"

	// Preconditions
	response, err := db.conn.Exec("INSERT INTO classes (class_code, name) VALUES (?, ?)", "basicos", "Basicos 2025")
	assert.NoError(t, err, "Could not insert class")

	_, err = db.conn.Exec("INSERT INTO users (discord_id, user_name) VALUES (?, ?)", discordID, userName)
	assert.NoError(t, err, "Could not insert class")

	classID, _ := response.LastInsertId()

	// Register a user
	err = db.RegisterUserIntoClass(discordID, classID)
	assert.NoError(t, err, "Expected no error, but got: ")

	// Try to register it again
	err = db.RegisterUserIntoClass(discordID, classID)
	assert.NoError(t, err, "Expected no error, but got: ")

	// Verify
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM enrollments WHERE user_id = ? AND class_id = ?", discordID, classID).Scan(&count)
	assert.NoError(t, err, "Could not query enrollments correctly: ")

	assert.Equal(t, 1, count, "User should be registered in the class")
}

// Deregister
func TestDeregisterUserIntoClass_ExistingUser(t *testing.T) {
	db, err := newDB(Test)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.closeDB()

	const discordID = "12345"
	const classID = 1
	queryEnrollment := fmt.Sprintf("SELECT COUNT(*) FROM enrollments WHERE user_id = %s AND class_id = %d", discordID, classID)

	// Register a user
	err = db.RegisterUserIntoClass(discordID, classID)
	assert.NoError(t, err, "Expected no error, but got: ")

	// Verify
	var count int
	err = db.conn.QueryRow(queryEnrollment).Scan(&count);
	assert.NoError(t, err, "Could not query enrollments correctly: ")
	assert.Equal(t, 1, count, "User should be registered")

	// Deregister
	err = db.DeregisterUserFromClass(discordID, classID)
	assert.NoError(t, err, "Expected no error, but got: ")

	// Verify
	err = db.conn.QueryRow(queryEnrollment).Scan(&count);
	assert.NoError(t, err, "Could not query enrollments correctly: ")

	assert.Equal(t, 0, count, "User should not be registered in the class")
}

// Add problem
func TestAddProblem_New(t *testing.T) {
	db, err := newDB(Test)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.closeDB()

	// Precondition
	classResult, err := db.conn.Exec(`INSERT INTO classes (class_code, name) VALUES ('intermedios', 'Intermedios 2024')`)
	assert.NoError(t, err)
	classID, err := classResult.LastInsertId()
	assert.NoError(t, err)

	// Add a problem
	problemID, err := db.AddProblem("Sorting Algorithms", classID)
	assert.NoError(t, err)
	assert.Greater(t, problemID, int64(0), "Problem ID should be greater than 0")
}

// Remove problem
func TestRemoveProblem(t *testing.T) {
	db, err := newDB(Test)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.closeDB()

	const problemName = "Sorting Algorithms"

	// Precondition
	classResult, err := db.conn.Exec(`INSERT INTO classes (class_code, name) VALUES ('intermedios', 'Intermedios 2024')`)
	assert.NoError(t, err)
	classID, err := classResult.LastInsertId()
	assert.NoError(t, err)

	// Add a problem
	problemID, err := db.AddProblem(problemName, classID)
	assert.NoError(t, err)
	assert.Greater(t, problemID, int64(0), "Problem ID should be greater than 0")

	// Remove the problem
	rowsAffected, err := db.RemoveProblem(problemName, classID)
	assert.NoError(t, err)

	assert.Equal(t, int64(1), rowsAffected, "One row should have been removed")
}