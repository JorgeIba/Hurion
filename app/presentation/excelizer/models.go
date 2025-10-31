package excelizer

type User struct {
	ID   string
	Name string
}

type Problem struct {
	ID         int64
	Name       string
	URL        string
	Difficulty string
	Topic      string
}

type UserScore struct {
	User          User
	CountProblems int
	Problems      map[Problem]bool
}

type ProblemsByTopic struct {
	Topic    string
	Problems []Problem
}
