package mentor

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/JorgeIba/Hurion/app/customerr/controllerserr"
	"github.com/JorgeIba/Hurion/app/customerr/serviceerr"
	"github.com/JorgeIba/Hurion/app/data"
	"github.com/JorgeIba/Hurion/app/data/mock_data"
	"github.com/JorgeIba/Hurion/app/providers/mock_providers"
	"github.com/JorgeIba/Hurion/app/services"
	"github.com/JorgeIba/Hurion/app/services/service_model"
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestBatchPreparationRejectsMissingNullAndEmptyRequiredJSONValues(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, input string
		problems    bool
	}{
		{name: "missing problem values", input: `[{}]`, problems: true},
		{name: "null problem values", input: `[{"class":null,"name":null,"url":null}]`, problems: true},
		{name: "empty problem values", input: `[{"class":"","name":"","url":""}]`, problems: true},
		{name: "missing AC values", input: `[{}]`},
		{name: "null AC values", input: `[{"user_id":null,"problem_id":null}]`},
		{name: "empty AC values", input: `[{"user_id":"","problem_id":0}]`},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			validator := mock_providers.NewMockDiscordValidator(ctrl)
			validator.EXPECT().IsMentor("guild", "mentor").Return(true, nil)
			preparer := batchPreparer{batch: services.NewServices(mock_data.NewMockDataStore(ctrl)).Batch, validator: validator, fetcher: mock_providers.NewMockDiscordFetcher(ctrl)}
			if test.problems {
				entries, parseIssues := parseProblemBatch(test.input)
				assert.Empty(t, parseIssues)
				prepared, issues, err := preparer.prepareProblemSubmission("guild", "mentor", entries)
				assert.NoError(t, err)
				assert.Nil(t, prepared)
				if assert.Len(t, issues, 1) {
					assert.Equal(t, 1, issues[0].Entry)
					assert.IsType(t, &serviceerr.ValidationError{}, issues[0].Err)
				}
			} else {
				entries, parseIssues := parseACBatch(test.input)
				assert.Empty(t, parseIssues)
				prepared, issues, err := preparer.prepareACSubmission("guild", "mentor", entries)
				assert.NoError(t, err)
				assert.Nil(t, prepared)
				if assert.Len(t, issues, 1) {
					assert.Equal(t, 1, issues[0].Entry)
					assert.IsType(t, &serviceerr.ValidationError{}, issues[0].Err)
				}
			}
		})
	}
}

func TestBatchPreparationChecksActorBeforeTargetLookups(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, guild, actor string
		problems, admin    bool
		roleErr            error
	}{
		{name: "no guild", actor: "mentor", problems: true},
		{name: "no actor", guild: "guild"},
		{name: "problem management denied", guild: "guild", actor: "actor", problems: true},
		{name: "mentor lookup failed", guild: "guild", actor: "actor", problems: true, roleErr: errors.New("Discord unavailable")},
		{name: "AC management denied", guild: "guild", actor: "actor"},
		{name: "admin can grant AC", guild: "guild", actor: "actor", admin: true},
		{name: "admin can manage problems", guild: "guild", actor: "actor", problems: true, admin: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			validator := mock_providers.NewMockDiscordValidator(ctrl)
			if test.guild != "" && test.actor != "" {
				validator.EXPECT().IsMentor(test.guild, test.actor).Return(false, test.roleErr)
				if test.roleErr == nil {
					validator.EXPECT().IsAdmin(test.guild, test.actor).Return(test.admin, nil)
				}
			}
			preparer := batchPreparer{batch: services.NewServices(mock_data.NewMockDataStore(ctrl)).Batch, validator: validator, fetcher: mock_providers.NewMockDiscordFetcher(ctrl)}
			var issues []service_model.BatchEntryError
			var err error
			if test.problems {
				_, issues, err = preparer.prepareProblemSubmission(test.guild, test.actor, []problemBatchEntry{{entry: 1}})
			} else {
				_, issues, err = preparer.prepareACSubmission(test.guild, test.actor, []acBatchEntry{{entry: 1}})
			}
			if test.admin {
				assert.NoError(t, err)
				if assert.Len(t, issues, 1) {
					assert.Equal(t, 1, issues[0].Entry)
					assert.IsType(t, &serviceerr.ValidationError{}, issues[0].Err)
				}
			} else {
				assert.Error(t, err)
				assert.Empty(t, issues)
			}
		})
	}
}

func TestACPreparationFallsBackToLiveMembersAndFailsClosed(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		member   *discordgo.Member
		fetchErr error
		wantType error
	}{
		{name: "current student role", member: &discordgo.Member{User: &discordgo.User{ID: "123"}, Nick: "Ana", Roles: []string{"student"}}},
		{name: "current roles deny access", member: &discordgo.Member{User: &discordgo.User{ID: "123"}, Roles: []string{"other"}}, wantType: &serviceerr.UserNotEnrolledError{}},
		{name: "member missing", wantType: &controllerserr.InternalError{}},
		{name: "member has no user", member: &discordgo.Member{}, wantType: &controllerserr.InternalError{}},
		{name: "wrong identity", member: &discordgo.Member{User: &discordgo.User{ID: "456"}}, wantType: &controllerserr.InternalError{}},
		{name: "Discord unavailable", fetchErr: errors.New("offline"), wantType: &controllerserr.InternalError{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_data.NewMockDataStore(ctrl)
			validator := mock_providers.NewMockDiscordValidator(ctrl)
			fetcher := mock_providers.NewMockDiscordFetcher(ctrl)
			validator.EXPECT().IsMentor("guild", "mentor").Return(true, nil)
			fetcher.EXPECT().GetCachedMember("guild", "123").Return(nil, nil)
			fetcher.EXPECT().FetchMember("guild", "123").Return(test.member, test.fetchErr)
			if test.member != nil && test.member.User != nil && test.member.User.ID == "123" && test.fetchErr == nil {
				store.EXPECT().GetProblemByID(int64(42)).Return(data.Problem{ID: 42, ClassID: 10}, nil)
				store.EXPECT().GetClassByClassID(int64(10)).Return(data.Class{ID: 10, ClassCode: "basicos"}, nil)
				store.EXPECT().GetClassAccessRolesByClassCode("basicos").Return([]data.ClassAccessRole{{RoleID: "student"}}, nil)
			}
			if test.wantType == nil {
				store.EXPECT().HasUserSolvedProblem("123", int64(42)).Return(false, nil)
			}
			entries, parseIssues := parseACBatch(`[{"user_id":"123","problem_id":42}]`)
			assert.Empty(t, parseIssues)
			preparer := batchPreparer{batch: services.NewServices(store).Batch, validator: validator, fetcher: fetcher}
			prepared, issues, err := preparer.prepareACSubmission("guild", "mentor", entries)
			if _, controllerFailure := test.wantType.(*controllerserr.InternalError); controllerFailure {
				assert.Nil(t, prepared)
				assert.Empty(t, issues)
				var controllerErr *controllerserr.InternalError
				assert.ErrorAs(t, err, &controllerErr)
				if test.fetchErr != nil {
					assert.ErrorIs(t, err, test.fetchErr)
				}
			} else if test.wantType != nil {
				assert.NoError(t, err)
				assert.Nil(t, prepared)
				if assert.Len(t, issues, 1) {
					assert.IsType(t, test.wantType, issues[0].Err)
				}
			} else {
				assert.NoError(t, err)
				assert.Empty(t, issues)
				if assert.Len(t, prepared, 1) {
					assert.Equal(t, "Ana", prepared[0].User.DisplayName)
					assert.Equal(t, "basicos", prepared[0].Class.ClassCode)
				}
			}
		})
	}
}

func TestProblemPreparationAcceptsNullOptionalMetadata(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	store := mock_data.NewMockDataStore(ctrl)
	validator := mock_providers.NewMockDiscordValidator(ctrl)
	validator.EXPECT().IsMentor("guild", "mentor").Return(true, nil)
	store.EXPECT().GetClassByClassCode("basicos").Return(data.Class{ID: 10, ClassCode: "basicos"}, nil)
	entries, parseIssues := parseProblemBatch(`[{"class":" basicos ","name":" Two Sum ","url":" https://example.com/problem ","difficulty":null,"topic":null,"priority":null}]`)
	assert.Empty(t, parseIssues)
	preparer := batchPreparer{batch: services.NewServices(store).Batch, validator: validator}
	prepared, issues, err := preparer.prepareProblemSubmission("guild", "mentor", entries)
	assert.NoError(t, err)
	assert.Empty(t, issues)
	if assert.Len(t, prepared, 1) {
		assert.Equal(t, "Two Sum", prepared[0].Params.Name)
		assert.Equal(t, "basicos", prepared[0].Class.ClassCode)
		assert.Nil(t, prepared[0].Params.Difficulty)
		assert.Nil(t, prepared[0].Params.Topic)
		assert.Nil(t, prepared[0].Params.Priority)
	}
}

func TestACPreparationResolvesEachStudentOnce(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		lookupFails bool
	}{
		{name: "SDK cache hit"},
		{name: "failed lookup is not retried", lookupFails: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_data.NewMockDataStore(ctrl)
			validator := mock_providers.NewMockDiscordValidator(ctrl)
			fetcher := mock_providers.NewMockDiscordFetcher(ctrl)
			validator.EXPECT().IsMentor("guild", "mentor").Return(true, nil)
			if test.lookupFails {
				fetcher.EXPECT().GetCachedMember("guild", "123").Return(nil, errors.New("cache miss"))
				fetcher.EXPECT().FetchMember("guild", "123").Return(nil, errors.New("Discord unavailable"))
			} else {
				fetcher.EXPECT().GetCachedMember("guild", "123").Return(&discordgo.Member{User: &discordgo.User{ID: "123"}, Nick: "Ana", Roles: []string{"student"}}, nil)
				store.EXPECT().GetClassAccessRolesByClassCode("basicos").Return([]data.ClassAccessRole{{RoleID: "student"}}, nil).Times(2)
			}
			if !test.lookupFails {
				for _, id := range []int64{42, 43} {
					store.EXPECT().GetProblemByID(id).Return(data.Problem{ID: id, ClassID: 10}, nil)
					store.EXPECT().HasUserSolvedProblem("123", id).Return(false, nil)
				}
				store.EXPECT().GetClassByClassID(int64(10)).Return(data.Class{ID: 10, ClassCode: "basicos"}, nil).Times(2)
			}
			entries, parseIssues := parseACBatch(`[{"user_id":" 00123 ","problem_id":42},{"user_id":"123","problem_id":43}]`)
			assert.Empty(t, parseIssues)
			preparer := batchPreparer{batch: services.NewServices(store).Batch, validator: validator, fetcher: fetcher}
			prepared, issues, err := preparer.prepareACSubmission("guild", "mentor", entries)
			if test.lookupFails {
				assert.Nil(t, prepared)
				assert.Empty(t, issues)
				var controllerErr *controllerserr.InternalError
				assert.ErrorAs(t, err, &controllerErr)
				assert.Contains(t, err.Error(), "entry 1")
			} else {
				assert.NoError(t, err)
				assert.Empty(t, issues)
				if assert.Len(t, prepared, 2) {
					assert.Equal(t, "Ana", prepared[0].User.DisplayName)
					assert.Equal(t, int64(42), prepared[0].Problem.ID)
					assert.Equal(t, int64(43), prepared[1].Problem.ID)
				}
			}
		})
	}
}

func TestBatchPreparationRejectsInvalidInputBeforeLookups(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		problems  bool
		size      int
		wantEntry int
	}{
		{name: "empty problem batch", problems: true},
		{name: "oversized problem batch", problems: true, size: service_model.MaxBatchOperations + 1},
		{name: "empty AC batch"},
		{name: "oversized AC batch", size: service_model.MaxBatchOperations + 1},
		{name: "invalid AC problem ID", size: 1, wantEntry: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			validator := mock_providers.NewMockDiscordValidator(ctrl)
			validator.EXPECT().IsMentor("guild", "mentor").Return(true, nil)
			fetcher := mock_providers.NewMockDiscordFetcher(ctrl)
			preparer := batchPreparer{batch: services.NewServices(mock_data.NewMockDataStore(ctrl)).Batch, validator: validator, fetcher: fetcher}
			entryJSON := `{"class":"basicos","name":"Valid","url":"https://example.com/problem"}`
			if !test.problems {
				entryJSON = `{"user_id":"123","problem_id":42}`
				if test.wantEntry != 0 {
					entryJSON = `{"user_id":"123","problem_id":0}`
				}
			}
			input := "[]"
			if test.size > 0 {
				input = "[" + strings.Repeat(entryJSON+",", test.size-1) + entryJSON + "]"
			}
			var issues []service_model.BatchEntryError
			var err error
			if test.problems {
				entries, parseIssues := parseProblemBatch(input)
				assert.Empty(t, parseIssues)
				assert.Len(t, entries, test.size)
				prepared, resultIssues, resultErr := preparer.prepareProblemSubmission("guild", "mentor", entries)
				assert.Nil(t, prepared)
				issues, err = resultIssues, resultErr
			} else {
				entries, parseIssues := parseACBatch(input)
				assert.Empty(t, parseIssues)
				assert.Len(t, entries, test.size)
				prepared, resultIssues, resultErr := preparer.prepareACSubmission("guild", "mentor", entries)
				assert.Nil(t, prepared)
				issues, err = resultIssues, resultErr
			}
			assert.NoError(t, err)
			if assert.Len(t, issues, 1) {
				assert.Equal(t, test.wantEntry, issues[0].Entry)
				assert.IsType(t, &serviceerr.ValidationError{}, issues[0].Err)
			}
		})
	}
}

func TestACPreparationRejectsMalformedTargetsBeforeDiscord(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		input       string
		wantEntries []int
	}{
		{name: "non-numeric student", input: `[{"user_id":"Ana","problem_id":42}]`, wantEntries: []int{1}},
		{name: "overflowing student ID", input: `[{"user_id":"18446744073709551616","problem_id":42}]`, wantEntries: []int{1}},
		{name: "negative problem", input: `[{"user_id":"123","problem_id":-1}]`, wantEntries: []int{1}},
		{name: "valid target followed by invalid target", input: `[{"user_id":"123","problem_id":42},{"user_id":"Ana","problem_id":42}]`, wantEntries: []int{2}},
		{name: "all invalid entries are reported", input: `[{"user_id":"Ana","problem_id":42},{"user_id":"123","problem_id":0}]`, wantEntries: []int{1, 2}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			validator := mock_providers.NewMockDiscordValidator(ctrl)
			validator.EXPECT().IsMentor("guild", "mentor").Return(true, nil)
			preparer := batchPreparer{validator: validator, fetcher: mock_providers.NewMockDiscordFetcher(ctrl)}
			entries, parseIssues := parseACBatch(test.input)
			assert.Empty(t, parseIssues)
			prepared, issues, err := preparer.prepareACSubmission("guild", "mentor", entries)
			assert.NoError(t, err)
			assert.Nil(t, prepared)
			if assert.Len(t, issues, len(test.wantEntries)) {
				for index, issue := range issues {
					assert.Equal(t, test.wantEntries[index], issue.Entry)
					assert.IsType(t, &serviceerr.ValidationError{}, issue.Err)
				}
			}
		})
	}
}

func TestACPreparationRejectsOversizedDistinctStudentsBeforeDiscord(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	validator := mock_providers.NewMockDiscordValidator(ctrl)
	validator.EXPECT().IsMentor("guild", "mentor").Return(true, nil)
	preparer := batchPreparer{validator: validator, fetcher: mock_providers.NewMockDiscordFetcher(ctrl)}
	entries := make([]acBatchEntry, service_model.MaxBatchOperations+1)
	for index := range entries {
		entries[index] = acBatchEntry{entry: index + 1, input: acBatchInput{UserID: strconv.Itoa(index + 1), ProblemID: 42}}
	}
	prepared, issues, err := preparer.prepareACSubmission("guild", "mentor", entries)
	assert.NoError(t, err)
	assert.Nil(t, prepared)
	if assert.Len(t, issues, 1) {
		assert.Zero(t, issues[0].Entry)
		assert.IsType(t, &serviceerr.ValidationError{}, issues[0].Err)
	}
}
