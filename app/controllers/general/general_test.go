package general

import (
	"testing"

	"github.com/JorgeIba/Huron-Problem-Checker-Bot/app/context"
	"github.com/bwmarrin/discordgo"
	"github.com/ewohltman/discordgo-mock/mockuser"
)

func TestBindUser_Success(t *testing.T) {
	const userName = "Jorge"
	const discordID = "123"
	const discordNickName = "DaddyYorch"

	mockedUser := mockuser.New()
	mockedUser.ID = discordID
	mockedUser.GlobalName = discordNickName

	ctx := &context.ContextMessage{
		Message: &discordgo.MessageCreate{
			Message: &discordgo.Message{
				Author: discordgo.NewState().User,
			},
		},
		Args: []string{"!bind", userName},
	}

	// Act
	BindUser(ctx)

	// Verify
}
