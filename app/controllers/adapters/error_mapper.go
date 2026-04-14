package adapters

import (
	"errors"
	"fmt"
	"strings"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/customerr/serviceerr"
)

type Action int

type ErrorKind int

const (
	ActionRegisterInClass Action = iota
	ActionListProblemsByClass
	ActionScoreByClass
	ActionScoreByUserAndClass
	ActionGetUsersInClass
	ActionGetClassesOfUser

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

	// ActionSentinel must stay as the last Action constant.
	// It is used as an exclusive upper bound when iterating all actions.
	ActionSentinel
)

const (
	errorKindUserNotEnrolled ErrorKind = iota
	errorKindUserNotFound
	errorKindClassCodeNotFound
	errorKindProblemNotFound
	errorKindDuplicateProblem
	errorKindConflict
	errorKindNotFound
	errorKindInternal
	errorKindDefault

	defaultFallbackMessage = "Unexpected error."
	internalHiddenMessage  = "Unexpected internal error bro."
)

var templates = map[Action]map[ErrorKind]string{
	ActionRegisterInClass: {
		errorKindClassCodeNotFound: "Class code `%[1]s` not found for user `%[2]s`.",
		errorKindConflict:          "User `%[2]s` already registered on class `%[1]s`",
		errorKindDefault:           "Unexpected error while registering in class.",
	},
	ActionListProblemsByClass: {
		errorKindClassCodeNotFound: "Class code `%s` not found",
		errorKindDefault:           "Unexpected error while listing class problems.",
	},
	ActionScoreByClass: {
		errorKindClassCodeNotFound: "Class code `%s` doesn't match with any class",
		errorKindDefault:           "Unexpected error while getting class scores.",
	},
	ActionScoreByUserAndClass: {
		errorKindClassCodeNotFound: "Class code `%[2]s` not found",
		errorKindUserNotFound:      "User `%[1]s` not found",
		errorKindUserNotEnrolled:   "User `%[1]s` is not enrolled in class `%[2]s`",
		errorKindDefault:           "User `%[1]s` is registered in `%[2]s`? I didn't find any AC",
	},
	ActionGetUsersInClass: {
		errorKindClassCodeNotFound: "No class was found with class code `%s`",
		errorKindDefault:           "Unexpected error while listing users in class.",
	},
	ActionGetClassesOfUser: {
		errorKindUserNotFound: "User `%s` not found",
		errorKindDefault:      "Unexpected error while listing classes for user.",
	},
	ActionClearClass: {
		errorKindClassCodeNotFound: "Class code `%s` not found",
		errorKindDefault:           "Unexpected error while clearing class data.",
	},
	ActionCreateClass: {
		errorKindConflict: "Class code `%s` already exists",
		errorKindDefault:  "Unexpected error while creating class.",
	},
	ActionExterminateUser: {
		errorKindUserNotFound: "User <@%s> not found in the system",
		errorKindDefault:      "Unexpected error while exterminating user.",
	},
	ActionExportClass: {
		errorKindClassCodeNotFound: "Class code `%s` not found",
		errorKindDefault:           "Unexpected error while retrieving class data.",
	},
	ActionMentorRegisterUser: {
		errorKindClassCodeNotFound: "Class code `%[2]s` not found for user `%[1]s`",
		errorKindUserNotFound:      "User `%[1]s` not found",
		errorKindDefault:           "Unexpected error while mentor-registering user.",
	},
	ActionMentorDeregisterUser: {
		errorKindClassCodeNotFound: "Class code `%[2]s` not found for user `%[1]s`",
		errorKindUserNotEnrolled:   "User `%[1]s` is not enrolled in class `%[2]s`",
		errorKindUserNotFound:      "User `%[1]s` not found",
		errorKindDefault:           "Unexpected error while mentor-deregistering user.",
	},
	ActionMentorAddAC: {
		errorKindProblemNotFound: "Problem id `%[2]d` not found for user `%[1]s`",
		errorKindUserNotEnrolled: "User `%[1]s` is not enrolled in the class of the problem `%[2]d`",
		errorKindUserNotFound:    "User `%[1]s` not found",
		errorKindDefault:         "Unexpected error while adding AC.",
	},
	ActionMentorUndoAC: {
		errorKindProblemNotFound: "Problem id `%[2]d` not found for user `%[1]s`",
		errorKindUserNotFound:    "User `%[1]s` not found",
		errorKindConflict:        "User `%[1]s` has not solved problem `%[2]d` yet or is not enrolled in the class",
		errorKindDefault:         "Unexpected error while undoing AC.",
	},
	ActionMentorAddProblem: {
		errorKindClassCodeNotFound: "Class code `%[1]s` not found while adding `%[3]s`.",
		errorKindDuplicateProblem:  "Problem `%[3]s` already exists on class `%[1]s` (with URL `%[2]s`)",
		errorKindConflict:          "Problem `%[3]s` already exists on class `%[1]s` with URL `%[2]s`",
		errorKindDefault:           "Unexpected error while adding problem.",
	},
	ActionMentorRemoveProblem: {
		errorKindProblemNotFound: "Problem id `%d` not found",
		errorKindDefault:         "Unexpected error while removing problem.",
	},
	ActionMentorUpdateProblemClass: {
		errorKindClassCodeNotFound: "Class code `%s` not found",
		errorKindDefault:           "Unexpected error while moving problem to class.",
	},
	ActionMentorUpdateProblem: {
		errorKindProblemNotFound:  "Problem with ID: `%[1]d` not found.",
		errorKindDuplicateProblem: "Problem `%[3]s` already exists in class `%[2]s`",
		errorKindDefault:          "Unexpected error while updating problem.",
	},
	ActionEnrolledDeregisterUser: {
		errorKindClassCodeNotFound: "Class code `%[2]s` not found for user `%[1]s`.",
		errorKindUserNotEnrolled:   "User `%[1]s` is not registered in class `%[2]s`.",
		errorKindDefault:           "Unexpected error while deregistering from class.",
	},
	ActionEnrolledProposeAC: {
		errorKindProblemNotFound: "Problem `%[2]d` not found for user `%[1]s`",
		errorKindUserNotEnrolled: "User `%[1]s` is not enrolled in the class of the problem `%[2]d`",
		errorKindConflict:        "User `%s` has already solved problem `%d`",
		errorKindDefault:         "Unexpected error while proposing AC.",
	},
	ActionEnrolledUndoAC: {
		errorKindProblemNotFound: "Problem `%[2]d` not found",
		errorKindConflict:        "Bro `%[1]s`, you haven't solved problem `%[2]d` yet, so you can't undo it",
		errorKindDefault:         "Unexpected error while undoing AC.",
	},
}

// MapError maps service-layer errors to user-friendly messages by action.
func MapError(err error, action Action, args ...any) string {
	if err == nil {
		return defaultFallbackMessage
	}

	kind := getErrorKindFromError(err)
	if kind == errorKindInternal {
		return internalHiddenMessage
	}

	if msg, ok := fromTemplate(action, kind, args...); ok {
		return msg
	}

	// If no specific template was found for this error kind, try the action's default template.
	if msg, ok := fromTemplate(action, errorKindDefault, args...); ok {
		return msg
	}

	return defaultFallbackMessage
}

func getErrorKindFromError(err error) ErrorKind {
	var userNotEnrolledErr *serviceerr.UserNotEnrolledError
	var userNotFoundErr *serviceerr.UserNotFoundError
	var classCodeNotFoundErr *serviceerr.ClassCodeNotFoundError
	var problemNotFoundErr *serviceerr.ProblemNotFoundError
	var duplicateProblemErr *serviceerr.DuplicateProblemError
	var conflictErr *serviceerr.ConflictError
	var notFoundErr *serviceerr.NotFoundInfoError
	var internalErr *serviceerr.InternalError

	switch {
	case errors.As(err, &internalErr):
		// Prioritize internal errors to avoid leaking sensitive info from wrapped errors
		return errorKindInternal
	case errors.As(err, &userNotEnrolledErr):
		return errorKindUserNotEnrolled
	case errors.As(err, &userNotFoundErr):
		return errorKindUserNotFound
	case errors.As(err, &classCodeNotFoundErr):
		return errorKindClassCodeNotFound
	case errors.As(err, &problemNotFoundErr):
		return errorKindProblemNotFound
	case errors.As(err, &duplicateProblemErr):
		return errorKindDuplicateProblem
	case errors.As(err, &conflictErr):
		return errorKindConflict
	case errors.As(err, &notFoundErr):
		return errorKindNotFound
	default:
		return errorKindDefault
	}
}

func fromTemplate(action Action, kind ErrorKind, args ...any) (string, bool) {
	// First-level lookup
	byType, ok := templates[action]
	if !ok {
		return "", false
	}

	// Second-level lookup
	// If no template is configured for this pair, caller should try another fallback.
	tpl, ok := byType[kind]
	if !ok || strings.TrimSpace(tpl) == "" {
		return "", false
	}

	// Keep plain templates untouched. Format only when placeholders are present.
	formatted := tpl
	if strings.Contains(tpl, "%") {
		formatted = fmt.Sprintf(tpl, args...)
	}

	// fmt embeds "%!" markers when args are missing or have the wrong type.
	// Treat that as a rendering failure and let MapError fall back to default message.
	if strings.Contains(formatted, "%!") {
		return "", false
	}

	return formatted, true
}
