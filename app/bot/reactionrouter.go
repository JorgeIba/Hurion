package bot

import (
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/common"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/mentor"
)

type ReactionRouter func(c *context.ContextMessage)

var reactionRouterMap = map[string]CommandRouter{
	common.REACTION_NO:  routerPendingConfirmationAC,
	common.REACTION_YES: routerPendingConfirmationAC,
}

func routerPendingConfirmationAC(c *context.ContextMessage) {
	mentor.HandlePendingConfirmation(c)
}
