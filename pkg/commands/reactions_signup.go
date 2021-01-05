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
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := reactions.LoggerWithReaction(msg, c.deps.Logger())
	level.Info(logger).Message("handling reaction", "command", "signup")

	trialName, err := c.getTrialNameForReaction(ctx, logger, msg)
	if err != nil {
		return r, msghandler.ErrNoResponse
	}
	level.Info(logger).Message("reaction event identified", "trial_name", trialName)

	// proceed with the signup
	gsettings, err := storage.GetSettings(ctx, c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	t, err := c.deps.TrialAPI().NewTransaction(ctx, msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	var descStr string

	trial, err := t.GetTrial(ctx, trialName)
	if err != nil {
		return r, err
	}

	if !isSignupChannel(ctx, logger, msg, trial.GetSignupChannel(ctx), gsettings.AdminChannel, gsettings.AdminRoles, c.deps.BotSession(), c.deps.Bot()) {
		level.Info(logger).Message("command not in signup channel", "signup_channel", trial.GetSignupChannel(ctx))
		return r, msghandler.ErrNoResponse
	}

	// find the role based on the emoji
	role := ""
	roleCounts := trial.GetRoleCounts(ctx)
	for _, rc := range roleCounts {
		rcEmoji := rc.GetEmoji(ctx)
		if rcEmoji == msg.Emoji() || strings.HasPrefix(rcEmoji, fmt.Sprintf("<:%s:", msg.Emoji())) {
			role = rc.GetRole(ctx)
		}
	}
	if role == "" {
		allEmojis := make([]string, 0, len(roleCounts))
		for _, rc := range roleCounts {
			allEmojis = append(allEmojis, rc.GetEmoji(ctx))
		}
		level.Error(logger).Message("could not find role based on emoji", "emoji", msg.Emoji(), "all_emojis", allEmojis)
		return r, msghandler.ErrNoResponse
	}

	if trial.GetState(ctx) != storage.TrialStateOpen {
		return r, errors.New("cannot sign up for a closed trial")
	}

	overflow, err := signupUser(ctx, trial, cmdhandler.UserMentionString(msg.UserID()), role)
	if err != nil {
		return r, err
	}

	if err = t.SaveTrial(ctx, trial); err != nil {
		return r, errors.Wrap(err, "could not save trial signup")
	}

	if overflow {
		level.Info(logger).Message("signed up", "overflow", true, "role", role, "trial_name", trialName)
		descStr += fmt.Sprintf("Signed up as OVERFLOW for %s in %s\n", role, trialName)
	} else {
		level.Info(logger).Message("signed up", "overflow", false, "role", role, "trial_name", trialName)
		descStr += fmt.Sprintf("Signed up for %s in %s\n", role, trialName)
	}

	if err = t.Commit(ctx); err != nil {
		return r, errors.Wrap(err, "could not save trial signup")
	}

	if gsettings.ShowAfterSignup == "true" {
		r2 := formatTrialDisplay(ctx, trial, true)

		r2.Description = fmt.Sprintf("%s\n\n%s", descStr, r2.Description)
		r2.To = cmdhandler.UserMentionString(msg.UserID())

		return r2, nil
	}

	r.Description = descStr

	return r, nil
}
