package commands

import (
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v19/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v19/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v19/snowflake"
)

func (c *adminCommands) announce(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "adminCommands.announce", "guild_id", msg.GuildID().ToString())
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
	phrase := strings.Join(msg.Contents()[1:], " ")

	t, err := c.deps.TrialAPI().NewTransaction(ctx, msg.GuildID().ToString(), false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	trial, err := t.GetTrial(ctx, trialName)
	if err != nil {
		return r, err
	}

	sessionGuild, ok := c.deps.BotSession().Guild(msg.GuildID())
	if !ok {
		return r, ErrGuildNotFound
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

	r2 := &cmdhandler.EmbedResponse{
		To:          fmt.Sprintf("%s %s", toStr, phrase),
		ToChannel:   announceCid,
		Title:       fmt.Sprintf("Signups are open for %s", trial.GetName(ctx)),
		Description: trial.GetDescription(ctx),
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

	level.Info(logger).Message("trial announced", "trial_name", trialName, "announce_channel", r2.ToChannel.ToString(), "announce_to", r2.To)

	return r2, nil
}
