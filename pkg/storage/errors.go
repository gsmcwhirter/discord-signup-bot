package storage

import "github.com/gsmcwhirter/go-util/v8/errors"

// ErrGuildNotExist is the error returned if a guild does not exist
var ErrGuildNotExist = errors.New("guild does not exist")

// ErrTrialNotExist is the error returned if a trial does not exist
var ErrTrialNotExist = errors.New("event does not exist")
