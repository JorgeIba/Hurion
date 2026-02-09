package data

type QueryDatastore interface {
	GetProblemsByClass(classID int64) ([]Problem, error)
	GetAllSolvedProblemsByClass(classID int64) (map[User][]SolvedProblem, error)
	GetUserByUserName(userName string) (User, error)
	GetUserByUserID(userID string) (User, error)
	GetClassByClassCode(classCode string) (Class, error)
	IsUserBinded(userID string) (bool, error)
	HasUsernameBound(userID string) (bool, error)
	IsUserRegisteredInClass(userID string, classID int64) (bool, error)
	IsUsernameAvailable(userName string, excludeUserID string) (bool, error)
	GetPendingConfirmation(messageID string) (string, int64, error)
	ExistsPendingConfirmation(messageID string) (bool, error)
	GetProblemByID(problemID int64) (Problem, error)
	GetAllUsersByClassID(classID int64) ([]User, error)
	GetAllClasses() ([]Class, error)
	GetClassesByUserID(userID string) ([]Class, error)
	HasUserSolvedProblem(userID string, problemID int64) (bool, error)
}

type UpdateDataStore interface {
	BindUser(user User) error
	RegisterUserIntoClass(userID string, classID int64) error
	DeregisterUserFromClass(userID string, classID int64) error
	AddProblem(problem Problem) (int64, error)
	RemoveProblemByID(problemID int64) (int64, error)
	AddAC(userID string, problemID int64) error
	RemoveAC(userID string, problemID int64) error
	ClearClass(classID int64) error
	CreateClass(class Class) (int64, error)
	AddPendingConfirmation(confirmation PendingConfirmation) error
	RemovePendingConfirmation(messageID string) error
	UpdateProblemByProblemID(problemID int64, problem Problem) error
	ExterminateUser(userID string) error
}

type DataStore interface {
	QueryDatastore
	UpdateDataStore
}
