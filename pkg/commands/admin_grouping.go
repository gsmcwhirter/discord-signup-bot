package commands

import (
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v7/deferutil"
	"github.com/gsmcwhirter/go-util/v7/errors"
	"github.com/gsmcwhirter/go-util/v7/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v18/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v18/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v18/snowflake"
)

func (c *adminCommands) grouping(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "adminCommands.grouping", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling adminCommand", "command", "grouping", "args", msg.Contents())

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
	phrase := fmt.Sprintf("Grouping now for %s!", trialName)
	if len(msg.Contents()) > 1 {
		phrase = strings.Join(msg.Contents()[1:], " ")
	}

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

	var announceCid snowflake.Snowflake
	if acID, ok := sessionGuild.ChannelWithName(trial.GetAnnounceChannel(msg.Context())); ok {
		announceCid = acID
	}

	roleCounts := trial.GetRoleCounts(msg.Context()) // already sorted by name
	signups := trial.GetSignups(msg.Context())

	userMentions := make([]string, 0, len(signups))

	for _, rc := range roleCounts {
		suNames, ofNames := getTrialRoleSignups(msg.Context(), signups, rc)

		userMentions = append(userMentions, suNames...)
		userMentions = append(userMentions, ofNames...)
	}

	fmt.Printf("** %v\n", userMentions)

	toStr := strings.Join(userMentions, ", ")

	r = &cmdhandler.SimpleEmbedResponse{
		To:          toStr,
		ToChannel:   announceCid,
		Description: phrase,
	}

	level.Info(logger).Message("trial grouping", "trial_name", trialName, "announce_channel", r.ToChannel.ToString(), "announce_to", r.To)

	return r, nil
}
