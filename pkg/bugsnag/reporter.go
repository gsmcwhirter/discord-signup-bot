package bugsnag

import (
	"context"

	bugsnag "github.com/bugsnag/bugsnag-go"
	bserr "github.com/bugsnag/bugsnag-go/errors"
	log "github.com/gsmcwhirter/go-util/v7/logging"
	"github.com/gsmcwhirter/go-util/v7/logging/level"
)

type Reporter struct {
	logger  log.Logger
	bugsnag *bugsnag.Notifier
}

func NewReporter(l log.Logger, apiKey, buildVersion, releaseStage string) Reporter {
	r := Reporter{
		logger: l,
		bugsnag: bugsnag.New(bugsnag.Configuration{
			APIKey:          apiKey,
			AppVersion:      buildVersion,
			Logger:          l,
			ProjectPackages: []string{"main", "github.com/gsmcwhirter/discord-signup-bot"},
			ReleaseStage:    releaseStage,
		}),
	}

	r.bugsnag.FlushSessionsOnRepanic(true)

	return r
}

func (r Reporter) report(ctx context.Context, rcv interface{}) {
	level.Error(log.WithContext(ctx, r.logger)).Message("reporting panic", "panic", rcv)

	// this is a copy-pasta modification of https://github.com/bugsnag/bugsnag-go/blob/master/notifier.go#L74
	state := bugsnag.HandledState{
		SeverityReason:   bugsnag.SeverityReasonHandledPanic,
		OriginalSeverity: bugsnag.SeverityError,
		Unhandled:        true,
		Framework:        "",
	}
	rawData := append([]interface{}{state}, rcv)
	if err := r.bugsnag.NotifySync(bserr.New(rcv, 2), true, rawData...); err != nil {
		level.Error(log.WithContext(ctx, r.logger)).Err("error reporting panic", err)
	}
}

func (r Reporter) AutoNotify(ctx context.Context) {
	if rcv := recover(); rcv != nil {
		r.report(ctx, rcv)

		panic(rcv)
	}
}

func (r Reporter) Recover(ctx context.Context) {
	if rcv := recover(); rcv != nil {
		r.report(ctx, rcv)
	}
}

func (r Reporter) Notify(ctx context.Context, err error) {
	r.report(ctx, err)
}
