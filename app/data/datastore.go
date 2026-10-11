package data

type QueryDatastore interface {
	GetProblemsByClass(classID int64) ([]Problem, error)
	GetAllSolvedProblemsByClass(classID int64) (map[User][]SolvedProblem, error)
	GetUserByUserID(userID string) (User, error)
	GetClassByClassCode(classCode string) (Class, error)
	GetClassesByClassCodes(classCodes []string) ([]Class, error)
	GetClassByClassID(classID int64) (Class, error)
	GetClassCodeByRoomID(roomID string) (string, error)
	IsRoomRegistered(roomID string) (string, bool, error)
	HasClassAccessRole(classCode, roleID string) (bool, error)
	GetClassroomsByClassCode(classCode string) ([]Classroom, error)
	GetClassAccessRolesByClassCode(classCode string) ([]ClassAccessRole, error)
	GetPendingConfirmation(messageID string) (string, int64, error)
	ExistsPendingConfirmation(messageID string) (bool, error)
	GetProblemByID(problemID int64) (Problem, error)
	GetProblemsByIDs(problemIDs []int64) ([]Problem, error)
	GetActiveUsersByClassID(classID int64) ([]User, error)
	GetClassesWhereUserIsActive(userID string) ([]Class, error)
	GetClassesByAccessRoles(roleIDs []string) ([]Class, error)
	GetAllClasses() ([]Class, error)
	HasUserSolvedProblem(userID string, problemID int64) (bool, error)
	GetACValidationData(targets []ACKey) (map[ACKey]ACValidationData, error)
}

type UpdateDataStore interface {
	EnsureUser(user User) error
	AddClassroom(classroom Classroom) error
	RemoveClassroomByRoomID(roomID string) error
	AddClassAccessRole(role ClassAccessRole) error
	RemoveClassAccessRole(role ClassAccessRole) error
	AddProblem(problem Problem) (int64, error)
	AddProblemsBatch(problems []Problem) ([]int64, error)
	RemoveProblemByID(problemID int64) (int64, error)
	AddAC(userID string, problemID int64) error
	AddACBatch(targets []ACKey, users []User) (int, error)
	RemoveAC(userID string, problemID int64) error
	ClearClassContents(classID int64) error
	DeleteClass(classID int64) error
	CreateClass(class Class) (int64, error)
	AddPendingConfirmation(confirmation PendingConfirmation) error
	RemovePendingConfirmation(messageID string) error
	ReplaceProblemByProblemID(problemID int64, problem Problem) error
	PatchProblemByProblemID(problemID int64, patch ProblemPatch) error
	ExterminateUser(userID string) error
}

//go:generate go tool mockgen -destination=mock_data/mock_datastore.go -package=mock_data . DataStore
type DataStore interface {
	QueryDatastore
	UpdateDataStore
}
