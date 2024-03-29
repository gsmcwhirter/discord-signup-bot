package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
	"github.com/gsmcwhirter/discord-bot-lib/v23/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
)

func (c *AdminCommands) debugInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "adminCommands.debugInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling admin interaction", "command", "debug")

	gsettings, err := storage.GetSettings(ctx, c.deps.GuildAPI(), ix.GuildID())
	if err != nil {
		return r, nil, err
	}

	okColor, err := colorToInt(gsettings.MessageColor)
	if err != nil {
		return r, nil, err
	}

	errColor, err := colorToInt(gsettings.ErrorColor)
	if err != nil {
		return r, nil, err
	}

	r.SetColor(errColor)

	if !isAdminChannel(logger, ix, gsettings.AdminChannel, c.deps.BotSession()) {
		level.Info(logger).Message("command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, nil, msghandler.ErrUnauthorized
	}

	var eventName string
	for i := range opts {
		if opts[i].Name == "event_name" {
			eventName = opts[i].ValueString
			continue
		}
	}

	r2, err := c.debug(ctx, ix.GuildID(), eventName)
	if err != nil {
		return r, nil, errors.Wrap(err, "could not debug event")
	}
	r2.SetColor(okColor)

	level.Info(logger).Message("trial debugged", "trial_name", eventName)

	return r2, nil, nil
}

func (c *AdminCommands) debugHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "adminCommands.debugHandler", "guild_id", msg.GuildID().ToString())
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
		return r, msghandler.ErrUnauthorized
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

	r2, err := c.debug(ctx, msg.GuildID(), trialName)
	if err != nil {
		return r, errors.Wrap(err, "could not debug event")
	}
	r2.SetColor(okColor)
	r2.SetReplyTo(msg)

	level.Info(logger).Message("trial debug shown", "trial_name", trialName)

	return r2, nil
}

func (c *AdminCommands) debug(ctx context.Context, gid snowflake.Snowflake, eventName string) (*cmdhandler.SimpleEmbedResponse, error) {
	ctx, span := c.deps.Census().StartSpan(ctx, "adminCommands.debug", "guild_id", gid.ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	t, err := c.deps.TrialAPI().NewTransaction(ctx, gid.ToString(), false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	trial, err := t.GetTrial(ctx, eventName)
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

	g, ok := c.deps.BotSession().Guild(gid)
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
	- Time: '%[11]s',
	- AnnounceChannel: '#%[2]s',
	- AnnounceChannelID: %[9]s,
	- SignupChannel: '#%[3]s',
	- SignupChannelID: %[10]s,
	- AnnounceTo: '%[4]s', 
	- HideReactionsAnnounce: '%[12]v',
	- HideReactionsShow: '%[13]v',
	- RoleOrder: '%[8]s',
	- Roles:
		%[6]s
%[1]s

Description:
%[1]s
%[7]s

%[1]s`, "```", announceChannel, signupChannel, trial.GetAnnounceTo(ctx), trial.GetState(ctx), roleStr, trial.GetDescription(ctx), roleOrderStr, announceChannelID.ToString(), signupChannelID.ToString(), trial.GetTime(ctx), trial.HideReactionsAnnounce(ctx), trial.HideReactionsShow(ctx))

	return r, nil
}
