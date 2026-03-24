package adapters

import (
	"errors"
	"fmt"
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
)

type Action int

const (
	ActionBindUser Action = iota
	ActionRegisterInClass
	ActionListProblemsByClass
	ActionScoreByClass
	ActionScoreByUserAndClass
	ActionGetUsersInClass
	ActionGetClassesOfUsername
	ActionGetMyUsername
	ActionChangeUsername

	ActionClearClass
	ActionCreateClass
	ActionExterminateUser
	ActionExportClass

	ActionMentorRegisterUser
	ActionMentorDeregisterUser
	ActionMentorAddAC
	ActionMentorUndoAC
	ActionMentorAddProblem
	ActionMentorRemoveProblem
	ActionMentorUpdateProblemClass
	ActionMentorUpdateProblem

	ActionEnrolledDeregisterUser
	ActionEnrolledProposeAC
	ActionEnrolledUndoAC
)

const (
	errorTypeConflict = "Conflict"
	errorTypeNotFound = "NotFound"
	errorTypeDefault  = "Default"

	defaultFallbackMessage = "Unexpected error."
	internalHiddenMessage  = "Unexpected internal error bro."
)

var templates = map[Action]map[string]string{
	ActionBindUser: {
		errorTypeConflict: "Username `%s` is already taken by another user. Please choose a different username.",
		errorTypeDefault:  "Unexpected error.",
	},
	ActionRegisterInClass: {
		errorTypeNotFound: "Class code `%s` not found or user `%s` not binded yet",
		errorTypeConflict: "User `%s` already registered on class `%s`",
		errorTypeDefault:  "Unexpected error.",
	},
	ActionListProblemsByClass: {
		errorTypeNotFound: "Class code `%s` not found",
		errorTypeDefault:  "Unexpected error.",
	},
	ActionScoreByClass: {
		errorTypeNotFound: "Class code %s doesn't match with any class",
		errorTypeDefault:  "Unexpected error.",
	},
	ActionScoreByUserAndClass: {
		errorTypeNotFound: "Not found info for class code `%[2]s` or user name `%[1]s`",
		errorTypeDefault:  "User `%[1]s` is registered in `%[2]s`? I didn't find any AC",
	},
	ActionGetUsersInClass: {
		errorTypeNotFound: "No class was found with class code %s",
		errorTypeDefault:  "Unexpected error.",
	},
	ActionGetClassesOfUsername: {
		errorTypeNotFound: "Username `%s` not found",
		errorTypeDefault:  "Unexpected error.",
	},
	ActionGetMyUsername: {
		errorTypeNotFound: "You are not binded to any username. Use `!bind <username>` to bind your account.",
		errorTypeDefault:  "Unexpected error.",
	},
	ActionChangeUsername: {
		errorTypeConflict: "Username `%s` is already taken by another user. Please choose a different username.",
		errorTypeDefault:  "Unexpected error.",
	},
	ActionClearClass: {
		errorTypeNotFound: "Class code `%s` not found",
		errorTypeDefault:  "Unexpected error.",
	},
	ActionCreateClass: {
		errorTypeConflict: "Class code `%s` already exists",
		errorTypeDefault:  "Unexpected error.",
	},
	ActionExterminateUser: {
		errorTypeNotFound: "User <@%s> not found in the system",
		errorTypeDefault:  "Unexpected error while exterminating user.",
	},
	ActionExportClass: {
		errorTypeNotFound: "Class code `%s` not found",
		errorTypeDefault:  "Unexpected error while retrieving class data.",
	},
	ActionMentorRegisterUser: {
		errorTypeNotFound: "Username `%s` or class code `%s` not found",
		errorTypeDefault:  "Unexpected error.",
	},
	ActionMentorDeregisterUser: {
		errorTypeNotFound: "Username `%s` or class code `%s` not found",
		errorTypeDefault:  "Unexpected error.",
	},
	ActionMentorAddAC: {
		errorTypeNotFound: "Username `%s` or problem id `%d` not found, probably you are not binded or registered in the same class problem lives",
		errorTypeDefault:  "Unexpected error.",
	},
	ActionMentorUndoAC: {
		errorTypeNotFound: "Username `%s` or problem id `%d` not found",
		errorTypeConflict: "User `%s` has not solved problem `%d` yet or is not enrolled in the class",
		errorTypeDefault:  "Unexpected error.",
	},
	ActionMentorAddProblem: {
		errorTypeNotFound: "Class code `%[2]s` not found",
		errorTypeConflict: "Problem `%[1]s` already exists on class `%[2]s` with the ID: `%[3]d`",
		errorTypeDefault:  "Unexpected error.",
	},
	ActionMentorRemoveProblem: {
		errorTypeNotFound: "Problem id `%d` not found",
		errorTypeDefault:  "Unexpected error.",
	},
	ActionMentorUpdateProblemClass: {
		errorTypeNotFound: "Class code `%s` not found",
		errorTypeDefault:  "Unexpected error.",
	},
	ActionMentorUpdateProblem: {
		errorTypeNotFound: "Problem with ID: `%d` not found",
		errorTypeDefault:  "Unexpected error.",
	},
	ActionEnrolledDeregisterUser: {
		errorTypeNotFound: "User `%s` is not registered in class `%s` or class doesn't even exist",
		errorTypeDefault:  "Unexpected error.",
	},
	ActionEnrolledProposeAC: {
		errorTypeNotFound: "Can you check if user `%s` is binded and registered on the same class of problem `%d`?",
		errorTypeConflict: "User `%s` has already solved problem `%d`",
		errorTypeDefault:  "Unexpected error.",
	},
	ActionEnrolledUndoAC: {
		errorTypeNotFound: "Problem `%[2]d` not found or you're not enrolled in the class",
		errorTypeConflict: "Bro `%[1]s`, you haven't solved problem `%[2]d` yet, so you can't undo it",
		errorTypeDefault:  "Unexpected error.",
	},
}

// MapError maps service-layer errors to user-friendly messages by action.
func MapError(err error, action Action, args ...any) string {
	if err == nil {
		return defaultFallbackMessage
	}

	var internalErr *serviceerr.InternalError
	if errors.As(err, &internalErr) {
		// Generic message for internal errors, we don't want to leak details
		return internalHiddenMessage
	}

	if msg, ok := resolveConflict(err, action, args...); ok {
		return msg
	}

	if msg, ok := resolveNotFound(err, action, args...); ok {
		return msg
	}

	if msg, ok := fromTemplate(action, errorTypeDefault, args...); ok {
		return msg
	}

	return defaultFallbackMessage
}

func resolveConflict(err error, action Action, args ...any) (string, bool) {
	var conflictErr *serviceerr.ConflictError
	if !errors.As(err, &conflictErr) {
		return "", false
	}

	return fromTemplate(action, errorTypeConflict, args...)
}

func resolveNotFound(err error, action Action, args ...any) (string, bool) {
	var notFoundErr *serviceerr.NotFoundInfoError
	if !errors.As(err, &notFoundErr) {
		return "", false
	}

	return fromTemplate(action, errorTypeNotFound, args...)
}

func fromTemplate(action Action, kind string, args ...any) (string, bool) {
	byType, ok := templates[action]
	if !ok {
		return "", false
	}

	tpl, ok := byType[kind]
	if !ok || strings.TrimSpace(tpl) == "" {
		return "", false
	}

	if len(args) == 0 || !strings.Contains(tpl, "%") {
		return tpl, true
	}

	return fmt.Sprintf(tpl, args...), true
}
