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

	// ActionSentinel must stay as the last Action constant.
	// It is used as an exclusive upper bound when iterating all actions.
	ActionSentinel
)

const (
	errorKindConflict ErrorKind = iota
	errorKindNotFound
	errorKindInternal
	errorKindDefault

	defaultFallbackMessage = "Unexpected error."
	internalHiddenMessage  = "Unexpected internal error bro."
)

var templates = map[Action]map[ErrorKind]string{
	ActionBindUser: {
		errorKindConflict: "Username `%s` is already taken by another user. Please choose a different username.",
		errorKindDefault:  "Unexpected error while binding your username.",
	},
	ActionRegisterInClass: {
		errorKindNotFound: "Class code `%s` not found or user `%s` not binded yet",
		errorKindConflict: "User `%s` already registered on class `%s`",
		errorKindDefault:  "Unexpected error while registering in class.",
	},
	ActionListProblemsByClass: {
		errorKindNotFound: "Class code `%s` not found",
		errorKindDefault:  "Unexpected error while listing class problems.",
	},
	ActionScoreByClass: {
		errorKindNotFound: "Class code %s doesn't match with any class",
		errorKindDefault:  "Unexpected error while getting class scores.",
	},
	ActionScoreByUserAndClass: {
		errorKindNotFound: "Not found info for class code `%[2]s` or user name `%[1]s`",
		errorKindDefault:  "User `%[1]s` is registered in `%[2]s`? I didn't find any AC",
	},
	ActionGetUsersInClass: {
		errorKindNotFound: "No class was found with class code %s",
		errorKindDefault:  "Unexpected error while listing users in class.",
	},
	ActionGetClassesOfUsername: {
		errorKindNotFound: "Username `%s` not found",
		errorKindDefault:  "Unexpected error while listing classes for username.",
	},
	ActionGetMyUsername: {
		errorKindNotFound: "You are not binded to any username. Use `!bind <username>` to bind your account.",
		errorKindDefault:  "Unexpected error while fetching your username.",
	},
	ActionChangeUsername: {
		errorKindConflict: "Username `%s` is already taken by another user. Please choose a different username.",
		errorKindDefault:  "Unexpected error while changing username.",
	},
	ActionClearClass: {
		errorKindNotFound: "Class code `%s` not found",
		errorKindDefault:  "Unexpected error while clearing class data.",
	},
	ActionCreateClass: {
		errorKindConflict: "Class code `%s` already exists",
		errorKindDefault:  "Unexpected error while creating class.",
	},
	ActionExterminateUser: {
		errorKindNotFound: "User <@%s> not found in the system",
		errorKindDefault:  "Unexpected error while exterminating user.",
	},
	ActionExportClass: {
		errorKindNotFound: "Class code `%s` not found",
		errorKindDefault:  "Unexpected error while retrieving class data.",
	},
	ActionMentorRegisterUser: {
		errorKindNotFound: "Username `%s` or class code `%s` not found",
		errorKindDefault:  "Unexpected error while mentor-registering user.",
	},
	ActionMentorDeregisterUser: {
		errorKindNotFound: "Username `%s` or class code `%s` not found",
		errorKindDefault:  "Unexpected error while mentor-deregistering user.",
	},
	ActionMentorAddAC: {
		errorKindNotFound: "Username `%s` or problem id `%d` not found, probably you are not binded or registered in the same class problem lives",
		errorKindDefault:  "Unexpected error while adding AC.",
	},
	ActionMentorUndoAC: {
		errorKindNotFound: "Username `%s` or problem id `%d` not found",
		errorKindConflict: "User `%s` has not solved problem `%d` yet or is not enrolled in the class",
		errorKindDefault:  "Unexpected error while undoing AC.",
	},
	ActionMentorAddProblem: {
		errorKindNotFound: "Class code `%[2]s` not found",
		errorKindConflict: "Problem `%[1]s` already exists on class `%[2]s` with the ID: `%[3]d`",
		errorKindDefault:  "Unexpected error while adding problem.",
	},
	ActionMentorRemoveProblem: {
		errorKindNotFound: "Problem id `%d` not found",
		errorKindDefault:  "Unexpected error while removing problem.",
	},
	ActionMentorUpdateProblemClass: {
		errorKindNotFound: "Class code `%s` not found",
		errorKindDefault:  "Unexpected error while moving problem to class.",
	},
	ActionMentorUpdateProblem: {
		errorKindNotFound: "Problem with ID: `%d` not found",
		errorKindDefault:  "Unexpected error while updating problem.",
	},
	ActionEnrolledDeregisterUser: {
		errorKindNotFound: "User `%s` is not registered in class `%s` or class doesn't even exist",
		errorKindDefault:  "Unexpected error while deregistering from class.",
	},
	ActionEnrolledProposeAC: {
		errorKindNotFound: "Can you check if user `%s` is binded and registered on the same class of problem `%d`?",
		errorKindConflict: "User `%s` has already solved problem `%d`",
		errorKindDefault:  "Unexpected error while proposing AC.",
	},
	ActionEnrolledUndoAC: {
		errorKindNotFound: "Problem `%[2]d` not found or you're not enrolled in the class",
		errorKindConflict: "Bro `%[1]s`, you haven't solved problem `%[2]d` yet, so you can't undo it",
		errorKindDefault:  "Unexpected error while undoing AC.",
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
	var conflictErr *serviceerr.ConflictError
	var notFoundErr *serviceerr.NotFoundInfoError
	var internalErr *serviceerr.InternalError

	switch {
	case errors.As(err, &internalErr):
		// Prioritize internal errors to avoid leaking sensitive info from wrapped errors
		return errorKindInternal
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
