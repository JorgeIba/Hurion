package bot

import (
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/controllers/mentor"
)

type ReactionRouter func(c *context.ContextMessage)

const REACTION_YES = "✅"
const REACTION_NO = "❌"

var reactionRouterMap = map[string]CommandRouter{
	REACTION_NO:  routerPendingConfirmationAC,
	REACTION_YES: routerPendingConfirmationAC,
}

func routerPendingConfirmationAC(c *context.ContextMessage) {
	mentor.HandlePendingConfirmation(c)
}
