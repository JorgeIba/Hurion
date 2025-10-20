package excelizer

type User struct {
	ID   string
	Name string
}

type Problem struct {
	ID   int64
	Name string
	URL  string
}

type UserScore struct {
	User          User
	CountProblems int
	Problems      map[Problem]bool
}
