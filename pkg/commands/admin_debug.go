package commands

import (
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v20/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v20/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v20/snowflake"
)

func (c *adminCommands) debug(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "adminCommands.debug", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling adminCommand", "command", "debug", "args", msg.Contents())

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

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		level.Info(logger).Message("command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return nil, msghandler.ErrUnauthorized
	}

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 1 {
		return r, errors.New("need event name")
	}

	if len(msg.Contents()) > 1 {
		return r, errors.New("too many arguments")
	}

	trialName := msg.Contents()[0]

	t, err := c.deps.TrialAPI().NewTransaction(ctx, msg.GuildID().ToString(), false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	trial, err := t.GetTrial(ctx, trialName)
	if err != nil {
		return r, err
	}

	rcs := trial.GetRoleCounts(ctx)
	rsParts := make([]string, 0, len(rcs))
	for _, rc := range rcs {
		rsParts = append(rsParts, fmt.Sprintf("'%s' %d '%s'", rc.GetRole(ctx), rc.GetCount(ctx), rc.GetEmoji(ctx)))
	}
	roleStr := strings.Join(rsParts, "\n		")

	ro := trial.GetRoleOrder(ctx)
	roleOrderStr := strings.Join(ro, ", ")

	announceChannel := trial.GetAnnounceChannel(ctx)
	signupChannel := trial.GetSignupChannel(ctx)

	g, ok := c.deps.BotSession().Guild(msg.GuildID())
	if !ok {
		return r, errors.New("could not find guild in session")
	}

	var announceChannelID, signupChannelID snowflake.Snowflake
	if announceChannel != "" {
		if cid, ok := g.ChannelWithName(announceChannel); ok {
			announceChannelID = cid
		}
	}

	if signupChannel != "" {
		if cid, ok := g.ChannelWithName(signupChannel); ok {
			signupChannelID = cid
		}
	}

	r.Description = fmt.Sprintf(`
Event settings:
%[1]s
	- State: '%[5]s',
	- AnnounceChannel: '#%[2]s',
	- AnnounceChannelID: %[9]s,
	- SignupChannel: '#%[3]s',
	- SignupChannelID: %[10]s,
	- AnnounceTo: '%[4]s', 
	- RoleOrder: '%[8]s',
	- Roles:
		%[6]s
%[1]s

Description:
%[1]s
%[7]s

%[1]s`, "```", announceChannel, signupChannel, trial.GetAnnounceTo(ctx), trial.GetState(ctx), roleStr, trial.GetDescription(ctx), roleOrderStr, announceChannelID.ToString(), signupChannelID.ToString())
	r.SetColor(okColor)

	level.Info(logger).Message("trial debug shown", "trial_name", trialName)

	return r, nil
}
