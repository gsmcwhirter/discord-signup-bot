package commands

import (
	"fmt"
	"strings"

	"github.com/go-kit/kit/log/level"
	"github.com/gsmcwhirter/discord-bot-lib/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/logging"
	"github.com/gsmcwhirter/discord-bot-lib/snowflake"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
	"github.com/gsmcwhirter/go-util/deferutil"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

func (c *adminCommands) signup(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling adminCommand", "command", "signup", "args", msg.Contents())

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		_ = level.Info(logger).Log("message", "command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, msghandler.ErrNoResponse
	}

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 3 {
		return r, errors.New("not enough arguments (need `event-name role user-mention(s)`")
	}

	trialName := msg.Contents()[0]
	role := msg.Contents()[1]
	userMentions := make([]string, 0, len(msg.Contents())-2)

	for _, m := range msg.Contents()[2:] {
		if !cmdhandler.IsUserMention(m) {
			_ = level.Warn(logger).Log("message", "skipping signup user", "reason", "not user mention")
			continue
		}

		m, err = cmdhandler.ForceUserNicknameMention(m)
		if err != nil {
			_ = level.Warn(logger).Log("message", "skipping signup user", "reason", err)
			continue
		}

		userMentions = append(userMentions, m)
	}

	if len(userMentions) == 0 {
		return r, errors.New("you must mention one or more users that you are trying to sign up (@...)")
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	if trial.GetState() != storage.TrialStateOpen {
		return r, errors.New("cannot sign up for a closed event")
	}

	sessionGuild, ok := c.deps.BotSession().Guild(msg.GuildID())
	if !ok {
		return r, ErrGuildNotFound
	}

	var signupCid snowflake.Snowflake
	if scID, ok := sessionGuild.ChannelWithName(trial.GetSignupChannel()); ok {
		signupCid = scID
	}

	overflows := make([]bool, len(userMentions))
	regularUsers := make([]string, 0, len(userMentions))
	overflowUsers := make([]string, 0, len(userMentions))

	for i, userMention := range userMentions {
		var serr error
		overflows[i], serr = signupUser(trial, userMention, role)
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

	if err = t.SaveTrial(trial); err != nil {
		return r, errors.Wrap(err, "could not save event signup")
	}

	if err = t.Commit(); err != nil {
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
		_ = level.Debug(logger).Log("message", "auto-show after signup", "trial_name", trialName)

		r2 := formatTrialDisplay(trial, true)
		r2.To = strings.Join(userMentions, ", ")
		r2.ToChannel = signupCid
		r2.Description = fmt.Sprintf("%s\n\n%s", descStr, r2.Description)
		return r2, nil
	}

	r.To = strings.Join(userMentions, ", ")
	r.ToChannel = signupCid
	r.Description = descStr

	_ = level.Info(logger).Log("message", "admin signup complete", "trial_name", trialName, "signup_users", userMentions, "overflows", overflows, "role", role, "signup_channel", r.ToChannel.ToString())

	return r, nil
}
