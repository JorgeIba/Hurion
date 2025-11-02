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

	const userID = "12345"
	const nameQuery = "Jorge"
	const nameExpected = "Jorge"

	err = db.BindUser(User{UserID: userID, UserName: nameQuery})
	assert.NoError(t, err, "Error inserting")

	var userName string
	err = db.conn.QueryRow("SELECT user_name FROM users WHERE user_id = ?", userID).Scan(&userName)
	assert.NoError(t, err, "User not found")
	assert.Equal(t, nameExpected, userName, "Username doesn't match")
}

func TestBindUser_UpdateUser(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const userID = "12345"
	const nameFirst = "Jorge"
	const nameUpdated = "JorgeLuis"
	const nameExpected = "JorgeLuis"

	// Create a new entry
	err = db.BindUser(User{UserID: userID, UserName: nameFirst})
	assert.NoError(t, err, "Error inserting")

	// Update entry
	err = db.BindUser(User{UserID: userID, UserName: nameUpdated})
	assert.NoError(t, err, "Error updating")

	// Verify
	var userName string
	err = db.conn.QueryRow("SELECT user_name FROM users WHERE user_id = ?", userID).Scan(&userName)
	assert.NoError(t, err, "User not found")
	assert.Equal(t, nameExpected, userName, "Username doesn't match")
}

// Register
func TestRegisterUserIntoClass_NewUser(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const userID = "12345"
	const userName = "Jorge"

	// Preconditions
	response, err := db.conn.Exec("INSERT INTO classes (class_code, name) VALUES (?, ?)", "basicos", "Basicos 2025")
	assert.NoError(t, err, "Could not insert class")

	_, err = db.conn.Exec("INSERT INTO users (user_id, user_name) VALUES (?, ?)", userID, userName)
	assert.NoError(t, err, "Could not insert class")

	classID, _ := response.LastInsertId()

	// Register a user
	err = db.RegisterUserIntoClass(userID, classID)
	assert.NoError(t, err, "Expected not error, but got: ")

	// Verify
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM enrollments WHERE user_id = ? AND class_id = ?", userID, classID).Scan(&count)
	assert.NoError(t, err, "Could not query enrollments correctly")

	assert.Equal(t, 1, count, "User should be registered in the class")
}

func TestRegisterUserIntoClass_ExistingUser(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const userID = "12345"
	const userName = "Jorge"

	// Preconditions
	response, err := db.conn.Exec("INSERT INTO classes (class_code, name) VALUES (?, ?)", "basicos", "Basicos 2025")
	assert.NoError(t, err, "Could not insert class")

	_, err = db.conn.Exec("INSERT INTO users (user_id, user_name) VALUES (?, ?)", userID, userName)
	assert.NoError(t, err, "Could not insert class")

	classID, _ := response.LastInsertId()

	// Register a user
	err = db.RegisterUserIntoClass(userID, classID)
	assert.NoError(t, err, "Expected no error, but got: ")

	// Try to register it again
	err = db.RegisterUserIntoClass(userID, classID)
	assert.NoError(t, err, "Expected no error, but got: ")

	// Verify
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM enrollments WHERE user_id = ? AND class_id = ?", userID, classID).Scan(&count)
	assert.NoError(t, err, "Could not query enrollments correctly: ")

	assert.Equal(t, 1, count, "User should be registered in the class")
}

// Deregister
func TestDeregisterUserIntoClass_ExistingUser(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const userID = "12345"
	const classID = int64(1)

	// Preconditions: Create class, user, and problem
	_, err = db.conn.Exec("INSERT INTO classes (id, class_code, name) VALUES (?, ?, ?)", classID, "CS101", "Intro to CS")
	assert.NoError(t, err)

	err = db.BindUser(User{UserID: userID, UserName: "testuser"})
	assert.NoError(t, err)

	problemID, err := db.AddProblem(Problem{Name: "Problem 1", URL: "url", ClassID: classID})
	assert.NoError(t, err)

	// Register user into class
	err = db.RegisterUserIntoClass(userID, classID)
	assert.NoError(t, err)

	// Add AC and pending confirmation for this user in this class
	err = db.AddAC(userID, problemID)
	assert.NoError(t, err)
	err = db.AddPendingConfirmation(PendingConfirmation{MessageID: "msg456", UserID: userID, ProblemID: problemID})
	assert.NoError(t, err)

	// Verify data exists before deregistration
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM enrollments WHERE user_id = ? AND class_id = ?", userID, classID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "User should be registered")

	err = db.conn.QueryRow("SELECT COUNT(*) FROM solved_problems WHERE user_id = ? AND problem_id = ?", userID, problemID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "AC should exist")

	err = db.conn.QueryRow("SELECT COUNT(*) FROM pending_confirmations WHERE user_id = ? AND problem_id = ?", userID, problemID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Pending confirmation should exist")

	// Act: Deregister user
	err = db.DeregisterUserFromClass(userID, classID)
	assert.NoError(t, err)

	// Verify: Enrollment, ACs, and pending confirmations should be deleted
	err = db.conn.QueryRow("SELECT COUNT(*) FROM enrollments WHERE user_id = ? AND class_id = ?", userID, classID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "User should not be registered in the class")

	err = db.conn.QueryRow("SELECT COUNT(*) FROM solved_problems WHERE user_id = ? AND problem_id = ?", userID, problemID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "ACs should be deleted")

	err = db.conn.QueryRow("SELECT COUNT(*) FROM pending_confirmations WHERE user_id = ? AND problem_id = ?", userID, problemID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "Pending confirmations should be deleted")
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

	// Test 1: Add a problem without difficulty and topic
	problemID1, err := db.AddProblem(Problem{Name: "Sorting Algorithms", URL: "url", ClassID: classID})
	assert.NoError(t, err)
	assert.Greater(t, problemID1, int64(0), "Problem ID should be greater than 0")

	// Test 2: Add a problem with difficulty and topic
	difficulty := string(DifficultyMedium)
	topic := "Sorting"
	problemID2, err := db.AddProblem(Problem{
		Name:       "Binary Search",
		URL:        "url2",
		ClassID:    classID,
		Difficulty: &difficulty,
		Topic:      &topic,
	})
	assert.NoError(t, err)
	assert.Greater(t, problemID2, int64(0), "Problem ID should be greater than 0")

	// Verify the problem with difficulty and topic
	problem, err := db.GetProblemByID(problemID2)
	assert.NoError(t, err)
	assert.NotNil(t, problem.Difficulty)
	assert.Equal(t, "Medium", *problem.Difficulty)
	assert.NotNil(t, problem.Topic)
	assert.Equal(t, "Sorting", *problem.Topic)
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
	problemID, err := db.AddProblem(Problem{Name: problemName, URL: "url", ClassID: classID})
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
	const userID = "user123"

	// Preconditions: Create class, user, problem
	classResult, err := db.conn.Exec(`INSERT INTO classes (class_code, name) VALUES ('intermedios', 'Intermedios 2024')`)
	assert.NoError(t, err)
	classID, err := classResult.LastInsertId()
	assert.NoError(t, err)

	err = db.BindUser(User{UserID: userID, UserName: "testuser"})
	assert.NoError(t, err)

	// Add a problem
	problemID, err := db.AddProblem(Problem{Name: problemName, URL: "url", ClassID: classID})
	assert.NoError(t, err)
	assert.Greater(t, problemID, int64(0), "Problem ID should be greater than 0")

	// Add AC and pending confirmation for this problem
	err = db.AddAC(userID, problemID)
	assert.NoError(t, err)
	err = db.AddPendingConfirmation(PendingConfirmation{MessageID: "msg123", UserID: userID, ProblemID: problemID})
	assert.NoError(t, err)

	// Verify data exists before deletion
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM solved_problems WHERE problem_id = ?", problemID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "AC should exist")

	err = db.conn.QueryRow("SELECT COUNT(*) FROM pending_confirmations WHERE problem_id = ?", problemID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Pending confirmation should exist")

	// Act: Remove the problem
	rowsAffected, err := db.RemoveProblemByID(problemID)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected, "One row should have been removed")

	// Verify: Problem and all related data should be deleted
	err = db.conn.QueryRow("SELECT COUNT(*) FROM problems WHERE id = ?", problemID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "Problem should be deleted")

	err = db.conn.QueryRow("SELECT COUNT(*) FROM solved_problems WHERE problem_id = ?", problemID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "ACs should be deleted")

	err = db.conn.QueryRow("SELECT COUNT(*) FROM pending_confirmations WHERE problem_id = ?", problemID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "Pending confirmations should be deleted")
}

func TestRemoveProblemByNameAndClassID(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const problemName = "Graph Algorithms"
	const userID = "user456"

	// Preconditions: Create class, user, problem
	classResult, err := db.conn.Exec(`INSERT INTO classes (class_code, name) VALUES ('avanzados', 'Avanzados 2025')`)
	assert.NoError(t, err)
	classID, err := classResult.LastInsertId()
	assert.NoError(t, err)

	err = db.BindUser(User{UserID: userID, UserName: "testuser2"})
	assert.NoError(t, err)

	// Add a problem
	problemID, err := db.AddProblem(Problem{Name: problemName, URL: "url", ClassID: classID})
	assert.NoError(t, err)

	// Add AC and pending confirmation
	err = db.AddAC(userID, problemID)
	assert.NoError(t, err)
	err = db.AddPendingConfirmation(PendingConfirmation{MessageID: "msg999", UserID: userID, ProblemID: problemID})
	assert.NoError(t, err)

	// Verify data exists before deletion
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM solved_problems WHERE problem_id = ?", problemID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "AC should exist")

	err = db.conn.QueryRow("SELECT COUNT(*) FROM pending_confirmations WHERE problem_id = ?", problemID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Pending confirmation should exist")

	// Act: Remove the problem by name and class ID
	rowsAffected, err := db.RemoveProblemByNameAndClassID(problemName, classID)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected, "One row should have been removed")

	// Verify: Problem and all related data should be deleted
	err = db.conn.QueryRow("SELECT COUNT(*) FROM problems WHERE id = ?", problemID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "Problem should be deleted")

	err = db.conn.QueryRow("SELECT COUNT(*) FROM solved_problems WHERE problem_id = ?", problemID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "ACs should be deleted")

	err = db.conn.QueryRow("SELECT COUNT(*) FROM pending_confirmations WHERE problem_id = ?", problemID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "Pending confirmations should be deleted")
}

func TestAddAC(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const userID = "user123"
	const classID = int64(1)

	// Preconditions
	problemID, err := db.AddProblem(Problem{Name: "Sorting", URL: "url", ClassID: classID})
	assert.NoError(t, err, "Problem trying to adding the problem")

	// Verify
	err = db.AddAC(userID, problemID)
	assert.NoError(t, err, "Problem trying to ACing the problem for the user")

	err = db.AddAC(userID, problemID)
	assert.NoError(t, err, "Re-adding the same AC should not fail")

	var count int64
	err = db.conn.QueryRow(`
		SELECT COUNT(*)
		FROM solved_problems sp
		JOIN problems p ON sp.problem_id = p.id
		WHERE sp.user_id = ? AND p.class_id = ?;
	`, userID, classID).Scan(&count)
	assert.NoError(t, err, "Cannot check the count for problems solved")

	assert.Equal(t, int64(1), count, "A single problem should have been solved")
}

func TestRemoveAC(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const userID = "user123"
	const classID = int64(1)
	queryCount := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM solved_problems sp 
		JOIN problems p ON sp.problem_id = p.id 
		WHERE sp.user_id = '%s' AND p.class_id = %d;
	`, userID, classID)

	// Preconditions
	problemID, err := db.AddProblem(Problem{Name: "Sorting", URL: "url", ClassID: classID})
	assert.NoError(t, err, "Precondition failed, problem could not be added")

	err = db.AddAC(userID, problemID)
	assert.NoError(t, err, "Precondition failed, problem could not be checked as solved")

	var count int64
	err = db.conn.QueryRow(queryCount).Scan(&count)
	assert.NoError(t, err, "Cannot check the count for problems solved")
	assert.Equal(t, int64(1), count, "A problem should have been added to solved list")

	// Verify
	err = db.RemoveAC(userID, problemID)
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
	difficulty := string(DifficultyHard)
	topic := "Algorithms"

	_, err = db.AddProblem(Problem{Name: "Sorting Stuff 1", URL: "url1", ClassID: classID})
	assert.NoError(t, err, "Precondition failed, problem 1 could not be added")

	_, err = db.AddProblem(Problem{
		Name:       "Sorting Stuff 2",
		URL:        "url2",
		ClassID:    classID,
		Difficulty: &difficulty,
		Topic:      &topic,
	})
	assert.NoError(t, err, "Precondition failed, problem 2 could not be added")

	_, err = db.AddProblem(Problem{Name: "Sorting Stuff 3", URL: "url3", ClassID: classID2})
	assert.NoError(t, err, "Precondition failed, problem 3 could not be added")

	// Do
	problemsList1, err := db.GetProblemsByClass(classID)
	assert.NoError(t, err, "List 1 could not be retrieved")

	problemsList2, err := db.GetProblemsByClass(classID2)
	assert.NoError(t, err, "List 2 could not be retrieved")

	// Verify
	assert.Equal(t, 2, len(problemsList1), "Length are not the same")
	assert.Equal(t, 1, len(problemsList2), "Length are not the same")

	// Verify problem with difficulty and topic
	assert.NotNil(t, problemsList1[1].Difficulty, "Difficulty should not be nil")
	assert.Equal(t, "Hard", *problemsList1[1].Difficulty)
	assert.NotNil(t, problemsList1[1].Topic, "Topic should not be nil")
	assert.Equal(t, "Algorithms", *problemsList1[1].Topic)

	// Verify problem without difficulty and topic
	assert.Nil(t, problemsList1[0].Difficulty, "Difficulty should be nil")
	assert.Nil(t, problemsList1[0].Topic, "Topic should be nil")
}

func TestGetAllSolvedProblemsByUser(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	// Preconditions
	problemID1, err := db.AddProblem(Problem{Name: "Problem 1", URL: "url", ClassID: 1})
	assert.NoError(t, err)

	problemID2, err := db.AddProblem(Problem{Name: "Problem 2", URL: "url", ClassID: 1})
	assert.NoError(t, err)

	problemID3, err := db.AddProblem(Problem{Name: "Problem 3", URL: "url", ClassID: 2})
	assert.NoError(t, err)

	problemID4, err := db.AddProblem(Problem{Name: "Problem 4", URL: "url", ClassID: 2})
	assert.NoError(t, err)

	const userID1 = "user1"
	const userID2 = "user2"
	var updates = []struct {
		userID    string
		problemID int64
		classID   int64
	}{
		{userID: userID1, problemID: problemID1},
		{userID: userID1, problemID: problemID2},
		{userID: userID1, problemID: problemID3},
		{userID: userID1, problemID: problemID4},

		{userID: userID2, problemID: problemID2},
		{userID: userID2, problemID: problemID3},
	}

	for _, update := range updates {
		err = db.AddAC(update.userID, update.problemID)
		assert.NoError(t, err, "Problem trying to ACing the problem for the user")
	}

	// Act
	listUser1Class1, err := db.GetSolvedProblemsByUserAndClass(userID1, 1)
	assert.NoError(t, err, "Problem trying to get the list")
	listUser1Class1OnlyIDs := lo.Map(listUser1Class1, func(problem SolvedProblem, _ int) int64 {
		return problem.ProblemID
	})

	listUser1Class2, err := db.GetSolvedProblemsByUserAndClass(userID1, 2)
	assert.NoError(t, err, "Problem trying to get the list")
	listUser1Class2OnlyIDs := lo.Map(listUser1Class2, func(problem SolvedProblem, _ int) int64 {
		return problem.ProblemID
	})

	listUser2Class1, err := db.GetSolvedProblemsByUserAndClass(userID2, 1)
	assert.NoError(t, err, "Problem trying to get the list")
	listUser2Class1OnlyIDs := lo.Map(listUser2Class1, func(problem SolvedProblem, _ int) int64 {
		return problem.ProblemID
	})

	listUser2Class2, err := db.GetSolvedProblemsByUserAndClass(userID2, 2)
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
	problemID1, err := db.AddProblem(Problem{Name: "Problem 1", URL: "url", ClassID: 1})
	assert.NoError(t, err)

	problemID2, err := db.AddProblem(Problem{Name: "Problem 2", URL: "url", ClassID: 1})
	assert.NoError(t, err)

	problemID3, err := db.AddProblem(Problem{Name: "Problem 3", URL: "url", ClassID: 1})
	assert.NoError(t, err)

	problemID4, err := db.AddProblem(Problem{Name: "Problem 4", URL: "url", ClassID: 2})
	assert.NoError(t, err)

	user1 := User{
		UserID:   "user1",
		UserName: "name1",
	}

	user2 := User{
		UserID:   "user2",
		UserName: "name2",
	}

	var updates = []struct {
		userID    string
		problemID int64
		classID   int64
	}{
		{userID: user1.UserID, problemID: problemID1},
		{userID: user1.UserID, problemID: problemID2},
		{userID: user1.UserID, problemID: problemID3},
		{userID: user1.UserID, problemID: problemID4},

		{userID: user2.UserID, problemID: problemID2},
		{userID: user2.UserID, problemID: problemID3},
	}

	err = db.BindUser(user1)
	assert.NoError(t, err)

	err = db.BindUser(user2)
	assert.NoError(t, err)

	for _, update := range updates {
		err = db.AddAC(update.userID, update.problemID)
		assert.NoError(t, err, "Problem trying to ACing the problem for the user")
	}

	// Act
	solvedProblemsClass1, err := db.GetAllSolvedProblemsByClass(1)
	assert.NoError(t, err, "Problem trying to get the list")

	solvedProblemsClass2, err := db.GetAllSolvedProblemsByClass(2)
	assert.NoError(t, err, "Problem trying to get the list")

	// Verify
	assert.Equal(t, 2, len(solvedProblemsClass1))
	assert.Equal(t, 3, len(solvedProblemsClass1[user1]))
	assert.Equal(t, 2, len(solvedProblemsClass1[user2]))

	assert.Equal(t, 1, len(solvedProblemsClass2))
	assert.Equal(t, 1, len(solvedProblemsClass2[user1]))
	assert.Equal(t, 0, len(solvedProblemsClass2[user2]))
}

func TestClearClass(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	// Preconditions: Create class, user, problem, enrollment, AC, and pending confirmation
	_, err = db.conn.Exec("INSERT INTO classes (id, class_code, name) VALUES (?, ?, ?)", 1, "CS101", "Intro to CS")
	assert.NoError(t, err)

	err = db.BindUser(User{UserID: "user1", UserName: "testuser1"})
	assert.NoError(t, err)

	_, err = db.conn.Exec("INSERT INTO problems (id, name, class_id) VALUES (?, ?, ?)", 1, "Problem 1", 1)
	assert.NoError(t, err)

	_, err = db.conn.Exec("INSERT INTO enrollments (user_id, class_id) VALUES (?, ?)", "user1", 1)
	assert.NoError(t, err)

	_, err = db.conn.Exec("INSERT INTO solved_problems (user_id, problem_id) VALUES (?, ?)", "user1", 1)
	assert.NoError(t, err)

	_, err = db.conn.Exec("INSERT INTO pending_confirmations (message_id, user_id, problem_id) VALUES (?, ?, ?)", "msg789", "user1", 1)
	assert.NoError(t, err)

	// Act: Clear the class
	err = db.ClearClass(1)
	assert.NoError(t, err)

	// Verify: All related data should be deleted
	var count int
	queries := []struct {
		query string
		desc  string
	}{
		{"SELECT COUNT(*) FROM classes WHERE id = 1", "classes"},
		{"SELECT COUNT(*) FROM problems WHERE class_id = 1", "problems"},
		{"SELECT COUNT(*) FROM enrollments WHERE class_id = 1", "enrollments"},
		{"SELECT COUNT(*) FROM solved_problems WHERE problem_id = 1", "solved_problems"},
		{"SELECT COUNT(*) FROM pending_confirmations WHERE problem_id = 1", "pending_confirmations"},
	}

	for _, q := range queries {
		err = db.conn.QueryRow(q.query).Scan(&count)
		assert.NoError(t, err, "Could not check %s", q.desc)
		assert.Equal(t, 0, count, "Expected 0 rows in %s, but found %d", q.desc, count)
	}
}

func TestCreateClass(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	classCode := "avanzados"
	className := "Avanzados 2025"

	// Act
	classID, err := db.CreateClass(Class{ClassCode: classCode, Name: className})
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

func TestGetUserByUserID(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const userID = "123"
	const userName = "Jorge"

	// Preconditions
	err = db.BindUser(User{UserID: userID, UserName: userName})
	assert.NoError(t, err, "Could not insert user")

	// Test
	user, err := db.GetUserByUserID(userID)
	assert.NoError(t, err, "Expected no error, but got: ")
	assert.Equal(t, userID, user.UserID, "UserID doesn't match")
	assert.Equal(t, userName, user.UserName, "UserName doesn't match")
}

func TestGetUserByUserName_NotFound(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	// Test
	_, err = db.GetUserByUserName("NONEXISTENT")
	assert.Error(t, err, "Expected error, but got nil")
	assert.IsType(t, &dataerr.NotFoundError{}, err, "Expected NotFoundError")
}

func TestGetClassByClassCode_Success(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const classCode = "basicos"
	const className = "Basicos 2024"

	// Act
	classID, err := db.CreateClass(Class{ClassCode: classCode, Name: className})
	assert.NoError(t, err)

	// Verify
	class, err := db.GetClassByClassCode(classCode)
	assert.NoError(t, err)
	assert.Equal(t, classID, class.ID)
	assert.Equal(t, classCode, class.ClassCode)
	assert.Equal(t, className, class.Name)
}

func TestGetClassByClassCode_NotFound(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	// Act
	_, err = db.GetClassByClassCode("UNKNOWN")
	assert.Error(t, err)

	// Verify
	assert.Contains(t, err.Error(), "class with code UNKNOWN not found")
}

func TestGetClassByClassID_Success(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const classCode = "basicos"
	const className = "Basicos 2024"

	// Act
	classID, err := db.CreateClass(Class{ClassCode: classCode, Name: className})
	assert.NoError(t, err)

	// Verify
	class, err := db.GetClassByClassID(classID)
	assert.NoError(t, err)
	assert.Equal(t, classID, class.ID)
	assert.Equal(t, classCode, class.ClassCode)
	assert.Equal(t, className, class.Name)
}

func TestGetClassByClassID_NotFound(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	// Act
	_, err = db.GetClassByClassID(999999)
	assert.Error(t, err)

	// Verify
	assert.Contains(t, err.Error(), "class with id 999999 not found")
}

func TestAddPendingConfirmation(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const messageID = "123456"
	const userID = "789"
	const problemID = int64(1)

	// Act
	err = db.AddPendingConfirmation(PendingConfirmation{MessageID: messageID, UserID: userID, ProblemID: problemID})
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
	err = db.AddPendingConfirmation(PendingConfirmation{MessageID: messageIDPrevious, UserID: userID, ProblemID: problemID})
	assert.NoError(t, err, "Expected no error on first insertion")

	var messageID string
	err = db.conn.QueryRow(query).Scan(&messageID)
	assert.NoError(t, err, "Expected no error when retrieving the first message_id")
	assert.Equal(t, messageIDPrevious, messageID, "Expected message_id to be 'msg1'")

	// Act
	err = db.AddPendingConfirmation(PendingConfirmation{MessageID: messageIDUpdated, UserID: userID, ProblemID: problemID})
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
	err = db.AddPendingConfirmation(PendingConfirmation{MessageID: messageID, UserID: userID, ProblemID: problemID})
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
	err = db.AddPendingConfirmation(PendingConfirmation{MessageID: messageID, UserID: userID, ProblemID: problemID})
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
	classID, err := db.CreateClass(Class{ClassCode: "basicos", Name: "Basicos 2024"})
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
	classID, err := db.CreateClass(Class{ClassCode: "basicos", Name: "Basicos 2024"})
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
	db.BindUser(User{UserID: userID, UserName: "Jorge"})
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

func TestGetClassNameByClassID(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const classCode = "basicos"
	const className = "Basicos 2025"

	// Preconditions
	classID, err := db.CreateClass(Class{ClassCode: classCode, Name: className})
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
	classID, err := db.CreateClass(Class{ClassCode: classCode, Name: className})
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
	const url = "url"
	const problemName = "Sorting"

	// Preconditions
	classID, err := db.CreateClass(Class{ClassCode: classCode, Name: className})
	assert.NoError(t, err)

	difficulty := string(DifficultyEasy)
	topic := "Data Structures"

	problemID, err := db.AddProblem(Problem{
		Name:       problemName,
		URL:        url,
		ClassID:    classID,
		Difficulty: &difficulty,
		Topic:      &topic,
	})
	assert.NoError(t, err)

	// Act
	problem, err := db.GetProblemByID(problemID)
	assert.NoError(t, err)

	// Verify
	assert.Equal(t, classID, problem.ClassID)
	assert.Equal(t, url, problem.URL)
	assert.Equal(t, problemName, problem.Name)
	assert.NotNil(t, problem.Difficulty, "Difficulty should not be nil")
	assert.Equal(t, "Easy", *problem.Difficulty)
	assert.NotNil(t, problem.Topic, "Topic should not be nil")
	assert.Equal(t, "Data Structures", *problem.Topic)
}

func TestGetAllUsersByClassID(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const classCode = "basicos"
	const className = "Basicos 2025"

	users := []User{
		{
			UserID:   "1",
			UserName: "Jorge",
		},
		{
			UserID:   "2",
			UserName: "Luis",
		},
		{
			UserID:   "3",
			UserName: "Ibanez",
		},
	}

	// Preconditions
	classID, err := db.CreateClass(Class{ClassCode: classCode, Name: className})
	assert.NoError(t, err)

	for _, user := range users {
		err = db.BindUser(user)
		assert.NoError(t, err)
		err = db.RegisterUserIntoClass(user.UserID, classID)
		assert.NoError(t, err)
	}

	// Act
	usersInClass, err := db.GetAllUsersByClassID(classID)
	assert.NoError(t, err)

	// Verify
	assert.Equal(t, usersInClass, users)
}

func TestGetAllClases(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	classes := []Class{
		{
			ClassCode: "code1",
			Name:      "name1",
		},
		{
			ClassCode: "code2",
			Name:      "name2",
		},
		{
			ClassCode: "code3",
			Name:      "name3",
		},
	}

	// Preconditions
	for i := range len(classes) {
		class := classes[i]

		id, err := db.CreateClass(class)
		assert.NoError(t, err)
		class.ID = id
	}

	// Act
	allClasses, err := db.GetAllClasses()
	assert.NoError(t, err)

	// Verify
	assert.Equal(t, 3, len(allClasses))
}

// UpdateByProblemID Tests
func TestUpdateByProblemID_Success(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	// Preconditions
	classID1, err := db.CreateClass(Class{ClassCode: "intermedios", Name: "Intermedios 2024"})
	assert.NoError(t, err)

	classID2, err := db.CreateClass(Class{ClassCode: "avanzados", Name: "Avanzados 2024"})
	assert.NoError(t, err)

	originalProblem := Problem{
		Name:    "Original Problem",
		URL:     "url",
		ClassID: classID1,
	}
	problemID, err := db.AddProblem(originalProblem)
	assert.NoError(t, err)

	// Act
	updatedProblem := Problem{
		Name:    "Updated Problem Name",
		URL:     "url2",
		ClassID: classID2,
	}
	err = db.UpdateProblemByProblemID(problemID, updatedProblem)
	assert.NoError(t, err, "Should update problem without error")

	// Verify
	retrievedProblem, err := db.GetProblemByID(problemID)
	assert.NoError(t, err, "Should retrieve updated problem")
	assert.Equal(t, updatedProblem.Name, retrievedProblem.Name, "Name should be updated")
	assert.Equal(t, updatedProblem.URL, retrievedProblem.URL, "URL should be updated")
	assert.Equal(t, updatedProblem.ClassID, retrievedProblem.ClassID, "ClassID should be updated")
	assert.Equal(t, problemID, retrievedProblem.ID, "ID should remain the same")
}

func TestUpdateByProblemID_NotFound(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	// Preconditions
	classID, err := db.CreateClass(Class{ClassCode: "test", Name: "Test Class"})
	assert.NoError(t, err)

	// Act
	nonExistentProblemID := int64(99999)
	updateProblem := Problem{
		Name:    "Updated Problem",
		URL:     "url",
		ClassID: classID,
	}
	err = db.UpdateProblemByProblemID(nonExistentProblemID, updateProblem)

	// Verify
	assert.Error(t, err, "Should return error for non-existent problem")
	var notFoundErr *dataerr.NotFoundError
	assert.ErrorAs(t, err, &notFoundErr, "Should return NotFoundError type")
}

func TestGetClassesByUserID(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	class1 := Class{
		ClassCode: "class1",
		Name:      "Class 1",
	}
	class2 := Class{
		ClassCode: "class2",
		Name:      "Class 2",
	}
	class3 := Class{
		ClassCode: "class3",
		Name:      "Class 3",
	}

	classes := []Class{class1, class2, class3}

	const userID = "user123"

	// Precondition
	for i := range len(classes) {
		class := &classes[i]
		class.ID, err = db.CreateClass(*class)
		assert.NoError(t, err)
	}

	for _, class := range classes {
		err = db.RegisterUserIntoClass(userID, class.ID)
		assert.NoError(t, err, "Should register user into class without error")
	}

	// Act
	classes_response, err := db.GetClassesByUserID(userID)
	assert.NoError(t, err)

	// Verify
	assert.Len(t, classes_response, 3, "Should return two classes for the user")
}

func TestHasUserSolvedProblem_ReturnsTrue(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	// Preconditions
	user := User{UserID: "user123", UserName: "TestUser"}
	err = db.BindUser(user)
	assert.NoError(t, err)

	classID, err := db.CreateClass(Class{ClassCode: "basicos", Name: "Basicos 2025"})
	assert.NoError(t, err)

	problemID, err := db.AddProblem(Problem{Name: "Test Problem", URL: "url", ClassID: classID})
	assert.NoError(t, err)

	err = db.AddAC(user.UserID, problemID)
	assert.NoError(t, err)

	// Act
	solved, err := db.HasUserSolvedProblem(user.UserID, problemID)
	assert.NoError(t, err)

	// Verify
	assert.True(t, solved)
}

func TestHasUserSolvedProblem_ReturnsFalse(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	// Preconditions
	user := User{UserID: "user123", UserName: "TestUser"}
	err = db.BindUser(user)
	assert.NoError(t, err)

	classID, err := db.CreateClass(Class{ClassCode: "basicos", Name: "Basicos 2025"})
	assert.NoError(t, err)

	problemID, err := db.AddProblem(Problem{Name: "Test Problem", URL: "url", ClassID: classID})
	assert.NoError(t, err)

	// Act
	solved, err := db.HasUserSolvedProblem(user.UserID, problemID)
	assert.NoError(t, err)

	// Verify
	assert.False(t, solved)
}

// IsUsernameAvailable
func TestIsUsernameAvailable_AvailableUsername(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const userID = "12345"
	const userName = "jorge_test"
	const availableUsername = "maria_test"

	// Preconditions
	err = db.BindUser(User{UserID: userID, UserName: userName})
	assert.NoError(t, err)

	// Act
	isAvailable, err := db.IsUsernameAvailable(availableUsername, userID)
	assert.NoError(t, err)

	// Verify
	assert.True(t, isAvailable, "Username should be available")
}

func TestIsUsernameAvailable_TakenUsername(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const userID1 = "12345"
	const userID2 = "67890"
	const userName = "jorge_test"

	// Preconditions
	err = db.BindUser(User{UserID: userID1, UserName: userName})
	assert.NoError(t, err)

	// Act
	isAvailable, err := db.IsUsernameAvailable(userName, userID2)
	assert.NoError(t, err)

	// Verify
	assert.False(t, isAvailable, "Username should not be available")
}

func TestIsUsernameAvailable_SameUserCanKeepUsername(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const userID = "12345"
	const userName = "jorge_test"

	// Preconditions
	err = db.BindUser(User{UserID: userID, UserName: userName})
	assert.NoError(t, err)

	// Act
	isAvailable, err := db.IsUsernameAvailable(userName, userID)
	assert.NoError(t, err)

	// Verify
	assert.True(t, isAvailable, "User should be able to keep their own username")
}

func TestIsUsernameAvailable_EmptyDatabase(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const userID = "12345"
	const userName = "jorge_test"

	// Act
	isAvailable, err := db.IsUsernameAvailable(userName, userID)
	assert.NoError(t, err)

	// Verify
	assert.True(t, isAvailable, "Username should be available in empty database")
}

func TestHasUsernameBound_UserWithUsername(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const userID = "12345"
	const userName = "jorge_test"

	// Preconditions
	err = db.BindUser(User{UserID: userID, UserName: userName})
	assert.NoError(t, err)

	// Act
	hasBound, err := db.HasUsernameBound(userID)
	assert.NoError(t, err)

	// Verify
	assert.True(t, hasBound, "User should have username bound")
}

func TestHasUsernameBound_UserWithoutUsername(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const userID = "12345"

	// Act
	hasBound, err := db.HasUsernameBound(userID)
	assert.NoError(t, err)

	// Verify
	assert.False(t, hasBound, "User should not have username bound")
}

func TestExterminateUser_CompleteRemoval(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const userID = "12345"
	const userName = "Jorge"

	// Preconditions
	classID, err := db.CreateClass(Class{ClassCode: "test_class", Name: "Test Class"})
	assert.NoError(t, err)

	err = db.BindUser(User{UserID: userID, UserName: userName})
	assert.NoError(t, err)

	err = db.RegisterUserIntoClass(userID, classID)
	assert.NoError(t, err)

	problemID, err := db.AddProblem(Problem{Name: "Test Problem", URL: "http://test.com", ClassID: classID})
	assert.NoError(t, err)

	err = db.AddAC(userID, problemID)
	assert.NoError(t, err)

	err = db.AddPendingConfirmation(PendingConfirmation{MessageID: "msg123", UserID: userID, ProblemID: problemID})
	assert.NoError(t, err)

	// Act
	err = db.ExterminateUser(userID)
	assert.NoError(t, err)

	// Verify
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM users WHERE user_id = ?", userID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "User should be removed from users table")

	// Check enrollments table
	err = db.conn.QueryRow("SELECT COUNT(*) FROM enrollments WHERE user_id = ?", userID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "User should be removed from enrollments table")

	// Check solved_problems table
	err = db.conn.QueryRow("SELECT COUNT(*) FROM solved_problems WHERE user_id = ?", userID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "User should be removed from solved_problems table")

	// Check pending_confirmations table
	err = db.conn.QueryRow("SELECT COUNT(*) FROM pending_confirmations WHERE user_id = ?", userID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "User should be removed from pending_confirmations table")
}

func TestExterminateUser_NonexistentUser(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	const userID = "nonexistent"

	// Act
	err = db.ExterminateUser(userID)

	// Verify
	assert.NoError(t, err, "Exterminating nonexistent user should not return error")
}

func TestExterminateUser_MultipleUsers(t *testing.T) {
	db, err := NewDB(testConfig)
	assert.NoError(t, err, "Unable to create a DB")
	defer db.CloseDB()

	users := []User{
		{UserID: "user1", UserName: "Jorge"},
		{UserID: "user2", UserName: "Luis"},
	}

	// Preconditions
	classID, err := db.CreateClass(Class{ClassCode: "test_class", Name: "Test Class"})
	assert.NoError(t, err)

	for _, user := range users {
		err = db.BindUser(user)
		assert.NoError(t, err)
		err = db.RegisterUserIntoClass(user.UserID, classID)
		assert.NoError(t, err)
	}

	problemID, err := db.AddProblem(Problem{Name: "Test Problem", URL: "url", ClassID: classID})
	assert.NoError(t, err)

	for _, user := range users {
		err = db.AddAC(user.UserID, problemID)
		assert.NoError(t, err)
	}

	// Act
	err = db.ExterminateUser(users[0].UserID)
	assert.NoError(t, err)

	// Verify
	var count int

	// User1 should be completely removed
	err = db.conn.QueryRow("SELECT COUNT(*) FROM users WHERE user_id = ?", users[0].UserID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)

	err = db.conn.QueryRow("SELECT COUNT(*) FROM enrollments WHERE user_id = ?", users[0].UserID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)

	// User2 should still exist
	err = db.conn.QueryRow("SELECT COUNT(*) FROM users WHERE user_id = ?", users[1].UserID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	err = db.conn.QueryRow("SELECT COUNT(*) FROM enrollments WHERE user_id = ?", users[1].UserID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}
