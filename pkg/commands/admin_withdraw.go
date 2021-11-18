package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	log "github.com/gsmcwhirter/go-util/v8/logging"
	"github.com/gsmcwhirter/go-util/v8/logging/level"
	multierror "github.com/hashicorp/go-multierror"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
	"github.com/gsmcwhirter/discord-bot-lib/v23/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
)

func (c *AdminCommands) withdrawInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "adminCommands.withdrawInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling admin interaction", "command", "withdraw")

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

	var eventName string
	var uid snowflake.Snowflake
	for i := range opts {
		if opts[i].Name == "event_name" {
			eventName = opts[i].ValueString
			continue
		}

		if opts[i].Name == "user" {
			uid = opts[i].ValueUser
		}
	}

	userMentions := []string{cmdhandler.UserMentionString(uid)}

	signupCid, r2, err := c.withdraw(ctx, logger, ix, ix.GuildID(), gsettings, eventName, userMentions)
	if err != nil {
		return r, nil, errors.Wrap(err, "could not sign up for the event")
	}

	descStr := fmt.Sprintf("Withdrawn from %s by %s", eventName, cmdhandler.UserMentionString(ix.UserID()))

	level.Info(logger).Message("admin withdraw complete", "trial_name", eventName, "withdraw_users", userMentions, "signup_channel", signupCid.ToString())

	r.Description = "Users withdrawn successfully"

	if r2 != nil {
		r2.Description = fmt.Sprintf("%s\n\n%s", descStr, r2.Description)
		r2.SetColor(okColor)
	} else {
		r2 = &cmdhandler.EmbedResponse{}
		r2.To = strings.Join(userMentions, ", ")
		r2.ToChannel = signupCid
		r2.Description = descStr
		r2.SetColor(okColor)
	}
	return r, []cmdhandler.Response{r2}, nil
}

func (c *AdminCommands) withdrawHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "adminCommands.withdrawHandler", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling adminCommand", "command", "withdraw", "args", msg.Contents())

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

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 2 {
		return r, errors.New("not enough arguments (need `trial-name user-mention(s)`")
	}

	trialName := msg.Contents()[0]
	userMentions := make([]string, 0, len(msg.Contents())-2)

	for _, m := range msg.Contents()[1:] {
		if !cmdhandler.IsUserMention(m) {
			level.Info(logger).Message("skipping withdraw user", "reason", "not user mention")
			continue
		}

		m, err = cmdhandler.ForceUserNicknameMention(m)
		if err != nil {
			level.Info(logger).Message("skipping withdraw user", "reason", err)
			continue
		}

		userMentions = append(userMentions, m)
	}

	if len(userMentions) == 0 {
		return r, errors.New("you must mention one or more users that you are trying to withdraw (@...)")
	}

	signupCid, r2, err := c.withdraw(ctx, logger, msg, msg.GuildID(), gsettings, trialName, userMentions)
	if err != nil {
		return r, errors.Wrap(err, "could not sign up for the event")
	}

	descStr := fmt.Sprintf("Withdrawn from %s by %s", trialName, cmdhandler.UserMentionString(msg.UserID()))

	level.Info(logger).Message("admin withdraw complete", "trial_name", trialName, "withdraw_users", userMentions, "signup_channel", signupCid.ToString())

	if r2 != nil {
		r2.Description = fmt.Sprintf("%s\n\n%s", descStr, r2.Description)
		r2.SetColor(okColor)
		return r2, nil
	}

	r.To = strings.Join(userMentions, ", ")
	r.ToChannel = signupCid
	r.Description = descStr
	r.SetColor(okColor)

	return r, nil
}

func (c *AdminCommands) withdraw(ctx context.Context, logger log.Logger, msg msghandler.MessageLike, gid snowflake.Snowflake, gsettings storage.GuildSettings, eventName string, userMentions []string) (cid snowflake.Snowflake, r2 *cmdhandler.EmbedResponse, err error) {
	ctx, span := c.deps.Census().StartSpan(ctx, "adminCommands.withdraw", "guild_id", gid.ToString())
	defer span.End()

	t, err := c.deps.TrialAPI().NewTransaction(ctx, gid.ToString(), true)
	if err != nil {
		return 0, nil, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	trial, err := t.GetTrial(ctx, eventName)
	if err != nil {
		return 0, nil, err
	}

	// TODO: figure out an alternative here also
	if !isSignupChannel(ctx, logger, msg, trial.GetSignupChannel(ctx), gsettings.AdminChannel, gsettings.AdminRoles, c.deps.BotSession(), c.deps.Bot()) {
		level.Info(logger).Message("command not in admin or signup channel", "signup_channel", trial.GetSignupChannel(ctx))
		return 0, nil, msghandler.ErrUnauthorized
	}

	if trial.GetState(ctx) != storage.TrialStateOpen {
		return 0, nil, errors.New("cannot withdraw from a closed event")
	}

	sessionGuild, ok := c.deps.BotSession().Guild(gid)
	if !ok {
		return 0, nil, ErrGuildNotFound
	}

	var signupCid snowflake.Snowflake
	if scID, ok := sessionGuild.ChannelWithName(trial.GetSignupChannel(ctx)); ok {
		signupCid = scID
	}

	for _, m := range userMentions {
		userAcctMention, werr := cmdhandler.ForceUserAccountMention(m)
		if err != nil {
			err = multierror.Append(err, werr)
			continue
		}

		trial.RemoveSignup(ctx, userAcctMention)
		trial.RemoveSignup(ctx, m)
	}

	if err != nil {
		return 0, nil, err
	}

	if err = t.SaveTrial(ctx, trial); err != nil {
		return 0, nil, errors.Wrap(err, "could not save event withdraw")
	}

	if err = t.Commit(ctx); err != nil {
		return 0, nil, errors.Wrap(err, "could not save event withdraw")
	}

	if gsettings.ShowAfterWithdraw == "true" {
		level.Debug(logger).Message("auto-show after signup", "trial_name", eventName)

		r2 = formatTrialDisplay(ctx, trial, true)
		r2.To = strings.Join(userMentions, ", ")
		r2.ToChannel = signupCid
	}

	return signupCid, r2, nil
}
