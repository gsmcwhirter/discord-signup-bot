package commands

import (
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v3/deferutil"
	"github.com/gsmcwhirter/go-util/v3/errors"
	"github.com/gsmcwhirter/go-util/v3/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v7/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v7/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v7/snowflake"
)

func (c *adminCommands) grouping(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling adminCommand", "command", "grouping", "args", msg.Contents())

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID())
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

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	sessionGuild, ok := c.deps.BotSession().Guild(msg.GuildID())
	if !ok {
		return r, ErrGuildNotFound
	}

	var announceCid snowflake.Snowflake
	if acID, ok := sessionGuild.ChannelWithName(trial.GetAnnounceChannel()); ok {
		announceCid = acID
	}

	roleCounts := trial.GetRoleCounts() // already sorted by name
	signups := trial.GetSignups()

	userMentions := make([]string, 0, len(signups))

	for _, rc := range roleCounts {
		suNames, ofNames := getTrialRoleSignups(signups, rc)

		userMentions = append(userMentions, suNames...)
		userMentions = append(userMentions, ofNames...)
	}

	fmt.Printf("** %v\n", userMentions)

	toStr := strings.Join(userMentions, ", ")

	r.To = fmt.Sprintf("%s\n\n%s", toStr, phrase)
	r.ToChannel = announceCid

	level.Info(logger).Message("trial grouping", "trial_name", trialName, "announce_channel", r.ToChannel.ToString(), "announce_to", r.To)

	return r, nil
}
