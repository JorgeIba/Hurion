package main

import (
	"fmt"
	"log"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/excelizer"
)

func main() {
	// Example usage of the excelizer package

	// Create sample users using excelizer.User
	users := []excelizer.User{
		{ID: "123456789", Name: "Alice"},
		{ID: "987654321", Name: "Bob"},
		{ID: "555666777", Name: "Charlie"},
		{ID: "111222333", Name: "Diana"},
	}

	// Create sample problems using excelizer.Problem
	problems := []excelizer.Problem{
		{ID: 1, Name: "Two Sum", URL: "https://leetcode.com/problems/two-sum/"},
		{ID: 2, Name: "Add Two Numbers", URL: "https://leetcode.com/problems/add-two-numbers/"},
		{ID: 3, Name: "Longest Substring", URL: "https://leetcode.com/problems/longest-substring-without-repeating-characters/"},
		{ID: 4, Name: "Median of Two Arrays", URL: "https://leetcode.com/problems/median-of-two-sorted-arrays/"},
	}

	// Create sample user scores using excelizer.UserScore
	userScores := []excelizer.UserScore{
		{
			User:          users[0], // Alice
			CountProblems: 3,
			Problems: map[excelizer.Problem]bool{
				problems[0]: true,  // Two Sum - solved
				problems[1]: true,  // Add Two Numbers - solved
				problems[2]: true,  // Longest Substring - solved
				problems[3]: false, // Median of Two Arrays - not solved
			},
		},
		{
			User:          users[1], // Bob
			CountProblems: 2,
			Problems: map[excelizer.Problem]bool{
				problems[0]: true,  // Two Sum - solved
				problems[1]: false, // Add Two Numbers - not solved
				problems[2]: false, // Longest Substring - not solved
				problems[3]: true,  // Median of Two Arrays - solved
			},
		},
		{
			User:          users[2], // Charlie
			CountProblems: 1,
			Problems: map[excelizer.Problem]bool{
				problems[0]: true,  // Two Sum - solved
				problems[1]: false, // Add Two Numbers - not solved
				problems[2]: false, // Longest Substring - not solved
				problems[3]: false, // Median of Two Arrays - not solved
			},
		},
		{
			User:          users[3], // Diana
			CountProblems: 4,
			Problems: map[excelizer.Problem]bool{
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

	err := excelizer.AddUsersSheet(excel, "Enrolled Users", users)
	if err != nil {
		log.Fatalf("Error adding users sheet: %v", err)
	}

	err = excelizer.AddProblemsSheet(excel, "Problems List", problems)
	if err != nil {
		log.Fatalf("Error adding problems sheet: %v", err)
	}

	err = excelizer.AddScoresSheet(excel, "Scores", userScores, problems)
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
	fmt.Println("  3. Scores - User progress with problem solving status")
}
