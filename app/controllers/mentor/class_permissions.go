package mentor

import (
	"errors"
	"fmt"
	"strings"

	"github.com/JorgeIba/Hurion/app/context"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
)

// mentorTarget names a new problem's class, or an existing problem and optional
// destination. Existing targets are always resolved from storage, never a room
// or the class identity cached in a preview.
type mentorTarget struct {
	problemID int64
	classCode string
}

func authorizeMentorTargets(validator providers.DiscordValidator, svcs *services.Services, guildID, userID string, targets ...mentorTarget) error {
	scoped, ok := validator.(providers.ClassMentorValidator)
	if !ok {
		// The caller's legacy global mentor/admin gate already ran.
		return nil
	}
	checked := make(map[string]bool)
	check := func(code string) error {
		code = strings.TrimSpace(code)
		if checked[code] {
			return nil
		}
		allowed, err := scoped.CanManageClass(guildID, userID, code)
		if err != nil {
			return errors.New(messageErrVerifying)
		}
		if !allowed {
			return fmt.Errorf("The mentor or admin role for class `%s` is required. No changes were applied.", code)
		}
		checked[code] = true
		return nil
	}
	for _, target := range targets {
		if target.problemID != 0 {
			problem, err := svcs.Problem.GetProblemByID(target.problemID)
			if err != nil {
				return errors.New("Could not resolve the problem's class. No changes were applied.")
			}
			class, err := svcs.Class.GetClassByClassID(problem.ClassID)
			if err != nil {
				return errors.New(messageErrVerifying)
			}
			if err := check(class.ClassCode); err != nil {
				return err
			}
		}
		if target.classCode != "" {
			if err := check(target.classCode); err != nil {
				return err
			}
		}
	}
	return nil
}

func requireMessageTargets(c *context.ContextMessage, targets ...mentorTarget) error {
	err := authorizeMentorTargets(c.DiscordValidator, c.Services, c.ExecutorGuildID, c.Executor.UserID, targets...)
	if err != nil {
		c.Dispatcher.Send(dispatcher.MakeSimpleMessage(c.Message.ChannelID, err.Error()))
	}
	return err
}

func authorizeProposal(validator providers.DiscordValidator, svcs *services.Services, guildID, userID, referenceID string) error {
	if _, ok := validator.(providers.ClassMentorValidator); !ok {
		return nil
	}
	pending, err := svcs.Solution.GetPendingConfirmationByMessageID(referenceID)
	if err != nil {
		return errors.New(messageACProposalUnavailable)
	}
	return authorizeMentorTargets(validator, svcs, guildID, userID, mentorTarget{problemID: pending.ProblemID})
}

func problemBatchTargets(batch []service_model.PreparedProblem) []mentorTarget {
	targets := make([]mentorTarget, len(batch))
	for i, entry := range batch {
		targets[i].classCode = entry.Params.ClassCode
	}
	return targets
}

func problemEditBatchTargets(batch []service_model.PreparedProblemPatch) []mentorTarget {
	targets := make([]mentorTarget, len(batch))
	for i, entry := range batch {
		targets[i].problemID = entry.Params.ProblemID
		if entry.Params.ClassCode.Set {
			targets[i].classCode = entry.Params.ClassCode.Value
		}
	}
	return targets
}

func acBatchTargets(batch []service_model.PreparedAC) []mentorTarget {
	targets := make([]mentorTarget, len(batch))
	for i, entry := range batch {
		targets[i].problemID = entry.Problem.ID
	}
	return targets
}
