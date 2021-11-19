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

func (c *AdminCommands) announceInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "adminCommands.announceInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling admin interaction", "command", "announce")

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

	var eventName, phrase string
	var announceChannel snowflake.Snowflake
	for i := range opts {
		if opts[i].Name == "event_name" {
			eventName = opts[i].ValueString
			continue
		}

		if opts[i].Name == "announce_message" {
			phrase = opts[i].ValueString
			continue
		}

		if opts[i].Name == "announce_channel" {
			announceChannel = opts[i].ValueChannel
		}
	}

	r2, err := c.announce(ctx, ix.GuildID(), gsettings, eventName, phrase)
	if err != nil {
		return r, nil, err
	}
	r2.SetColor(okColor)

	if announceChannel != 0 {
		r2.ToChannel = announceChannel
	}

	r.Description = "Announcing to the designated channel."
	r.SetColor(okColor)

	level.Info(logger).Message("trial announced", "trial_name", eventName, "announce_channel", r2.ToChannel.ToString(), "announce_to", r2.To)

	return r, []cmdhandler.Response{r2}, nil
}

func (c *AdminCommands) announceHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "adminCommands.announceHandler", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling adminCommand", "command", "announce", "args", msg.Contents())

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

	trialName := msg.Contents()[0]
	phrase := strings.Join(msg.Contents()[1:], " ")

	r2, err := c.announce(ctx, msg.GuildID(), gsettings, trialName, phrase)
	if err != nil {
		return r, err
	}
	r2.SetColor(okColor)

	level.Info(logger).Message("trial announced", "trial_name", trialName, "announce_channel", r2.ToChannel.ToString(), "announce_to", r2.To)

	return r2, nil
}

func (c *AdminCommands) announce(ctx context.Context, gid snowflake.Snowflake, gsettings storage.GuildSettings, eventName, phrase string) (*cmdhandler.EmbedResponse, error) {
	ctx, span := c.deps.Census().StartSpan(ctx, "adminCommands.announce", "guild_id", gid.ToString())
	defer span.End()

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

	var signupCid snowflake.Snowflake
	var announceCid snowflake.Snowflake

	if scID, ok := sessionGuild.ChannelWithName(trial.GetSignupChannel(ctx)); ok {
		signupCid = scID
	}

	if acID, ok := sessionGuild.ChannelWithName(trial.GetAnnounceChannel(ctx)); ok {
		announceCid = acID
	}

	roles := trial.GetRoleCounts(ctx)
	signups := trial.GetSignups(ctx)

	roleStrs := make([]string, 0, len(roles))
	emojis := make([]string, 0, len(roles))
	for _, rc := range roles {
		suNames, ofNames := getTrialRoleSignups(ctx, signups, rc)

		filledStr := fmt.Sprintf("(%d signed up", len(suNames))
		if len(ofNames) != 0 {
			filledStr += fmt.Sprintf("; %d overflow)", len(ofNames))
		} else {
			filledStr += ")"
		}

		emoji := rc.GetEmoji(ctx)
		roleStrs = append(roleStrs, fmt.Sprintf("%s %s: %d %s", emoji, rc.GetRole(ctx), rc.GetCount(ctx), filledStr))

		if emoji != "" {
			emojis = append(emojis, emoji)
		}
	}

	var toStr string
	tAnnTo := trial.GetAnnounceTo(ctx)
	switch {
	case tAnnTo != "":
		toStr = tAnnTo
	case gsettings.AnnounceTo != "":
		toStr = gsettings.AnnounceTo
	default:
		toStr = "@everyone"
	}

	desc := trial.GetDescription(ctx)
	if t := trial.GetTime(ctx); t != "" {
		desc = fmt.Sprintf("When: %s\n\n%s", t, desc)
	}

	r2 := &cmdhandler.EmbedResponse{
		To:          fmt.Sprintf("%s %s", toStr, phrase),
		ToChannel:   announceCid,
		Title:       fmt.Sprintf("Signups are open for %s", trial.GetName(ctx)),
		Description: desc,
		Fields: []cmdhandler.EmbedField{
			{
				Name: "Roles Requested",
				Val:  fmt.Sprintf("```\n%s\n```\n", strings.Join(roleStrs, "\n")),
			},
		},
		FooterText: fmt.Sprintf("event:%s", trial.GetName(ctx)),
	}

	if !trial.HideReactionsAnnounce(ctx) {
		r2.Reactions = emojis
	}

	if signupCid != 0 {
		r2.Fields = append(r2.Fields, cmdhandler.EmbedField{
			Name: "Signup Channel",
			Val:  cmdhandler.ChannelMentionString(signupCid),
		})
	}

	return r2, nil
}
