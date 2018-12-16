package storage

//go:generate protoc --go_out=. --proto_path=. ./trialapi.proto

// TrialState represents the state of a trial
type TrialState string

// State Constants
const (
	TrialStateOpen   = "open"
	TrialStateClosed = "closed"
)

// TrialAPI is the API for managing trials transactions
type TrialAPI interface {
	NewTransaction(guild string, writable bool) (TrialAPITx, error)
}

// TrialAPITx is the api for managing trials within a transaction
type TrialAPITx interface {
	Commit() error
	Rollback() error

	GetTrial(name string) (Trial, error)
	AddTrial(name string) (Trial, error)
	SaveTrial(trial Trial) error
	DeleteTrial(name string) error

	GetTrials() []Trial
}

// Trial is the api for managing a particular trial
type Trial interface {
	GetName() string
	GetDescription() string
	GetAnnounceTo() string
	GetAnnounceChannel() string
	GetSignupChannel() string
	GetState() TrialState
	GetSignups() []TrialSignup
	GetRoleCounts() []RoleCount
	PrettySettings() string

	SetName(name string)
	SetDescription(d string)
	SetAnnounceTo(val string)
	SetAnnounceChannel(val string)
	SetSignupChannel(val string)
	SetState(state TrialState)
	AddSignup(name, role string)
	RemoveSignup(name string)
	SetRoleCount(name, emoji string, ct uint64)
	RemoveRole(name string)

	ClearSignups()

	Serialize() ([]byte, error)
}

// TrialSignup is the api for managing a signup for a trial
type TrialSignup interface {
	GetName() string
	GetRole() string
}

// RoleCount is the api for managing a role in a trial
type RoleCount interface {
	GetRole() string
	GetCount() uint64
	GetEmoji() string
}
