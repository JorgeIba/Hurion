package adapters

import (
	"errors"
	"fmt"
	"strings"

	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
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
	ActionExterminateClass
	ActionCreateClass
	ActionExterminateUser
	ActionExportClass
	ActionGetClassRoles
	ActionGetClassRooms

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
	errorKindUserMissingFromDB
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
		errorKindNotFound:          "I couldn't identify the class, bro. Try specifying the class code (e.g., `!list basicos_1`) or run this command inside a class channel.",
		errorKindDefault:           "Unexpected error while listing class problems.",
	},
	ActionScoreByClass: {
		errorKindClassCodeNotFound: "Class code `%s` doesn't match with any class",
		errorKindNotFound:          "I couldn't identify the class, bro. Try specifying the class code (e.g., `!score basicos_1`) or run this command inside a class channel.",
		errorKindDefault:           "Unexpected error while getting class scores.",
	},
	ActionScoreByUserAndClass: {
		errorKindClassCodeNotFound: "Class code `%[2]s` not found",
		errorKindUserNotFound:      "User `%[1]s` not found",
		errorKindUserNotEnrolled:   "Bro, `%[1]s` doesn't have any solved problems in class `%[2]s` yet.",
		errorKindUserMissingFromDB: "System Error: User `%[1]s` not found in database. Contact Admin.",
		errorKindNotFound:          "I couldn't identify the class, bro. Try specifying the class code (e.g., `!score_of @user basicos_1`) or run this command inside a class channel.",
		errorKindDefault:           "User `%[1]s` has access to `%[2]s`? I didn't find any AC",
	},
	ActionGetUsersInClass: {
		errorKindClassCodeNotFound: "No class was found with class code `%s`",
		errorKindDefault:           "Unexpected error while listing users in class.",
	},
	ActionGetClassesOfUser: {
		errorKindUserNotFound:      "User `%s` not found",
		errorKindUserMissingFromDB: "System Error: User `%s` not found in database. Contact Admin.",
		errorKindDefault:           "Unexpected error while listing classes for user.",
	},
	ActionClearClass: {
		errorKindClassCodeNotFound: "Class code `%s` not found",
		errorKindNotFound:          "I couldn't identify the class, bro. Try specifying the class code (e.g., `!clear_class basicos_1`) or run this command inside a class channel.",
		errorKindDefault:           "Unexpected error while clearing class data.",
	},
	ActionExterminateClass: {
		errorKindClassCodeNotFound: "Class code `%s` not found",
		errorKindDefault:           "Unexpected error while exterminating class.",
	},
	ActionCreateClass: {
		errorKindConflict: "Class code `%s` already exists",
		errorKindDefault:  "Unexpected error while creating class.",
	},
	ActionExterminateUser: {
		errorKindUserNotFound:      "User <@%s> not found in the system",
		errorKindUserMissingFromDB: "System Error: User <@%s> not found in database. Contact Admin.",
		errorKindDefault:           "Unexpected error while exterminating user.",
	},
	ActionExportClass: {
		errorKindClassCodeNotFound: "Class code `%s` not found",
		errorKindDefault:           "Unexpected error while retrieving class data.",
	},
	ActionGetClassRoles: {
		errorKindClassCodeNotFound: "Class code `%s` not found",
		errorKindNotFound:          "I couldn't identify the class, bro. Try specifying the class code (e.g., `!get_roles basicos_1`) or run this command inside a class channel.",
		errorKindDefault:           "Unexpected error while getting class roles.",
	},
	ActionGetClassRooms: {
		errorKindClassCodeNotFound: "Class code `%s` not found",
		errorKindNotFound:          "I couldn't identify the class, bro. Try specifying the class code (e.g., `!get_channels basicos_1`) or run this command inside a class channel.",
		errorKindDefault:           "Unexpected error while getting class rooms.",
	},
	ActionMentorRegisterUser: {
		errorKindClassCodeNotFound: "Class code `%[2]s` not found for user `%[1]s`",
		errorKindUserNotFound:      "User `%[1]s` not found",
		errorKindUserMissingFromDB: "System Error: User `%[1]s` not found in database. Contact Admin.",
		errorKindDefault:           "Unexpected error while mentor-registering user.",
	},
	ActionMentorDeregisterUser: {
		errorKindClassCodeNotFound: "Class code `%[2]s` not found for user `%[1]s`",
		errorKindUserNotEnrolled:   "User `%[1]s` doesn't have access to class `%[2]s`",
		errorKindUserNotFound:      "User `%[1]s` not found",
		errorKindUserMissingFromDB: "System Error: User `%[1]s` not found in database. Contact Admin.",
		errorKindDefault:           "Unexpected error while mentor-deregistering user.",
	},
	ActionMentorAddAC: {
		errorKindProblemNotFound:   "Problem id `%[2]d` not found for user `%[1]s`",
		errorKindUserNotEnrolled:   "User `%[1]s` doesn't have access to the class of problem `%[2]d`",
		errorKindUserNotFound:      "User `%[1]s` not found",
		errorKindUserMissingFromDB: "System Error: User `%[1]s` not found in database. Contact Admin.",
		errorKindConflict:          "User `%[1]s` has already solved problem `%[2]d`",
		errorKindDefault:           "Unexpected error while adding AC for user `%[1]s` (problem `%[2]d`).",
	},
	ActionMentorUndoAC: {
		errorKindProblemNotFound:   "Problem id `%[2]d` not found for user `%[1]s`",
		errorKindUserNotFound:      "User `%[1]s` not found",
		errorKindUserMissingFromDB: "System Error: User `%[1]s` not found in database. Contact Admin.",
		errorKindConflict:          "User `%[1]s` has not solved problem `%[2]d` yet",
		errorKindDefault:           "Unexpected error while undoing AC for user `%[1]s` (problem `%[2]d`).",
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
		errorKindProblemNotFound:   "Problem with ID: `%[1]d` not found.",
		errorKindClassCodeNotFound: "Class code `%[2]s` not found while updating problem `%[1]d`.",
		errorKindDuplicateProblem:  "Problem `%[3]s` already exists in class `%[2]s`",
		errorKindDefault:           "Unexpected error while updating problem.",
	},
	ActionEnrolledDeregisterUser: {
		errorKindClassCodeNotFound: "Class code `%[2]s` not found for user `%[1]s`.",
		errorKindUserNotEnrolled:   "User `%[1]s` doesn't have access to class `%[2]s`.",
		errorKindUserMissingFromDB: "System Error: User `%[1]s` not found in database. Contact Admin.",
		errorKindDefault:           "Unexpected error while deregistering from class.",
	},
	ActionEnrolledProposeAC: {
		errorKindProblemNotFound:   "Problem `%[2]d` not found for user `%[1]s`",
		errorKindUserNotEnrolled:   "User `%[1]s` doesn't have access to the class of problem `%[2]d`",
		errorKindUserMissingFromDB: "System Error: User `%[1]s` not found in database. Contact Admin.",
		errorKindConflict:          "User `%[1]s` has already solved problem `%[2]d`",
		errorKindDefault:           "Unexpected error while proposing AC.",
	},
	ActionEnrolledUndoAC: {
		errorKindProblemNotFound:   "Problem `%[2]d` not found.",
		errorKindUserNotEnrolled:   "User `%[1]s` doesn't have access to the class of problem `%[2]d`",
		errorKindUserMissingFromDB: "System Error: User `%[1]s` not found in database. Contact Admin.",
		errorKindConflict:          "Bro `%[1]s`, you haven't solved problem `%[2]d` yet",
		errorKindDefault:           "Unexpected error while undoing AC.",
	},
}

// MapError maps service-layer errors to user-friendly messages by action.
func MapError(err error, action Action, args ...any) string {
	if err == nil {
		return defaultFallbackMessage
	}

	if validation, ok := errors.AsType[*serviceerr.ValidationError](err); ok {
		reason := validation.Message
		if validation.WrappedError != nil {
			reason = validation.WrappedError.Error()
		}
		return "Please correct the values: " + reason
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
	switch {
	case hasErrorType[*dataerr.UserMissingFromDatabaseError](err):
		return errorKindUserMissingFromDB
	case hasErrorType[*serviceerr.UserNotEnrolledError](err):
		return errorKindUserNotEnrolled
	case hasErrorType[*serviceerr.UserNotFoundError](err):
		return errorKindUserNotFound
	case hasErrorType[*serviceerr.ClassCodeNotFoundError](err):
		return errorKindClassCodeNotFound
	case hasErrorType[*serviceerr.ProblemNotFoundError](err):
		return errorKindProblemNotFound
	case hasErrorType[*serviceerr.DuplicateProblemError](err):
		return errorKindDuplicateProblem
	case hasErrorType[*serviceerr.ConflictError](err):
		return errorKindConflict
	case hasErrorType[*serviceerr.NotFoundInfoError](err):
		return errorKindNotFound
	case hasErrorType[*serviceerr.InternalError](err):
		// InternalError must stay at the end because it unwraps and could shadow
		// more specific wrapped errors if checked first.
		return errorKindInternal
	default:
		return errorKindDefault
	}
}

func hasErrorType[E error](err error) bool {
	_, ok := errors.AsType[E](err)
	return ok
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
