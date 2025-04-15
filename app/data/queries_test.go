package data

import (
	"fmt"
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/dataerr"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

var testConfig *DBConfig = &DBConfig{DBPath: ":memory:", SchemaPath: "./schema.sql", DBType: Test}

// Bind
func TestBindUser_NewUser(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

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
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

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
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

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
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

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
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

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
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

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
func RemoveProblemByNameAndClassID(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

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
	rowsAffected, err := db.RemoveProblemByNameAndClassID(problemName, classID)
	assert.NoError(t, err)

	assert.Equal(t, int64(1), rowsAffected, "One row should have been removed")
}

func TestRemoveProblemByID(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

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
	rowsAffected, err := db.RemoveProblemByID(problemID)
	assert.NoError(t, err)

	assert.Equal(t, int64(1), rowsAffected, "One row should have been removed")
}

func TestAddAC(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const discordID = "user123"
	const classID = int64(1)

	// Preconditions
	problemID, err := db.AddProblem("Sorting", classID)
	assert.NoError(t, err, "Problem trying to adding the problem")

	// Verify
	err = db.AddAC(discordID, problemID)
	assert.NoError(t, err, "Problem trying to ACing the problem for the user")

	err = db.AddAC(discordID, problemID)
	assert.NoError(t, err, "Re-adding the same AC should not fail")

	var count int64
	err = db.conn.QueryRow(`
		SELECT COUNT(*)
		FROM solved_problems sp
		JOIN problems p ON sp.problem_id = p.id
		WHERE sp.user_id = ? AND p.class_id = ?;
	`, discordID, classID).Scan(&count)
	assert.NoError(t, err, "Cannot check the count for problems solved")

	assert.Equal(t, int64(1), count, "A single problem should have been solved")
}

func TestRemoveAC(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const discordID = "user123"
	const classID = int64(1)
	queryCount := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM solved_problems sp 
		JOIN problems p ON sp.problem_id = p.id 
		WHERE sp.user_id = '%s' AND p.class_id = %d;
	`, discordID, classID)

	// Preconditions
	problemID, err := db.AddProblem("Sorting", classID)
	assert.NoError(t, err, "Precondition failed, problem could not be added")

	err = db.AddAC(discordID, problemID)
	assert.NoError(t, err, "Precondition failed, problem could not be checked as solved")

	var count int64
	err = db.conn.QueryRow(queryCount).Scan(&count)
	assert.NoError(t, err, "Cannot check the count for problems solved")
	assert.Equal(t, int64(1), count, "A problem should have been added to solved list")

	// Verify
	err = db.RemoveAC(discordID, problemID)
	assert.NoError(t, err, "Problem trying to remove the AC of the problem for the user")
	err = db.conn.QueryRow(queryCount).Scan(&count)
	assert.NoError(t, err, "Cannot check the count for problems solved")
	assert.Equal(t, int64(0), count, "Solved problem should been removed")
}

func TestGetProblemsByClass(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const classID = int64(1)
	const classID2 = int64(2)

	// Preconditions
	_, err = db.AddProblem("Sorting Stuff 1", classID)
	assert.NoError(t, err, "Precondition failed, problem 1 could not be added")

	_, err = db.AddProblem("Sorting Stuff 2", classID)
	assert.NoError(t, err, "Precondition failed, problem 2 could not be added")

	_, err = db.AddProblem("Sorting Stuff 3", classID2)
	assert.NoError(t, err, "Precondition failed, problem 3 could not be added")

	// Do
	problemsList1, err := db.GetProblemsByClass(classID)
	assert.NoError(t, err, "List 1 could not be retrieved")

	problemsList2, err := db.GetProblemsByClass(classID2)
	assert.NoError(t, err, "List 2 could not be retrieved")

	// Verify
	assert.Equal(t, 2, len(problemsList1), "Length are not the same")
	assert.Equal(t, 1, len(problemsList2), "Length are not the same")
}

func TestGetAllSolvedProblemsByUser(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	// Preconditions
	problemID1, err := db.AddProblem("Problem 1", 1)
	assert.NoError(t, err)

	problemID2, err := db.AddProblem("Problem 2", 1)
	assert.NoError(t, err)

	problemID3, err := db.AddProblem("Problem 3", 2)
	assert.NoError(t, err)

	problemID4, err := db.AddProblem("Problem 4", 2)
	assert.NoError(t, err)

	const discordID1 = "user1"
	const discordID2 = "user2"
	var updates = []struct {
		discordID string
		problemID int64
		classID   int64
	}{
		{discordID: discordID1, problemID: problemID1},
		{discordID: discordID1, problemID: problemID2},
		{discordID: discordID1, problemID: problemID3},
		{discordID: discordID1, problemID: problemID4},

		{discordID: discordID2, problemID: problemID2},
		{discordID: discordID2, problemID: problemID3},
	}

	for _, update := range updates {
		err = db.AddAC(update.discordID, update.problemID)
		assert.NoError(t, err, "Problem trying to ACing the problem for the user")
	}

	// Act
	listUser1Class1, err := db.GetSolvedProblemsByUserAndClass(discordID1, 1)
	assert.NoError(t, err, "Problem trying to get the list")
	listUser1Class1OnlyIDs := lo.Map(listUser1Class1, func(problem SolvedProblem, _ int) int64 {
		return problem.ProblemID
	})

	listUser1Class2, err := db.GetSolvedProblemsByUserAndClass(discordID1, 2)
	assert.NoError(t, err, "Problem trying to get the list")
	listUser1Class2OnlyIDs := lo.Map(listUser1Class2, func(problem SolvedProblem, _ int) int64 {
		return problem.ProblemID
	})

	listUser2Class1, err := db.GetSolvedProblemsByUserAndClass(discordID2, 1)
	assert.NoError(t, err, "Problem trying to get the list")
	listUser2Class1OnlyIDs := lo.Map(listUser2Class1, func(problem SolvedProblem, _ int) int64 {
		return problem.ProblemID
	})

	listUser2Class2, err := db.GetSolvedProblemsByUserAndClass(discordID2, 2)
	assert.NoError(t, err, "Problem trying to get the list")
	listUser2Class2OnlyIDs := lo.Map(listUser2Class2, func(problem SolvedProblem, _ int) int64 {
		return problem.ProblemID
	})

	// Verify
	assert.Equal(t, []int64{problemID1, problemID2}, listUser1Class1OnlyIDs, "List should have matched")
	assert.Equal(t, []int64{problemID3, problemID4}, listUser1Class2OnlyIDs, "List should have matched")

	assert.Equal(t, []int64{problemID2}, listUser2Class1OnlyIDs, "List should have matched")
	assert.Equal(t, []int64{problemID3}, listUser2Class2OnlyIDs, "List should have matched")
}

func TestGetAllSolvedProblemsByClass(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	// Preconditions
	problemID1, err := db.AddProblem("Problem 1", 1)
	assert.NoError(t, err)

	problemID2, err := db.AddProblem("Problem 2", 1)
	assert.NoError(t, err)

	problemID3, err := db.AddProblem("Problem 3", 1)
	assert.NoError(t, err)

	problemID4, err := db.AddProblem("Problem 4", 2)
	assert.NoError(t, err)

	const discordID1 = "user1"
	const discordID2 = "user2"
	var updates = []struct {
		discordID string
		problemID int64
		classID   int64
	}{
		{discordID: discordID1, problemID: problemID1},
		{discordID: discordID1, problemID: problemID2},
		{discordID: discordID1, problemID: problemID3},
		{discordID: discordID1, problemID: problemID4},

		{discordID: discordID2, problemID: problemID2},
		{discordID: discordID2, problemID: problemID3},
	}

	for _, update := range updates {
		err = db.AddAC(update.discordID, update.problemID)
		assert.NoError(t, err, "Problem trying to ACing the problem for the user")
	}

	// Act
	solvedProblemsClass1, err := db.GetAllSolvedProblemsByClass(1)
	assert.NoError(t, err, "Problem trying to get the list")

	solvedProblemsClass2, err := db.GetAllSolvedProblemsByClass(2)
	assert.NoError(t, err, "Problem trying to get the list")

	// Verify
	assert.Equal(t, 5, len(solvedProblemsClass1))
	assert.Equal(t, 1, len(solvedProblemsClass2))
}

func TestClearClass(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	// Preconditions
	_, err = db.conn.Exec("INSERT INTO classes (id, class_code, name) VALUES (?, ?, ?)", 1, "CS101", "Intro to CS")
	assert.NoError(t, err, "Could not insert test data.")
	_, err = db.conn.Exec("INSERT INTO problems (id, name, class_id) VALUES (?, ?, ?)", 1, "Problem 1", 1)
	assert.NoError(t, err, "Could not insert test data.")
	_, err = db.conn.Exec("INSERT INTO enrollments (user_id, class_id) VALUES (?, ?)", "user1", 1)
	assert.NoError(t, err, "Could not insert test data.")
	_, err = db.conn.Exec("INSERT INTO solved_problems (user_id, problem_id) VALUES (?, ?)", "user1", 1)
	assert.NoError(t, err, "Could not insert test data.")

	// Act
	err = db.ClearClass(1)
	assert.NoError(t, err, "Unexpected error")

	// Verify
	var count int
	queries := []string{
		"SELECT COUNT(*) FROM classes WHERE id = 1",
		"SELECT COUNT(*) FROM problems WHERE class_id = 1",
		"SELECT COUNT(*) FROM enrollments WHERE class_id = 1",
		"SELECT COUNT(*) FROM solved_problems sp JOIN problems p ON sp.problem_id = p.id WHERE p.class_id = 1",
	}
	for _, query := range queries {
		if err := db.conn.QueryRow(query).Scan(&count); err != nil {
			t.Fatalf("could not check data: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected 0 rows, but found %d", count)
		}
	}
}

func TestCreateClass(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	classCode := "avanzados"
	className := "Avanzados 2025"

	// Act
	classID, err := db.CreateClass(classCode, className)
	assert.NoError(t, err, "Unexpected error")

	// Verify
	var dbClassID int64
	var dbClassCode, dbClassName string
	query := "SELECT id, class_code, name FROM classes WHERE id = ?"
	err = db.conn.QueryRow(query, classID).Scan(&dbClassID, &dbClassCode, &dbClassName)
	if err != nil {
		t.Fatalf("could not retrieve inserted class: %v", err)
	}

	assert.Equal(t, classID, dbClassID, "Class ID should match")
	assert.Equal(t, classCode, dbClassCode, "Class Code should match")
	assert.Equal(t, className, dbClassName, "Class Name should match")
}

func TestGetDiscordIDByUserName_Success(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const discordID = "12345"
	const userName = "Jorge"

	// Act
	err = db.BindUser(discordID, userName)
	assert.NoError(t, err)

	// Verify
	responseDiscordID, err := db.GetDiscordIDByUserName(userName)
	assert.NoError(t, err)
	assert.Equal(t, discordID, responseDiscordID)
}

func TestGetDiscordIDByUserName_NotFound(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	// Act
	_, err = db.GetDiscordIDByUserName("nonexistent")
	assert.Error(t, err)

	// Verify
	assert.Contains(t, err.Error(), "no user found")
}

func TestGetClassIDByClassCode_Success(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const classCode = "basicos"
	const className = "Basicos 2024"

	// Act
	classID, err := db.CreateClass(classCode, className)
	assert.NoError(t, err)

	// Verify
	retrievedID, err := db.GetClassIDByClassCode(classCode)
	assert.NoError(t, err)
	assert.Equal(t, classID, retrievedID)
}

func TestGetClassIDByClassCode_NotFound(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	// Act
	_, err = db.GetClassIDByClassCode("UNKNOWN")
	assert.Error(t, err)

	// Verify
	assert.Contains(t, err.Error(), "no class found")
}

func TestAddPendingConfirmation(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const messageID = "123456"
	const userID = "789"
	const problemID = int64(1)

	// Act
	err = db.AddPendingConfirmation(messageID, userID, problemID)
	assert.NoError(t, err, "Should insert pending confirmation without error")

	// Verify
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM pending_confirmations WHERE message_id = ?", messageID).Scan(&count)
	assert.NoError(t, err, "Should execute count query without error")
	assert.Equal(t, 1, count, "Should have exactly one record in pending_confirmations")
}

func TestAddPendingConfirmation_UpdateMessageID(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const messageIDPrevious = "123"
	const messageIDUpdated = "456"
	const userID = "789"
	const problemID = int64(1)
	query := fmt.Sprintf("SELECT message_id FROM pending_confirmations WHERE user_id = %s AND problem_id = %d", userID, problemID)

	// Precondition
	err = db.AddPendingConfirmation(messageIDPrevious, userID, problemID)
	assert.NoError(t, err, "Expected no error on first insertion")

	var messageID string
	err = db.conn.QueryRow(query).Scan(&messageID)
	assert.NoError(t, err, "Expected no error when retrieving the first message_id")
	assert.Equal(t, messageIDPrevious, messageID, "Expected message_id to be 'msg1'")

	// Act
	err = db.AddPendingConfirmation(messageIDUpdated, userID, problemID)
	assert.NoError(t, err, "Expected no error on update")

	// Verify
	err = db.conn.QueryRow(query).Scan(&messageID)
	assert.NoError(t, err, "Expected no error when retrieving the updated message_id")
	assert.Equal(t, messageIDUpdated, messageID, "Expected message_id to be updated to 'msg2'")
}

func TestGetPendingConfirmation_Found(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const messageID = "123456"
	const userID = "789"
	const problemID = int64(1)

	// Precondition
	err = db.AddPendingConfirmation(messageID, userID, problemID)
	assert.NoError(t, err, "Should insert test data without error")

	// Act
	gotUserID, gotProblemID, err := db.GetPendingConfirmation(messageID)

	// Vertify
	assert.NoError(t, err, "Should get pending confirmation without error")
	assert.Equal(t, userID, gotUserID, "Should return correct userID")
	assert.Equal(t, problemID, gotProblemID, "Should return correct problemID")
}

func TestGetPendingConfirmation_NotFound(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const messageID = "nonexistent"

	gotUserID, gotProblemID, err := db.GetPendingConfirmation(messageID)
	notFoundErrorType := &dataerr.NotFoundError{}

	assert.ErrorAs(t, err, &notFoundErrorType)
	assert.Empty(t, gotUserID, "Should return empty userID for missing message_id")
	assert.Equal(t, int64(-1), gotProblemID, "Should return zero problemID for missing message_id")
}

func TestRemovePendingConfirmation(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const messageID = "msg123"
	const userID = "user123"
	const problemID = int64(1)

	// Precondition
	err = db.AddPendingConfirmation(messageID, userID, problemID)
	assert.NoError(t, err, "Should insert test data without error")

	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM pending_confirmations WHERE message_id = ?", messageID).Scan(&count)
	assert.NoError(t, err, "Should execute count query without error")
	assert.Equal(t, 1, count, "Should have inserted the record from pending_confirmations")

	// Act
	err = db.RemovePendingConfirmation(messageID)
	assert.NoError(t, err, "Should remove pending confirmation without error")

	// Verify
	err = db.conn.QueryRow("SELECT COUNT(*) FROM pending_confirmations WHERE message_id = ?", messageID).Scan(&count)
	assert.NoError(t, err, "Should execute count query without error")
	assert.Equal(t, 0, count, "Should have removed the record from pending_confirmations")
}

func TestRemovePendingConfirmation_NotFound(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	messageID := "nonexistent"

	err = db.RemovePendingConfirmation(messageID)
	assert.NoError(t, err, "Should not return error when removing nonexistent pending confirmation")
}

func TestIsUserRegisteredInClass_Enrolled(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const userID = "user123"

	// Precondition:
	classID, err := db.CreateClass("basicos", "Basicos 2024")
	assert.NoError(t, err, "Should insert class without error")

	db.RegisterUserIntoClass(userID, classID)
	assert.NoError(t, err, "Should insert enrollment without error")

	// Act
	enrolled, err := db.IsUserRegisteredInClass(userID, classID)
	assert.NoError(t, err, "Should check enrollment without error")
	assert.True(t, enrolled, "User should be enrolled in the class")
}

func TestIsUserRegisteredInClass_NotEnrolled(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const userID = "user123"

	// Precondition: insertar solo clase
	classID, err := db.CreateClass("basicos", "Basicos 2024")
	assert.NoError(t, err, "Should insert class without error")

	// Act
	enrolled, err := db.IsUserRegisteredInClass(userID, classID)
	assert.NoError(t, err, "Should check enrollment without error")
	assert.False(t, enrolled, "User should NOT be enrolled in the class")
}

func TestIsUserBinded_Yes(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const userID = "user123"

	// Precondition:
	db.BindUser(userID, "Jorge")
	assert.NoError(t, err, "Should insert enrollment without error")

	// Act
	binded, err := db.IsUserBinded(userID)
	assert.NoError(t, err, "Should check binding without error")
	assert.True(t, binded, "User should have binded some username")
}

func TestIsUserBinded_No(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	// Act
	binded, err := db.IsUserBinded("123")
	assert.NoError(t, err, "Should check binding without error")
	assert.False(t, binded, "User should have binded some username")
}

func TestGetUsernameByDiscordID(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const discordID = "123"
	const userName = "Jorge"

	// Preconditions
	db.BindUser(discordID, userName)

	// Act
	userNameResponse, err := db.GetUsernameByDiscordID(discordID)

	assert.NoError(t, err, "Should get Username without error")
	assert.Equal(t, userNameResponse, userName, "User names should match")
}

func TestGetClassNameByClassID(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const classCode = "basicos"
	const className = "Basicos 2025"

	// Preconditions
	classID, err := db.CreateClass(classCode, className)
	assert.NoError(t, err)

	// Act
	classNameResponse, err := db.GetClassByClassID(classID)
	assert.NoError(t, err)
	assert.Equal(t, classNameResponse.Name, className)
	assert.Equal(t, classNameResponse.ClassCode, classCode)
}

func TestGetClassNameByClassCode(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const classCode = "basicos"
	const className = "Basicos 2025"

	// Preconditions
	classID, err := db.CreateClass(classCode, className)
	assert.NoError(t, err)

	// Act
	classNameResponse, err := db.GetClassByClassCode(classCode)
	assert.NoError(t, err)
	assert.Equal(t, classNameResponse.Name, className)
	assert.Equal(t, classNameResponse.ID, classID)
}

func TestGetProblemByID(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const classCode = "basicos"
	const className = "Basicos 2025"
	const problemName = "Sorting"

	// Preconditions
	classID, err := db.CreateClass(classCode, className)
	assert.NoError(t, err)

	problemID, err := db.AddProblem(problemName, classID)
	assert.NoError(t, err)

	// Act
	problem, err := db.GetProblemByID(problemID)
	assert.NoError(t, err)
	assert.Equal(t, problem.ClassID, classID)
	assert.Equal(t, problem.Name, problemName)
}

func TestGetAllUsersByClassID(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const classCode = "basicos"
	const className = "Basicos 2025"

	users := []*User{
		{
			DiscordID: "1",
			UserName:  "Jorge",
		},
		{
			DiscordID: "2",
			UserName:  "Luis",
		},
		{
			DiscordID: "3",
			UserName:  "Ibanez",
		},
	}

	// Preconditions
	classID, err := db.CreateClass(classCode, className)
	assert.NoError(t, err)

	for _, user := range users {
		err = db.BindUser(user.DiscordID, user.UserName)
		assert.NoError(t, err)
		err = db.RegisterUserIntoClass(user.DiscordID, classID)
		assert.NoError(t, err)
	}

	// Act
	usersInClass, err := db.GetAllUsersByClassID(classID)
	assert.NoError(t, err)

	// Verify
	assert.Equal(t, usersInClass, users)
}
