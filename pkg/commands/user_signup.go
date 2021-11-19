package commands

import (
	"context"
	"fmt"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	log "github.com/gsmcwhirter/go-util/v8/logging"
	"github.com/gsmcwhirter/go-util/v8/logging/level"
	"github.com/hashicorp/go-multierror"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
	"github.com/gsmcwhirter/discord-bot-lib/v23/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
)

func (c *UserCommands) signupInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "userCommands.signupInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling root interaction", "command", "signup")

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

	var eventName, role string
	for i := range opts {
		if opts[i].Name == "event_name" {
			eventName = opts[i].ValueString
			continue
		}

		if opts[i].Name == "role" {
			role = opts[i].ValueString
			continue
		}
	}

	r2, overflow, err := c.signup(ctx, logger, ix, gsettings, false, ix.GuildID(), ix.UserID(), eventName, role)
	if err != nil {
		return r, nil, errors.Wrap(err, "could not sign up for event")
	}

	if overflow {
		r.Description = fmt.Sprintf("Signed up as OVERFLOW for %s in %s\n", role, eventName)
	} else {
		r.Description = fmt.Sprintf("Signed up for %s in %s\n", role, eventName)
	}

	r.SetColor(okColor)
	r.SetEphemeral(true)

	if r2 != nil {
		r2.SetColor(okColor)
		return r, []cmdhandler.Response{r2}, nil
	}

	return r, nil, nil
}

func (c *UserCommands) signupHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "userCommands.signupHandler", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling rootCommand", "command", "signup", "trial_and_role", msg.Contents())

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
		return r, errors.New("missing role")
	}

	if len(msg.Contents()) > 2 && len(msg.Contents())%2 != 0 {
		return r, errors.New("incorrect number of arguments")
	}

	var descStr string
	var lastResp *cmdhandler.EmbedResponse

	for i := 0; i < len(msg.Contents()); i += 2 {
		trialName, role := msg.Contents()[i], msg.Contents()[i+1]

		r2, overflow, err2 := c.signup(ctx, logger, msg, gsettings, true, msg.GuildID(), msg.UserID(), trialName, role)
		err = multierror.Append(err, err2)
		if err2 == msghandler.ErrNoResponse { // bad channel, for instance
			return r, err2
		}

		if err2 != nil {
			continue
		}

		if overflow {
			descStr += fmt.Sprintf("Signed up as OVERFLOW for %s in %s\n", role, trialName)
		} else {
			descStr += fmt.Sprintf("Signed up for %s in %s\n", role, trialName)
		}

		if r2 != nil {
			lastResp = r2
		}
	}

	err = errors.Wrap(err, "could not sign up for event(s)")

	if lastResp != nil {
		lastResp.Description = fmt.Sprintf("%s\n\n%s", descStr, lastResp.Description)
		if err == nil {
			lastResp.SetColor(okColor)
		}

		lastResp.ToChannel = 0
		lastResp.SetReplyTo(msg)

		return lastResp, err
	}

	r.Description = descStr
	if err == nil {
		r.SetColor(okColor)
	}

	return r, err
}

func (c *UserCommands) signup(ctx context.Context, logger log.Logger, msg msghandler.MessageLike, gsettings storage.GuildSettings, checkChannel bool, gid, uid snowflake.Snowflake, eventName, role string) (r2 *cmdhandler.EmbedResponse, overflow bool, err error) {
	ctx, span := c.deps.Census().StartSpan(ctx, "userCommands.signup", "guild_id", gid.ToString())
	defer span.End()

	t, err := c.deps.TrialAPI().NewTransaction(ctx, gid.ToString(), true)
	if err != nil {
		return nil, false, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	var trial storage.Trial

	trial, err = t.GetTrial(ctx, eventName)
	if err != nil {
		return nil, false, err
	}

	signupCidStr := trial.GetSignupChannel(ctx)

	if checkChannel {
		if !isSignupChannel(ctx, logger, msg, signupCidStr, gsettings.AdminChannel, gsettings.AdminRoles, c.deps.BotSession(), c.deps.Bot()) {
			level.Info(logger).Message("command not in signup channel", "signup_channel", trial.GetSignupChannel(ctx))
			return nil, false, msghandler.ErrNoResponse
		}
	}

	if trial.GetState(ctx) != storage.TrialStateOpen {
		return nil, false, errors.New("cannot sign up for a closed trial")
	}

	overflow, err = signupUser(ctx, trial, cmdhandler.UserMentionString(uid), role)
	if err != nil {
		return nil, false, err
	}

	if err = t.SaveTrial(ctx, trial); err != nil {
		return nil, overflow, errors.Wrap(err, "could not save trial signup")
	}

	if overflow {
		level.Info(logger).Message("signed up", "overflow", true, "role", role, "trial_name", eventName)
	} else {
		level.Info(logger).Message("signed up", "overflow", false, "role", role, "trial_name", eventName)
	}

	if err = t.Commit(ctx); err != nil {
		return nil, overflow, errors.Wrap(err, "could not save trial signup")
	}

	if gsettings.ShowAfterSignup == "true" {
		level.Debug(logger).Message("auto-show after signup", "trial_name", eventName)

		var signupCid snowflake.Snowflake

		sessionGuild, ok := c.deps.BotSession().Guild(gid)
		if ok {
			if scID, ok := sessionGuild.ChannelWithName(signupCidStr); ok {
				signupCid = scID
			}

			r2 = formatTrialDisplay(ctx, trial, true)
			r2.ToChannel = signupCid
		}
	}

	return r2, overflow, nil
}
