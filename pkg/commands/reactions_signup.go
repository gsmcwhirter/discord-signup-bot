package commands

import (
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v7/deferutil"
	"github.com/gsmcwhirter/go-util/v7/errors"
	"github.com/gsmcwhirter/go-util/v7/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/reactions"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v18/cmdhandler"
)

func (c *reactionHandler) signup(msg reactions.Reaction) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "reactionHandler.signup", "guild_id", msg.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	if m2, ok := msg.(cmdhandler.Message); ok {
		r.SetReplyTo(m2)
	} else {
		r.To = cmdhandler.UserMentionString(msg.UserID())
	}

	logger := reactions.LoggerWithReaction(msg, c.deps.Logger())
	level.Info(logger).Message("handling reaction", "command", "signup")

	msgInfo, err := c.deps.Bot().GetMessage(ctx, msg.ChannelID(), msg.MessageID())
	if err != nil {
		level.Error(logger).Err("could not get message information", err)
		return r, msghandler.ErrNoResponse
	}

	// was the message reacted to mine?
	if msgInfo.Author.IDSnowflake.ToString() != c.deps.Bot().Config().ClientID {
		level.Debug(logger).Message("reaction was not to our message")
		return r, msghandler.ErrNoResponse
	}

	// pull the trialName out of the message
	if len(msgInfo.Embeds) != 1 {
		level.Info(logger).Message("reaction was not to a message with a single embed", "num_embeds", len(msgInfo.Embeds))
		return r, msghandler.ErrNoResponse
	}

	if !strings.HasPrefix(msgInfo.Embeds[0].Footer.Text, "event:") {
		level.Info(logger).Message("reaction was not to a message with an event: footer", "footer_text", msgInfo.Embeds[0].Footer.Text)
		return r, msghandler.ErrNoResponse
	}

	trialName := strings.TrimSpace(msgInfo.Embeds[0].Footer.Text[6:])
	level.Info(logger).Message("reaction event identified", "trial_name", trialName)

	// proceed with the signup
	gsettings, err := storage.GetSettings(msg.Context(), c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.Context(), msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(msg.Context()) })

	var descStr string

	trial, err := t.GetTrial(msg.Context(), trialName)
	if err != nil {
		return r, err
	}

	if !isSignupChannel(ctx, logger, msg, trial.GetSignupChannel(msg.Context()), gsettings.AdminChannel, gsettings.AdminRole, c.deps.BotSession(), c.deps.Bot()) {
		level.Info(logger).Message("command not in signup channel", "signup_channel", trial.GetSignupChannel(msg.Context()))
		return r, msghandler.ErrNoResponse
	}

	// find the role based on the emoji
	role := ""
	roleCounts := trial.GetRoleCounts(msg.Context())
	for _, rc := range roleCounts {
		if rc.GetEmoji(msg.Context()) == msg.Emoji() {
			role = rc.GetRole(msg.Context())
		}
	}
	if role == "" {
		return r, msghandler.ErrNoResponse
	}

	if trial.GetState(msg.Context()) != storage.TrialStateOpen {
		return r, errors.New("cannot sign up for a closed trial")
	}

	overflow, err := signupUser(msg.Context(), trial, cmdhandler.UserMentionString(msg.UserID()), role)
	if err != nil {
		return r, err
	}

	if err = t.SaveTrial(msg.Context(), trial); err != nil {
		return r, errors.Wrap(err, "could not save trial signup")
	}

	if overflow {
		level.Info(logger).Message("signed up", "overflow", true, "role", role, "trial_name", trialName)
		descStr += fmt.Sprintf("Signed up as OVERFLOW for %s in %s\n", role, trialName)
	} else {
		level.Info(logger).Message("signed up", "overflow", false, "role", role, "trial_name", trialName)
		descStr += fmt.Sprintf("Signed up for %s in %s\n", role, trialName)
	}

	if err = t.Commit(msg.Context()); err != nil {
		return r, errors.Wrap(err, "could not save trial signup")
	}

	if gsettings.ShowAfterSignup == "true" {
		r2 := formatTrialDisplay(msg.Context(), trial, true)

		r2.Description = fmt.Sprintf("%s\n\n%s", descStr, r2.Description)
		if m2, ok := msg.(cmdhandler.Message); ok {
			r2.SetReplyTo(m2)
		} else {
			r2.To = cmdhandler.UserMentionString(msg.UserID())
		}

		return r2, nil
	}

	r.Description = descStr

	return r, nil
}
