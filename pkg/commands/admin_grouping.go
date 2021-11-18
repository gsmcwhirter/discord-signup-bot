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

func (c *AdminCommands) groupingInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "adminCommands.groupingInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling admin interaction", "command", "grouping")

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
		return nil, nil, msghandler.ErrUnauthorized
	}

	var eventName, phrase string
	var announceChannel snowflake.Snowflake
	for i := range opts {
		if opts[i].Name == "event_name" {
			eventName = opts[i].ValueString
			continue
		}

		if opts[i].Name == "grouping_message" {
			phrase = opts[i].ValueString
			continue
		}

		if opts[i].Name == "grouping_channel" {
			announceChannel = opts[i].ValueChannel
		}
	}
	if phrase == "" {
		phrase = fmt.Sprintf("Grouping now for %s!", eventName)
	}

	r2, err := c.grouping(ctx, ix.GuildID(), gsettings, eventName, phrase)
	if err != nil {
		return r, nil, err
	}
	r2.SetColor(okColor)

	if announceChannel != 0 {
		r2.ToChannel = announceChannel
	}

	r.Description = "Announcing to the designated channel."
	r.SetColor(okColor)

	level.Info(logger).Message("trial grouping announced", "trial_name", eventName, "announce_channel", r2.ToChannel.ToString(), "announce_to", r2.To)

	return r, []cmdhandler.Response{r2}, nil
}

func (c *AdminCommands) groupingHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "adminCommands.groupingHandler", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling adminCommand", "command", "grouping", "args", msg.Contents())

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

	trialName := msg.Contents()[0]
	phrase := fmt.Sprintf("Grouping now for %s!", trialName)
	if len(msg.Contents()) > 1 {
		phrase = strings.Join(msg.Contents()[1:], " ")
	}

	r2, err := c.grouping(ctx, msg.GuildID(), gsettings, trialName, phrase)
	if err != nil {
		return r, errors.Wrap(err, "could not make grouping announcement")
	}
	r2.SetColor(okColor)

	level.Info(logger).Message("trial grouping", "trial_name", trialName, "announce_channel", r.ToChannel.ToString(), "announce_to", r.To)

	return r2, nil
}

func (c *AdminCommands) grouping(ctx context.Context, gid snowflake.Snowflake, gsettings storage.GuildSettings, eventName, phrase string) (*cmdhandler.SimpleEmbedResponse, error) {
	t, err := c.deps.TrialAPI().NewTransaction(ctx, gid.ToString(), false)
	if err != nil {
		return nil, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	trial, err := t.GetTrial(ctx, eventName)
	if err != nil {
		return nil, err
	}

	sessionGuild, ok := c.deps.BotSession().Guild(gid)
	if !ok {
		return nil, ErrGuildNotFound
	}

	var announceCid snowflake.Snowflake
	if acID, ok := sessionGuild.ChannelWithName(trial.GetAnnounceChannel(ctx)); ok {
		announceCid = acID
	}

	roleCounts := trial.GetRoleCounts(ctx) // already sorted by name
	signups := trial.GetSignups(ctx)

	userMentions := make([]string, 0, len(signups))

	for _, rc := range roleCounts {
		suNames, ofNames := getTrialRoleSignups(ctx, signups, rc)

		userMentions = append(userMentions, suNames...)
		userMentions = append(userMentions, ofNames...)
	}

	toStr := strings.Join(userMentions, ", ")

	r := &cmdhandler.SimpleEmbedResponse{
		To:          toStr,
		ToChannel:   announceCid,
		Description: phrase,
	}

	return r, nil
}
