package storage

import (
	"context"
)

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
	NewTransaction(ctx context.Context, guild string, writable bool) (TrialAPITx, error)
}

// TrialAPITx is the api for managing trials within a transaction
type TrialAPITx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error

	GetTrial(ctx context.Context, name string) (Trial, error)
	AddTrial(ctx context.Context, name string) (Trial, error)
	SaveTrial(ctx context.Context, trial Trial) error
	DeleteTrial(ctx context.Context, name string) error

	GetTrials(ctx context.Context) []Trial
}

// Trial is the api for managing a particular trial
type Trial interface {
	GetName(ctx context.Context) string
	GetDescription(ctx context.Context) string
	GetAnnounceTo(ctx context.Context) string
	GetAnnounceChannel(ctx context.Context) string
	GetSignupChannel(ctx context.Context) string
	GetState(ctx context.Context) TrialState
	GetSignups(ctx context.Context) []TrialSignup
	GetRoleCounts(ctx context.Context) []RoleCount
	GetRoleOrder(ctx context.Context) []string
	PrettySettings(ctx context.Context) string

	SetName(ctx context.Context, name string)
	SetDescription(ctx context.Context, d string)
	SetAnnounceTo(ctx context.Context, val string)
	SetAnnounceChannel(ctx context.Context, val string)
	SetSignupChannel(ctx context.Context, val string)
	SetState(ctx context.Context, state TrialState)
	AddSignup(ctx context.Context, name, role string)
	RemoveSignup(ctx context.Context, name string)
	SetRoleCount(ctx context.Context, name, emoji string, ct uint64)
	RemoveRole(ctx context.Context, name string)
	SetRoleOrder(ctx context.Context, ord []string)

	ClearSignups(ctx context.Context)

	Serialize(ctx context.Context) ([]byte, error)
}

// TrialSignup is the api for managing a signup for a trial
type TrialSignup interface {
	GetName(ctx context.Context) string
	GetRole(ctx context.Context) string
}

// RoleCount is the api for managing a role in a trial
type RoleCount interface {
	GetRole(ctx context.Context) string
	GetCount(ctx context.Context) uint64
	GetEmoji(ctx context.Context) string
	Index() int
}
