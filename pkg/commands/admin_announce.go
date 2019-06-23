package commands

import (
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v5/deferutil"
	"github.com/gsmcwhirter/go-util/v5/errors"
	"github.com/gsmcwhirter/go-util/v5/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v11/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v11/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v11/snowflake"
)

func (c *adminCommands) announce(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "adminCommands.announce", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling adminCommand", "command", "announce", "args", msg.Contents())

	gsettings, err := storage.GetSettings(msg.Context(), c.deps.GuildAPI(), msg.GuildID())
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

	t, err := c.deps.TrialAPI().NewTransaction(msg.Context(), msg.GuildID().ToString(), false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(msg.Context()) })

	trial, err := t.GetTrial(msg.Context(), trialName)
	if err != nil {
		return r, err
	}

	sessionGuild, ok := c.deps.BotSession().Guild(msg.GuildID())
	if !ok {
		return r, ErrGuildNotFound
	}

	var signupCid snowflake.Snowflake
	var announceCid snowflake.Snowflake

	if scID, ok := sessionGuild.ChannelWithName(trial.GetSignupChannel(msg.Context())); ok {
		signupCid = scID
	}

	if acID, ok := sessionGuild.ChannelWithName(trial.GetAnnounceChannel(msg.Context())); ok {
		announceCid = acID
	}

	roles := trial.GetRoleCounts(msg.Context())
	roleStrs := make([]string, 0, len(roles))
	for _, rc := range roles {
		roleStrs = append(roleStrs, fmt.Sprintf("%s: %d", rc.GetRole(msg.Context()), rc.GetCount(msg.Context())))
	}

	var toStr string
	tAnnTo := trial.GetAnnounceTo(msg.Context())
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
		Title:       fmt.Sprintf("Signups are open for %s", trial.GetName(msg.Context())),
		Description: trial.GetDescription(msg.Context()),
		Fields: []cmdhandler.EmbedField{
			{
				Name: "Roles Requested",
				Val:  fmt.Sprintf("```\n%s\n```\n", strings.Join(roleStrs, "\n")),
			},
		},
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
