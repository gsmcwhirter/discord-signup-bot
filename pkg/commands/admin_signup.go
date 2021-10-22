package commands

import (
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/logging/level"
	multierror "github.com/hashicorp/go-multierror"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v20/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v20/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v20/snowflake"
)

func (c *adminCommands) signup(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "adminCommands.signup", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling adminCommand", "command", "signup", "args", msg.Contents())

	gsettings, err := storage.GetSettings(ctx, c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	okColor, err := colorToInt(gsettings.MessageColor)
	if err != nil {
		return r, err
	}

	errColor, err := colorToInt(gsettings.ErrorColor)
	if err != nil {
		return r, err
	}

	r.SetColor(errColor)

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 3 {
		return r, errors.New("not enough arguments (need `event-name role user-mention(s)`")
	}

	trialName := msg.Contents()[0]

	t, err := c.deps.TrialAPI().NewTransaction(ctx, msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	trial, err := t.GetTrial(ctx, trialName)
	if err != nil {
		return r, err
	}

	if !isSignupChannel(ctx, logger, msg, trial.GetSignupChannel(ctx), gsettings.AdminChannel, gsettings.AdminRoles, c.deps.BotSession(), c.deps.Bot()) {
		level.Info(logger).Message("command not in admin or signup channel", "signup_channel", trial.GetSignupChannel(ctx))
		return nil, msghandler.ErrUnauthorized
	}

	role := msg.Contents()[1]
	userMentions := make([]string, 0, len(msg.Contents())-2)

	for _, m := range msg.Contents()[2:] {
		if !cmdhandler.IsUserMention(m) {
			level.Info(logger).Message("skipping signup user", "reason", "not user mention")
			continue
		}

		m, err = cmdhandler.ForceUserNicknameMention(m)
		if err != nil {
			level.Info(logger).Message("skipping signup user", "reason", err)
			continue
		}

		userMentions = append(userMentions, m)
	}

	if len(userMentions) == 0 {
		return r, errors.New("you must mention one or more users that you are trying to sign up (@...)")
	}

	if trial.GetState(ctx) != storage.TrialStateOpen {
		return r, errors.New("cannot sign up for a closed event")
	}

	sessionGuild, ok := c.deps.BotSession().Guild(msg.GuildID())
	if !ok {
		return r, ErrGuildNotFound
	}

	var signupCid snowflake.Snowflake
	if scID, ok := sessionGuild.ChannelWithName(trial.GetSignupChannel(ctx)); ok {
		signupCid = scID
	}

	overflows := make([]bool, len(userMentions))
	regularUsers := make([]string, 0, len(userMentions))
	overflowUsers := make([]string, 0, len(userMentions))

	for i, userMention := range userMentions {
		var serr error
		overflows[i], serr = signupUser(ctx, trial, userMention, role)
		if serr != nil {
			err = multierror.Append(err, serr)
			continue
		}

		if overflows[i] {
			overflowUsers = append(overflowUsers, userMention)
		} else {
			regularUsers = append(regularUsers, userMention)
		}
	}

	if err != nil {
		return r, err
	}

	if err = t.SaveTrial(ctx, trial); err != nil {
		return r, errors.Wrap(err, "could not save event signup")
	}

	if err = t.Commit(ctx); err != nil {
		return r, errors.Wrap(err, "could not save event signup")
	}

	descStr := fmt.Sprintf("Signed up for %s in %s by %s\n\n", role, trialName, cmdhandler.UserMentionString(msg.UserID()))
	if len(regularUsers) > 0 {
		descStr += fmt.Sprintf("**Main Group:** %s\n", strings.Join(regularUsers, ", "))
	}
	if len(overflowUsers) > 0 {
		descStr += fmt.Sprintf("**Overflow:** %s\n", strings.Join(overflowUsers, ", "))
	}

	if gsettings.ShowAfterSignup == "true" {
		level.Debug(logger).Message("auto-show after signup", "trial_name", trialName)

		r2 := formatTrialDisplay(ctx, trial, true)
		r2.To = strings.Join(userMentions, ", ")
		r2.ToChannel = signupCid
		r2.Description = fmt.Sprintf("%s\n\n%s", descStr, r2.Description)
		r2.Color = okColor
		return r2, nil
	}

	r.To = strings.Join(userMentions, ", ")
	r.ToChannel = signupCid
	r.Description = descStr
	r.SetColor(okColor)

	level.Info(logger).Message("admin signup complete", "trial_name", trialName, "signup_users", userMentions, "overflows", overflows, "role", role, "signup_channel", r.ToChannel.ToString())

	return r, nil
}
