package main

import (
	"fmt"
	"log"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/data"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/excelizer"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/services"
)

func main() {
	// Example usage of the excelizer package

	// Create sample users
	users := []data.User{
		{UserID: "123456789", UserName: "Alice"},
		{UserID: "987654321", UserName: "Bob"},
		{UserID: "555666777", UserName: "Charlie"},
		{UserID: "111222333", UserName: "Diana"},
	}

	// Create sample problems
	problems := []data.Problem{
		{ID: 1, Name: "Two Sum", ClassID: 1, URL: "https://leetcode.com/problems/two-sum/"},
		{ID: 2, Name: "Add Two Numbers", ClassID: 1, URL: "https://leetcode.com/problems/add-two-numbers/"},
		{ID: 3, Name: "Longest Substring", ClassID: 1, URL: "https://leetcode.com/problems/longest-substring-without-repeating-characters/"},
		{ID: 4, Name: "Median of Two Arrays", ClassID: 1, URL: "https://leetcode.com/problems/median-of-two-sorted-arrays/"},
	}

	// Create sample user scores using PairUserProblemScore
	userScores := []services.PairUserProblemScore{
		{
			User:          users[0], // Alice
			CountProblems: 3,
			Problems: map[data.Problem]bool{
				problems[0]: true,  // Two Sum - solved
				problems[1]: true,  // Add Two Numbers - solved
				problems[2]: true,  // Longest Substring - solved
				problems[3]: false, // Median of Two Arrays - not solved
			},
		},
		{
			User:          users[1], // Bob
			CountProblems: 2,
			Problems: map[data.Problem]bool{
				problems[0]: true,  // Two Sum - solved
				problems[1]: false, // Add Two Numbers - not solved
				problems[2]: false, // Longest Substring - not solved
				problems[3]: true,  // Median of Two Arrays - solved
			},
		},
		{
			User:          users[2], // Charlie
			CountProblems: 1,
			Problems: map[data.Problem]bool{
				problems[0]: true,  // Two Sum - solved
				problems[1]: false, // Add Two Numbers - not solved
				problems[2]: false, // Longest Substring - not solved
				problems[3]: false, // Median of Two Arrays - not solved
			},
		},
		{
			User:          users[3], // Diana
			CountProblems: 4,
			Problems: map[data.Problem]bool{
				problems[0]: true, // Two Sum - solved
				problems[1]: true, // Add Two Numbers - solved
				problems[2]: true, // Longest Substring - solved
				problems[3]: true, // Median of Two Arrays - solved
			},
		},
	}

	// Create a new Excel file
	excel := excelizer.CreateExcel()
	defer excel.Close()

	// Add users sheet
	err := excelizer.AddUsersSheet(excel, "Enrolled Users", users)
	if err != nil {
		log.Fatalf("Error adding users sheet: %v", err)
	}

	// Add problems sheet
	err = excelizer.AddProblemsSheet(excel, "Problems List", problems)
	if err != nil {
		log.Fatalf("Error adding problems sheet: %v", err)
	}

	// Add scores sheet using PairUserProblemScore
	err = excelizer.AddScoresSheet(excel, "Scores", userScores)
	if err != nil {
		log.Fatalf("Error adding scores sheet: %v", err)
	}

	// Delete the default "Sheet1" that comes with new Excel files
	excel.DeleteSheet("Sheet1")

	// Save the Excel file
	err = excelizer.SaveExcel(excel, "class_report.xlsx")
	if err != nil {
		log.Fatalf("Error saving Excel file: %v", err)
	}

	fmt.Println("Excel file created successfully: class_report.xlsx")
	fmt.Println("Sheets created:")
	fmt.Println("  1. Enrolled Users - List of all users in the class")
	fmt.Println("  2. Problems List - List of all problems in the class")
	fmt.Println("  3. Scores - User progress using PairUserProblemScore model")
	fmt.Println("  4. Formatted Scores - Pre-formatted table using TableScore model")
}
