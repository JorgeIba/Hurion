package data

type QueryDatastore interface {
	GetProblemsByClass(classID int64) ([]Problem, error)
	GetAllSolvedProblemsByClass(classID int64) (map[User][]SolvedProblem, error)
	GetUserIDByUserName(userName string) (string, error)
	GetClassByClassCode(classCode string) (Class, error)
	GetClassIDByClassCode(classCode string) (int64, error)
	IsUserBinded(userID string) (bool, error)
	IsUserRegisteredInClass(userID string, classID int64) (bool, error)
	GetPendingConfirmation(messageID string) (string, int64, error)
	GetProblemByID(problemID int64) (Problem, error)
	GetAllUsersByClassID(classID int64) ([]User, error)
	GetAllClasses() ([]Class, error)
}

type UpdateDataStore interface {
	BindUser(userID, userName string) error
	RegisterUserIntoClass(userID string, classID int64) error
	DeregisterUserFromClass(userID string, classID int64) error
	AddProblem(problemName, url string, classID int64) (int64, error)
	RemoveProblemByID(problemID int64) (int64, error)
	AddAC(userID string, problemID int64) error
	RemoveAC(userID string, problemID int64) error
	ClearClass(classID int64) error
	CreateClass(classCode, className string) (int64, error)
	AddPendingConfirmation(messageID, userID string, problemID int64) error
	RemovePendingConfirmation(messageID string) error
}

type DataStore interface {
	QueryDatastore
	UpdateDataStore
}
