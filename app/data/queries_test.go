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
	queryEnrollment := fmt.Sprintf("SELECT COUNT(*) FROM enrollments WHERE user_id = '%s' AND class_id = %d", discordID, classID)

	// Register a user
	err = db.RegisterUserIntoClass(discordID, classID)
	assert.NoError(t, err, "Expected no error, but got: ")

	// Verify
	var count int
	err = db.conn.QueryRow(queryEnrollment).Scan(&count)
	assert.NoError(t, err, "Could not query enrollments correctly: ")
	assert.Equal(t, 1, count, "User should be registered")

	// Deregister
	err = db.DeregisterUserFromClass(discordID, classID)
	assert.NoError(t, err, "Expected no error, but got: ")

	// Verify
	err = db.conn.QueryRow(queryEnrollment).Scan(&count)
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

func TestAddAC(t *testing.T) {
	db, err := newDB(Test)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.closeDB()

	const discordID = "user123"
	const problemID = int64(1)
	const classID = int64(1)

	// Verify
	err = db.AddAC(discordID, problemID, classID)
	assert.NoError(t, err, "Problem trying to ACing the problem for the user")

	err = db.AddAC(discordID, problemID, classID)
	assert.NoError(t, err, "Re-adding the same AC should not fail")

	var count int64
	err = db.conn.QueryRow(`SELECT COUNT(*) FROM solved_problems WHERE user_id = ? AND class_id = ?;`, discordID, classID).Scan(&count)
	assert.NoError(t, err, "Cannot check the count for problems solved")

	assert.Equal(t, int64(1), count, "A single problem should have been solved")
}

func TestRemoveAC(t *testing.T) {
	db, err := newDB(Test)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.closeDB()

	const discordID = "user123"
	const problemID = int64(1)
	const classID = int64(1)
	queryCount := fmt.Sprintf("SELECT COUNT(*) FROM solved_problems WHERE user_id = '%s' AND class_id = %d;", discordID, classID)

	// Preconditions
	err = db.AddAC(discordID, problemID, classID)
	assert.NoError(t, err, "Precondition failed, problem could not be checked as solved")

	var count int64
	err = db.conn.QueryRow(queryCount).Scan(&count)
	assert.NoError(t, err, "Cannot check the count for problems solved")
	assert.Equal(t, int64(1), count, "A problem should have been added to solved list")

	// Verify
	err = db.RemoveAC(discordID, problemID, classID)
	assert.NoError(t, err, "Problem trying to remove the AC of the problem for the user")
	err = db.conn.QueryRow(queryCount).Scan(&count)
	assert.NoError(t, err, "Cannot check the count for problems solved")
	assert.Equal(t, int64(0), count, "Solved problem should been removed")
}
