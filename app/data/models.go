package data

type Problem struct {
	ID      int64
	Name    string
	ClassID int64
}

type Class struct {
	ID        int64
	ClassCode string
	Name      string
}
