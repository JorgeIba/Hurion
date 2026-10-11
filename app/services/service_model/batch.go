package service_model

const MaxBatchOperations = 100

// BatchEntryError uses one-based entry numbers; zero identifies a batch-wide error.
type BatchEntryError struct {
	Entry int
	Err   error
}

// Prepared values are read-only snapshots for preview. Targets, permissions and
// conflicts must be checked again before executing the batch.
type PreparedProblem struct {
	Entry  int
	Params CreateProblemParams
	Class  Class
}

type PreparedAC struct {
	Entry   int
	User    User
	Problem Problem
	Class   Class
	// AlreadyRecorded marks an entry omitted from writes and reported in the preview.
	AlreadyRecorded bool
}

// Execution results contain only acknowledged writes. When execution returns
// an error, the failed write's outcome may be uncertain; these are not rollback results.
type ProblemCreateBatchResult struct {
	CreatedProblemIDs []int64
}

type ACBatchResult struct {
	Recorded        int
	AlreadyRecorded int
}

// PreparedProblemReplacement contains normalized replacement values and destination identity.
type PreparedProblemReplacement struct {
	Entry  int
	Params ReplaceProblemParams
	Class  Class
}

// ProblemReplacementBatchResult reports acknowledged replacements in input order.
type ProblemReplacementBatchResult struct {
	ReplacedProblemIDs []int64
}

// PreparedProblemPatch contains a normalized patch and the destination class,
// when the patch explicitly changes class membership.
type PreparedProblemPatch struct {
	Entry  int
	Params PatchProblemParams
	Class  Class
}

// ProblemPatchBatchResult reports acknowledged patches in input order.
type ProblemPatchBatchResult struct {
	PatchedProblemIDs []int64
}
