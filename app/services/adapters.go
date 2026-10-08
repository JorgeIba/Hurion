package services

import (
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// ============================================================================
// Service Model to Request Params Adapters
// ============================================================================

// MapProblemToUpdateProblemParams prepares a pending update from saved values.
func MapProblemToUpdateProblemParams(problem service_model.Problem, classCode string) service_model.UpdateProblemParams {
	return service_model.UpdateProblemParams{
		ProblemID:  problem.ID,
		Name:       problem.Name,
		URL:        problem.URL,
		ClassCode:  classCode,
		Difficulty: problem.Difficulty,
		Topic:      problem.Topic,
		Priority:   problem.Priority,
	}
}

// ============================================================================
// Data to Service Adapters
// ============================================================================

func mapDataUserToServiceUser(u data.User) service_model.User {
	return service_model.User{
		UserID:      u.UserID,
		DisplayName: u.DisplayName,
	}
}

func mapDataUsersToServiceUsers(users []data.User) []service_model.User {
	result := make([]service_model.User, len(users))
	for i, u := range users {
		result[i] = mapDataUserToServiceUser(u)
	}
	return result
}

func mapDataProblemToServiceProblem(p data.Problem) service_model.Problem {
	return service_model.Problem{
		ID:         p.ID,
		Name:       p.Name,
		ClassID:    p.ClassID,
		URL:        p.URL,
		Difficulty: p.Difficulty,
		Topic:      p.Topic,
		Priority:   p.Priority,
		CreatedAt:  p.CreatedAt,
	}
}

func mapDataProblemsToServiceProblems(problems []data.Problem) []service_model.Problem {
	result := make([]service_model.Problem, len(problems))
	for i, p := range problems {
		result[i] = mapDataProblemToServiceProblem(p)
	}
	return result
}

func mapDataClassToServiceClass(c data.Class) service_model.Class {
	return service_model.Class{
		ID:        c.ID,
		ClassCode: c.ClassCode,
		Name:      c.Name,
	}
}

func mapDataClassesToServiceClasses(classes []data.Class) []service_model.Class {
	result := make([]service_model.Class, len(classes))
	for i, c := range classes {
		result[i] = mapDataClassToServiceClass(c)
	}
	return result
}

func mapDataClassroomToServiceClassroom(r data.Classroom) service_model.Classroom {
	return service_model.Classroom{
		RoomID:    r.RoomID,
		ClassCode: r.ClassCode,
	}
}

func mapDataClassroomsToServiceClassrooms(classrooms []data.Classroom) []service_model.Classroom {
	result := make([]service_model.Classroom, len(classrooms))
	for i, r := range classrooms {
		result[i] = mapDataClassroomToServiceClassroom(r)
	}
	return result
}

func mapDataRoleToServiceRole(r data.ClassAccessRole) service_model.ClassAccessRole {
	return service_model.ClassAccessRole{
		RoleID:    r.RoleID,
		ClassCode: r.ClassCode,
	}
}

func mapDataRolesToServiceRoles(roles []data.ClassAccessRole) []service_model.ClassAccessRole {
	result := make([]service_model.ClassAccessRole, len(roles))
	for i, r := range roles {
		result[i] = mapDataRoleToServiceRole(r)
	}
	return result
}

func mapDataSolvedProblemToServiceSolvedProblem(sp data.SolvedProblem) service_model.SolvedProblem {
	return service_model.SolvedProblem{
		UserID:    sp.UserID,
		ProblemID: sp.ProblemID,
		SolvedAt:  sp.SolvedAt,
	}
}

func mapDataSolvedProblemsToServiceSolvedProblems(sps []data.SolvedProblem) []service_model.SolvedProblem {
	result := make([]service_model.SolvedProblem, len(sps))
	for i, sp := range sps {
		result[i] = mapDataSolvedProblemToServiceSolvedProblem(sp)
	}
	return result
}

func mapDataPendingConfirmationToServicePendingConfirmation(pc data.PendingConfirmation) service_model.PendingConfirmation {
	return service_model.PendingConfirmation{
		MessageID: pc.MessageID,
		UserID:    pc.UserID,
		ProblemID: pc.ProblemID,
	}
}

func mapServicePendingConfirmationToDataPendingConfirmation(pc service_model.PendingConfirmation) data.PendingConfirmation {
	return data.PendingConfirmation{
		MessageID: pc.MessageID,
		UserID:    pc.UserID,
		ProblemID: pc.ProblemID,
	}
}

// ============================================================================
// Service to Data Adapters
// ============================================================================

// mapCreateProblemParamsToDataProblem uses the class ID resolved by the caller.
func mapCreateProblemParamsToDataProblem(params service_model.CreateProblemParams, classID int64) data.Problem {
	return data.Problem{
		Name:       params.Name,
		URL:        params.URL,
		ClassID:    classID,
		Difficulty: params.Difficulty,
		Topic:      params.Topic,
		Priority:   params.Priority,
	}
}

func mapUpdateProblemParamsToDataProblem(params service_model.UpdateProblemParams, classID int64) data.Problem {
	return data.Problem{
		ID:         params.ProblemID,
		Name:       params.Name,
		URL:        params.URL,
		ClassID:    classID,
		Difficulty: params.Difficulty,
		Topic:      params.Topic,
		Priority:   params.Priority,
	}
}

func mapServiceUserToDataUser(u service_model.User) data.User {
	return data.User{
		UserID:      u.UserID,
		DisplayName: u.DisplayName,
	}
}

func mapServiceClassToDataClass(c service_model.Class) data.Class {
	return data.Class{
		ID:        c.ID,
		ClassCode: c.ClassCode,
		Name:      c.Name,
	}
}

func mapServiceClassroomToDataClassroom(r service_model.Classroom) data.Classroom {
	return data.Classroom{
		RoomID:    r.RoomID,
		ClassCode: r.ClassCode,
	}
}

func mapServiceRoleToDataRole(r service_model.ClassAccessRole) data.ClassAccessRole {
	return data.ClassAccessRole{
		RoleID:    r.RoleID,
		ClassCode: r.ClassCode,
	}
}
