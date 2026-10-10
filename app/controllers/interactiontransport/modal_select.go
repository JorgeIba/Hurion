package interactiontransport

import (
	"encoding/json"

	"github.com/bwmarrin/discordgo"
)

// modalStringSelect works around the pinned DiscordGo SDK's select serialization:
// SelectMenu always emits "disabled", even when false, but Discord rejects that
// field inside modals. Incoming selections still decode through DiscordGo.
// TODO: Upgrade DiscordGo and replace this type with its native modal select
// once the SDK can omit "disabled"; keep the payload and submission tests.
type modalStringSelect struct {
	CustomID    string
	Placeholder string
	Options     []modalStringSelectOption
	Required    *bool
}

type modalStringSelectOption struct {
	Label   string `json:"label"`
	Value   string `json:"value"`
	Default bool   `json:"default,omitempty"`
}

func (modalStringSelect) Type() discordgo.ComponentType {
	return discordgo.SelectMenuComponent
}

func (menu modalStringSelect) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type        discordgo.ComponentType   `json:"type"`
		CustomID    string                    `json:"custom_id"`
		Placeholder string                    `json:"placeholder,omitempty"`
		Options     []modalStringSelectOption `json:"options"`
		Required    *bool                     `json:"required,omitempty"`
	}{
		Type:        menu.Type(),
		CustomID:    menu.CustomID,
		Placeholder: menu.Placeholder,
		Options:     menu.Options,
		Required:    menu.Required,
	})
}
