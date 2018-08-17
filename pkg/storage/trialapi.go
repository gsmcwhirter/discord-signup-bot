package storage

//go:generate protoc --go_out=. --proto_path=. ./trialapi.proto

// TrialState TODOC
type TrialState string

// State Constants
const (
	TrialStateOpen   = "open"
	TrialStateClosed = "closed"
)

// TrialAPI TODOC
type TrialAPI interface {
	NewTransaction(guild string, writable bool) (TrialAPITx, error)
}

// TrialAPITx TODOC
type TrialAPITx interface {
	Commit() error
	Rollback() error

	GetTrial(name string) (Trial, error)
	AddTrial(name string) (Trial, error)
	SaveTrial(trial Trial) error
	DeleteTrial(name string) error

	GetTrials() []Trial
}

// Trial TODOC
type Trial interface {
	GetName() string
	GetDescription() string
	GetAnnounceChannel() string
	GetSignupChannel() string
	GetState() TrialState
	GetSignups() []TrialSignup
	GetRoleCounts() []RoleCount

	SetName(name string)
	SetDescription(d string)
	SetAnnounceChannel(val string)
	SetSignupChannel(val string)
	SetState(state TrialState)
	AddSignup(name, role string)
	RemoveSignup(name string)
	SetRoleCount(name, emoji string, ct uint64)
	RemoveRole(name string)

	Serialize() ([]byte, error)
}

// TrialSignup TODOC
type TrialSignup interface {
	GetName() string
	GetRole() string
}

// RoleCount TODOC
type RoleCount interface {
	GetRole() string
	GetCount() uint64
	GetEmoji() string
}
