package data

type QueryDatastore interface {
	GetProblemsByClass(classID int64) ([]string, error)
	GetAllSolvedProblemsByUser(discordID string, classID int64) ([]int64, error)
	GetAllSolvedProblemsByClass(classID int64) (map[string][]int64, error)
	GetDiscordIDByUserName(userName string) (string, error)
	GetClassIDByClassCode(classCode string) (int64, error)
	IsUserRegisteredInClass(userID string, classID int64) (bool, error)
	GetPendingConfirmation(messageID string) (string, int64, error)
}

type UpdateDataStore interface {
	BindUser(discordId, userName string) error
	RegisterUserIntoClass(discordID string, classID int64) error
	DeregisterUserFromClass(discordID string, classID int64) error
	AddProblem(problemName string, classID int64) (int64, error)
	RemoveProblem(problemName string, classID int64) (int64, error)
	AddAC(discordID string, problemID, classID int64) error
	RemoveAC(discordID string, problemID, classID int64) error
	ClearClass(classID int) error
	CreateClass(classCode, className string) (int64, error)
	AddPendingConfirmation(messageID, userID string, problemID int64) error
	RemovePendingConfirmation(messageID string) error
}

type DataStore interface {
	QueryDatastore
	UpdateDataStore
}
