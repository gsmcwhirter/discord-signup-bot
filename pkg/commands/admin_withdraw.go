package commands

import (
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v3/deferutil"
	"github.com/gsmcwhirter/go-util/v3/errors"
	"github.com/gsmcwhirter/go-util/v3/logging/level"
	multierror "github.com/hashicorp/go-multierror"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v8/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v8/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v8/snowflake"
)

func (c *adminCommands) withdraw(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling adminCommand", "command", "withdraw", "args", msg.Contents())

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 2 {
		return r, errors.New("not enough arguments (need `trial-name user-mention(s)`")
	}

	trialName := msg.Contents()[0]

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	if !isSignupChannel(logger, msg, trial.GetSignupChannel(), gsettings.AdminChannel, gsettings.AdminRole, c.deps.BotSession()) {
		level.Info(logger).Message("command not in admin or signup channel", "signup_channel", trial.GetSignupChannel())
		return nil, msghandler.ErrUnauthorized
	}

	userMentions := make([]string, 0, len(msg.Contents())-2)

	for _, m := range msg.Contents()[1:] {
		if !cmdhandler.IsUserMention(m) {
			level.Info(logger).Message("skipping withdraw user", "reason", "not user mention")
			continue
		}

		m, err = cmdhandler.ForceUserNicknameMention(m)
		if err != nil {
			level.Info(logger).Message("skipping withdraw user", "reason", err)
			continue
		}

		userMentions = append(userMentions, m)
	}

	if len(userMentions) == 0 {
		return r, errors.New("you must mention one or more users that you are trying to withdraw (@...)")
	}

	if trial.GetState() != storage.TrialStateOpen {
		return r, errors.New("cannot withdraw from a closed event")
	}

	sessionGuild, ok := c.deps.BotSession().Guild(msg.GuildID())
	if !ok {
		return r, ErrGuildNotFound
	}

	var signupCid snowflake.Snowflake
	if scID, ok := sessionGuild.ChannelWithName(trial.GetSignupChannel()); ok {
		signupCid = scID
	}

	for _, m := range userMentions {
		userAcctMention, werr := cmdhandler.ForceUserAccountMention(m)
		if err != nil {
			err = multierror.Append(err, werr)
			continue
		}

		trial.RemoveSignup(userAcctMention)
		trial.RemoveSignup(m)
	}

	if err != nil {
		return r, err
	}

	if err = t.SaveTrial(trial); err != nil {
		return r, errors.Wrap(err, "could not save event withdraw")
	}

	if err = t.Commit(); err != nil {
		return r, errors.Wrap(err, "could not save event withdraw")
	}

	descStr := fmt.Sprintf("Withdrawn from %s by %s", trialName, cmdhandler.UserMentionString(msg.UserID()))

	if gsettings.ShowAfterWithdraw == "true" {
		level.Debug(logger).Message("auto-show after withdraw", "trial_name", trialName)

		r2 := formatTrialDisplay(trial, true)
		r2.To = strings.Join(userMentions, ", ")
		r2.ToChannel = signupCid
		r2.Description = fmt.Sprintf("%s\n\n%s", descStr, r2.Description)
		return r2, nil
	}

	r.To = strings.Join(userMentions, ", ")
	r.ToChannel = signupCid
	r.Description = descStr

	level.Info(logger).Message("admin withdraw complete", "trial_name", trialName, "withdraw_users", userMentions, "signup_channel", r.ToChannel.ToString())

	return r, nil
}
