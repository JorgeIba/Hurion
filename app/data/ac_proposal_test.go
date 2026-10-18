package data

import (
	"testing"

	"github.com/JorgeIba/Hurion/app/customerr/dataerr"
	"github.com/stretchr/testify/require"
)

func proposalFixture(t *testing.T) (*DB, PendingConfirmation) {
	t.Helper()
	db := setupDB(t)
	t.Cleanup(db.CloseDB)
	classID, err := db.CreateClass(Class{ClassCode: "ac", Name: "AC class"})
	require.NoError(t, err)
	problemID, err := db.AddProblem(Problem{Name: "Two Sum", URL: "https://example.com/problem", ClassID: classID})
	require.NoError(t, err)
	require.NoError(t, db.EnsureUser(User{UserID: "student", DisplayName: "Student"}))
	pending := PendingConfirmation{MessageID: "original", UserID: "student", ProblemID: problemID}
	written, err := db.AddPendingConfirmation(pending)
	require.NoError(t, err)
	require.True(t, written)
	return db, pending
}

func TestACProposalRenewalAndApprovalRespectWriteOrder(t *testing.T) {
	for _, approveFirst := range []bool{false, true} {
		name := "renewal wins"
		if approveFirst {
			name = "approval wins"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			db, original := proposalFixture(t)
			renewal := original
			renewal.MessageID = "renewed"
			if approveFirst {
				consumed, err := db.AcceptPendingConfirmation(original.MessageID)
				require.NoError(t, err)
				require.Equal(t, original, consumed)
				written, err := db.AddPendingConfirmation(renewal)
				require.NoError(t, err)
				require.False(t, written, "the final write must reject a proposal for an already recorded AC")
			} else {
				written, err := db.AddPendingConfirmation(renewal)
				require.NoError(t, err)
				require.True(t, written)
				_, err = db.AcceptPendingConfirmation(original.MessageID)
				require.True(t, dataerr.IsNotFound(err), "the replaced reference must not approve the renewal")
				_, err = db.RemovePendingConfirmation(original.MessageID)
				require.True(t, dataerr.IsNotFound(err), "the replaced reference must not reject the renewal")
				consumed, err := db.AcceptPendingConfirmation(renewal.MessageID)
				require.NoError(t, err)
				require.Equal(t, renewal, consumed)
			}
			solved, err := db.HasUserSolvedProblem(original.UserID, original.ProblemID)
			require.NoError(t, err)
			require.True(t, solved)
			_, err = db.RemovePendingConfirmation(renewal.MessageID)
			require.True(t, dataerr.IsNotFound(err))
		})
	}
}

func TestACProposalConsumptionDistinguishesSuccessAbsenceAndFailure(t *testing.T) {
	t.Parallel()
	db, pending := proposalFixture(t)
	consumed, err := db.RemovePendingConfirmation(pending.MessageID)
	require.NoError(t, err)
	require.Equal(t, pending, consumed)
	consumed, err = db.RemovePendingConfirmation(pending.MessageID)
	require.True(t, dataerr.IsNotFound(err))
	require.Empty(t, consumed)
	db.CloseDB()
	consumed, err = db.RemovePendingConfirmation(pending.MessageID)
	require.Error(t, err)
	require.False(t, dataerr.IsNotFound(err))
	require.Empty(t, consumed)
}

func TestACProposalAcceptanceRestoresPendingOnWriteFailure(t *testing.T) {
	t.Parallel()
	db, pending := proposalFixture(t)
	_, err := db.conn.Exec(`CREATE TRIGGER fail_approved_ac BEFORE INSERT ON solved_problems
 BEGIN SELECT RAISE(ABORT, 'simulated AC write failure'); END;`)
	require.NoError(t, err)
	consumed, err := db.AcceptPendingConfirmation(pending.MessageID)
	require.Error(t, err)
	require.False(t, dataerr.IsNotFound(err))
	require.Empty(t, consumed)
	userID, problemID, err := db.GetPendingConfirmation(pending.MessageID)
	require.NoError(t, err)
	require.Equal(t, pending.UserID, userID)
	require.Equal(t, pending.ProblemID, problemID)
	solved, err := db.HasUserSolvedProblem(userID, problemID)
	require.NoError(t, err)
	require.False(t, solved)
	_, err = db.conn.Exec("DROP TRIGGER fail_approved_ac")
	require.NoError(t, err)
	consumed, err = db.AcceptPendingConfirmation(pending.MessageID)
	require.NoError(t, err)
	require.Equal(t, pending, consumed)
}

func TestConcurrentACProposalDecisionsHaveOneWinner(t *testing.T) {
	t.Parallel()
	db, pending := proposalFixture(t)
	type outcome struct {
		approved bool
		pending  PendingConfirmation
		err      error
	}
	outcomes := make(chan outcome, 2)
	start := make(chan struct{})
	for _, approve := range []bool{true, false} {
		go func() {
			<-start
			var consumed PendingConfirmation
			var err error
			if approve {
				consumed, err = db.AcceptPendingConfirmation(pending.MessageID)
			} else {
				consumed, err = db.RemovePendingConfirmation(pending.MessageID)
			}
			outcomes <- outcome{approve, consumed, err}
		}()
	}
	close(start)
	winners := 0
	approved := false
	for range 2 {
		result := <-outcomes
		if result.err == nil {
			winners++
			approved = result.approved
			require.Equal(t, pending, result.pending)
		} else {
			require.True(t, dataerr.IsNotFound(result.err), "%v", result.err)
		}
	}
	require.Equal(t, 1, winners)
	solved, err := db.HasUserSolvedProblem(pending.UserID, pending.ProblemID)
	require.NoError(t, err)
	require.Equal(t, approved, solved)
}
