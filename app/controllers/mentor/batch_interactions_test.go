package mentor

import (
	"errors"
	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/dispatcher"
	"github.com/JorgeIba/Hurion/app/providers/mock_providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// Record actual dispatcher delivery, including the initial acknowledgement.
type batchInteractionClient struct {
	mu               sync.Mutex
	responses        []*discordgo.InteractionResponse
	edits            []*discordgo.WebhookEdit
	followups        []*discordgo.WebhookParams
	ackErr           error
	deletedResponses []*discordgo.Interaction
}

func (client *batchInteractionClient) InteractionRespond(_ *discordgo.Interaction, response *discordgo.InteractionResponse, _ ...discordgo.RequestOption) error {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.responses = append(client.responses, response)
	return client.ackErr
}
func (client *batchInteractionClient) InteractionResponseDelete(interaction *discordgo.Interaction, _ ...discordgo.RequestOption) error {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.deletedResponses = append(client.deletedResponses, interaction)
	return nil
}

func (client *batchInteractionClient) InteractionResponseEdit(_ *discordgo.Interaction, response *discordgo.WebhookEdit, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.edits = append(client.edits, response)
	return nil, nil
}

func (client *batchInteractionClient) FollowupMessageCreate(_ *discordgo.Interaction, _ bool, response *discordgo.WebhookParams, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.followups = append(client.followups, response)
	return nil, nil
}

type batchFileTransport func(*http.Request) (*http.Response, error)

func (transport batchFileTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestBatchSubmitReadsFilesAndStopsOnTransportOrPermissionErrors(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, route, body, want, text string
		status, size                  int
		ackFails, denied              bool
		permissionErr                 error
		wantDownload                  bool
	}{
		{name: "problem file", route: string(discordcommands.ProblemBatch), body: `[{"class":"basicos","name":"Two Sum","url":"https://example.com","topic":"Arrays"}]`, status: 200, want: "Entries: 1", wantDownload: true},
		{name: "file wins over invalid pasted text", text: "invalid JSON", body: `[{"user_id":"123","problem_id":42}]`, status: 200, want: "Entries: 1", wantDownload: true},
		{name: "malformed file does not fall back to valid text", text: `[{"user_id":"123","problem_id":42}]`, body: `[{`, status: 200, want: "Could not read JSON", wantDownload: true},
		{name: "failed download does not fall back to valid text", text: `[{"user_id":"123","problem_id":42}]`, body: `[]`, status: 403, want: "HTTP 403", wantDownload: true},
		{name: "AC file", body: `[{"user_id":"123","problem_id":42},{"user_id":"123","problem_id":42}]`, status: 200, want: "Entries: 2", wantDownload: true},
		{name: "malformed JSON file", body: `[{`, status: 200, want: "line 1, column", wantDownload: true},
		{name: "strict entry decoding", body: `[{"extra":1}]`, status: 200, want: "Entry 1", wantDownload: true},
		{name: "invalid UTF8", body: "[\"\xff\"]", status: 200, want: "Could not read JSON", wantDownload: true},
		{name: "expired attachment", body: `[]`, status: 403, want: "HTTP 403", wantDownload: true},
		{name: "oversized file", size: maxJSONFileBytes + 1, want: "5 MiB"},
		{name: "empty file", want: "non-empty"},
		{name: "failed acknowledgement", ackFails: true},
		{name: "failed problem acknowledgement", route: string(discordcommands.ProblemBatch), ackFails: true},
		{name: "permission revoked", denied: true, want: "mentor or admin"},
		{name: "problem permission revoked", route: string(discordcommands.ProblemBatch), denied: true, want: "mentor or admin"},
		{name: "permission lookup failure", permissionErr: errors.New("Discord unavailable"), want: "verifying permissions"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			mockCtrl := gomock.NewController(t)
			validator := mock_providers.NewMockDiscordValidator(mockCtrl)
			fetcher := mock_providers.NewMockDiscordFetcher(mockCtrl)
			store := mock_data.NewMockDataStore(mockCtrl)
			svcs := services.NewServices(store)
			if strings.HasPrefix(test.want, "Entries:") {
				if test.route == string(discordcommands.ProblemBatch) {
					store.EXPECT().GetClassesByClassCodes([]string{"basicos"}).Return([]data.Class{{ID: 10, ClassCode: "basicos"}}, nil)
				} else {
					fetcher.EXPECT().GetCachedMember("guild", "123").Return(&discordgo.Member{User: &discordgo.User{ID: "123"}, Roles: []string{"student"}}, nil)
					store.EXPECT().GetACValidationData(gomock.Any()).Return(map[data.ACKey]data.ACValidationData{{UserID: "123", ProblemID: 42}: {Problem: &data.Problem{ID: 42, ClassID: 10}, Class: &data.Class{ID: 10, ClassCode: "basicos"}, AccessRoleIDs: []string{"student"}}}, nil)

				}
			}
			if !test.ackFails {
				validator.EXPECT().IsMentor("guild", "mentor").Return(!test.denied, test.permissionErr)
				if test.denied {
					validator.EXPECT().IsAdmin("guild", "mentor").Return(false, nil)
				}
			}
			client := &batchInteractionClient{}
			if test.ackFails {
				client.ackErr = errors.New("ack rejected")
			}
			downloads := 0
			reader := newJSONTransport(&http.Client{Transport: batchFileTransport(func(request *http.Request) (*http.Response, error) {
				downloads++
				assert.Len(t, client.responses, 1, "acknowledge before downloading")
				return &http.Response{StatusCode: test.status, Body: io.NopCloser(strings.NewReader(test.body)), Header: make(http.Header), Request: request}, nil
			})})
			route := test.route
			if route == "" {
				route = string(discordcommands.ACBatch)
			}
			size := test.size
			if size == 0 {
				size = len(test.body)
			}
			event := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionModalSubmit, GuildID: "guild", Member: &discordgo.Member{User: &discordgo.User{ID: "mentor"}},
				Data: discordgo.ModalSubmitInteractionData{
					CustomID: route,
					Components: []discordgo.MessageComponent{
						&discordgo.Label{Component: &discordgo.TextInput{CustomID: "json", Value: test.text}},
						&discordgo.Label{Component: &discordgo.FileUpload{CustomID: "file", Values: []string{"42"}}},
					},
					Resolved: discordgo.ComponentInteractionDataResolved{Attachments: map[string]*discordgo.MessageAttachment{"42": {ID: "42", Filename: "batch.json", URL: "https://cdn.discordapp.com/ephemeral-attachments/1/42/batch.json?ex=123", Size: size}}},
				},
			}}
			responder := dispatcher.NewDiscordInteractionResponder(client)
			if route == string(discordcommands.ProblemBatch) {
				controller := NewProblemInteractionController(svcs, validator, responder)
				controller.attachments = reader
				controller.HandleCreateProblemBatchModalSubmitInteraction(event)
			} else {
				controller := NewACInteractionController(svcs, validator, fetcher, responder)
				controller.attachments = reader
				controller.HandleCreateACBatchModalSubmitInteraction(event)
			}
			if assert.Len(t, client.responses, 1) {
				assert.Equal(t, discordgo.InteractionResponseDeferredChannelMessageWithSource, client.responses[0].Type)
				assert.Equal(t, discordgo.MessageFlagsEphemeral, client.responses[0].Data.Flags)
			}
			if test.ackFails {
				assert.Empty(t, client.edits)
			} else if assert.Len(t, client.edits, 1) {
				assert.Contains(t, *client.edits[0].Content, test.want)
				assert.Empty(t, client.edits[0].AllowedMentions.Parse)
				if strings.HasPrefix(test.want, "Entries:") {
					assert.Contains(t, *client.edits[0].Content, "batch validated")
					assert.Contains(t, *client.edits[0].Content, "No changes were applied")
				}
			}
			if test.wantDownload {
				assert.Equal(t, 1, downloads)
			} else {
				assert.Zero(t, downloads)
			}
		})
	}
}
