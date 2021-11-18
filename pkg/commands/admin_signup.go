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

func (c *AdminCommands) signupInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "adminCommands.signupInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling admin interaction", "command", "signup")

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
	var uid snowflake.Snowflake
	for i := range opts {
		if opts[i].Name == "event_name" {
			eventName = opts[i].ValueString
			continue
		}

		if opts[i].Name == "role" {
			role = opts[i].ValueString
			continue
		}

		if opts[i].Name == "user" {
			uid = opts[i].ValueUser
		}
	}

	userMentions := []string{cmdhandler.UserMentionString(uid)}

	signupCid, r2, accepted, overflows, err := c.signup(ctx, logger, ix, ix.GuildID(), gsettings, eventName, role, userMentions)
	if err != nil {
		return r, nil, errors.Wrap(err, "could not sign up for the event")
	}

	descStr := fmt.Sprintf("Signed up for %s in %s by %s\n\n", role, eventName, cmdhandler.UserMentionString(ix.UserID()))
	if len(accepted) > 0 {
		descStr += fmt.Sprintf("**Main Group:** %s\n", strings.Join(accepted, ", "))
	}
	if len(overflows) > 0 {
		descStr += fmt.Sprintf("**Overflow:** %s\n", strings.Join(overflows, ", "))
	}

	level.Info(logger).Message("admin signup complete", "trial_name", eventName, "signup_users", accepted, "overflows", overflows, "role", role, "signup_channel", signupCid)

	r.Description = "Users signed up successfully"

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

func (c *AdminCommands) signupHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "adminCommands.signupHandler", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling adminCommand", "command", "signup", "args", msg.Contents())

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

	if len(msg.Contents()) < 3 {
		return r, errors.New("not enough arguments (need `event-name role user-mention(s)`")
	}

	trialName := msg.Contents()[0]

	role := msg.Contents()[1]
	userMentions := make([]string, 0, len(msg.Contents())-2)

	for _, m := range msg.Contents()[2:] {
		if !cmdhandler.IsUserMention(m) {
			level.Info(logger).Message("skipping signup user", "reason", "not user mention")
			continue
		}

		m, err = cmdhandler.ForceUserNicknameMention(m)
		if err != nil {
			level.Info(logger).Message("skipping signup user", "reason", err)
			continue
		}

		userMentions = append(userMentions, m)
	}

	if len(userMentions) == 0 {
		return r, errors.New("you must mention one or more users that you are trying to sign up (@...)")
	}

	signupCid, r2, accepted, overflows, err := c.signup(ctx, logger, msg, msg.GuildID(), gsettings, trialName, role, userMentions)
	if err != nil {
		return r, errors.Wrap(err, "could not sign up for the event")
	}

	descStr := fmt.Sprintf("Signed up for %s in %s by %s\n\n", role, trialName, cmdhandler.UserMentionString(msg.UserID()))
	if len(accepted) > 0 {
		descStr += fmt.Sprintf("**Main Group:** %s\n", strings.Join(accepted, ", "))
	}
	if len(overflows) > 0 {
		descStr += fmt.Sprintf("**Overflow:** %s\n", strings.Join(overflows, ", "))
	}

	level.Info(logger).Message("admin signup complete", "trial_name", trialName, "signup_users", accepted, "overflows", overflows, "role", role, "signup_channel", signupCid)

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

func (c *AdminCommands) signup(ctx context.Context, logger log.Logger, msg msghandler.MessageLike, gid snowflake.Snowflake, gsettings storage.GuildSettings, eventName, role string, userMentions []string) (cid snowflake.Snowflake, r2 *cmdhandler.EmbedResponse, accepted, overflows []string, err error) {
	ctx, span := c.deps.Census().StartSpan(ctx, "adminCommands.signup", "guild_id", gid.ToString())
	defer span.End()

	t, err := c.deps.TrialAPI().NewTransaction(ctx, gid.ToString(), true)
	if err != nil {
		return 0, nil, nil, nil, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	trial, err := t.GetTrial(ctx, eventName)
	if err != nil {
		return 0, nil, nil, nil, err
	}

	// TODO: figure out what to do differently here, because having to pass msg through kinda sucks
	if !isSignupChannel(ctx, logger, msg, trial.GetSignupChannel(ctx), gsettings.AdminChannel, gsettings.AdminRoles, c.deps.BotSession(), c.deps.Bot()) {
		level.Info(logger).Message("command not in admin or signup channel", "signup_channel", trial.GetSignupChannel(ctx), "admin_channel", gsettings.AdminChannel)
		return 0, nil, nil, nil, msghandler.ErrUnauthorized
	}

	if trial.GetState(ctx) != storage.TrialStateOpen {
		return 0, nil, nil, nil, errors.New("cannot sign up for a closed event")
	}

	sessionGuild, ok := c.deps.BotSession().Guild(gid)
	if !ok {
		return 0, nil, nil, nil, ErrGuildNotFound
	}

	var signupCid snowflake.Snowflake
	if scID, ok := sessionGuild.ChannelWithName(trial.GetSignupChannel(ctx)); ok {
		signupCid = scID
	}

	ofs := make([]bool, len(userMentions))
	accepted = make([]string, 0, len(userMentions))
	overflows = make([]string, 0, len(userMentions))

	for i, userMention := range userMentions {
		var serr error
		ofs[i], serr = signupUser(ctx, trial, userMention, role)
		if serr != nil {
			err = multierror.Append(err, serr)
			continue
		}

		if ofs[i] {
			overflows = append(overflows, userMention)
		} else {
			accepted = append(accepted, userMention)
		}
	}

	if err != nil {
		return signupCid, nil, nil, nil, err
	}

	if err = t.SaveTrial(ctx, trial); err != nil {
		return signupCid, nil, nil, nil, errors.Wrap(err, "could not save event signup")
	}

	if err = t.Commit(ctx); err != nil {
		return signupCid, nil, nil, nil, errors.Wrap(err, "could not save event signup")
	}

	if gsettings.ShowAfterSignup == "true" {
		level.Debug(logger).Message("auto-show after signup", "trial_name", eventName)

		r2 = formatTrialDisplay(ctx, trial, true)
		r2.To = strings.Join(userMentions, ", ")
		r2.ToChannel = signupCid
	}

	return signupCid, r2, accepted, overflows, nil
}
