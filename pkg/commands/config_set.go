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

type argPair struct {
	key, val string
}

func (c *ConfigCommands) setInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "configCommands.setInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling config interaction", "command", "set")

	gsettings, err := storage.GetSettings(ctx, c.deps.GuildAPI(), ix.GuildID())
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

	argPairs := make([]argPair, 0, len(opts))

	for i := range opts {
		name := strings.ToLower(opts[i].Name)
		ap := argPair{
			key: name,
		}

		switch name {
		case "controlsequence":
			ap.val = opts[i].ValueString
		case "announcechannel":
			c, ok := ix.Data.Resolved.Channels[opts[i].ValueChannel]
			if !ok {
				return r, nil, errors.Wrap(ErrMissingData, "could not find announce channel in resolved data to get name", "cid", opts[i].ValueChannel)
			}
			ap.val = c.Name
		case "adminchannel":
			c, ok := ix.Data.Resolved.Channels[opts[i].ValueChannel]
			if !ok {
				return r, nil, errors.Wrap(ErrMissingData, "could not find admin channel in resolved data to get name", "cid", opts[i].ValueChannel)
			}
			ap.val = c.Name
		case "signupchannel":
			c, ok := ix.Data.Resolved.Channels[opts[i].ValueChannel]
			if !ok {
				return r, nil, errors.Wrap(ErrMissingData, "could not find signup channel in resolved data to get name", "cid", opts[i].ValueChannel)
			}
			ap.val = c.Name
		case "announceto":
			ap.val = opts[i].ValueString
		case "showaftersignup":
			if opts[i].ValueBool {
				ap.val = "true"
			} else {
				ap.val = "false"
			}
		case "showafterwithdraw":
			if opts[i].ValueBool {
				ap.val = "true"
			} else {
				ap.val = "false"
			}
		case "hidereactionsannounce":
			if opts[i].ValueBool {
				ap.val = "true"
			} else {
				ap.val = "false"
			}
		case "hidereactionsshow":
			if opts[i].ValueBool {
				ap.val = "true"
			} else {
				ap.val = "false"
			}
		case "messagecolor":
			ap.val = opts[i].ValueString
		case "errorcolor":
			ap.val = opts[i].ValueString
		default:
			return r, nil, errors.WithDetails(errors.New("unknown setting"), "setting_name", name)
		}

		argPairs = append(argPairs, ap)
	}

	if err := c.setSettings(ctx, ix.GuildID(), argPairs); err != nil {
		return r, nil, errors.Wrap(err, "could not set settings")
	}

	return c.listInteraction(ix, opts)
}

func (c *ConfigCommands) setHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "configCommands.setHandler", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling configCommand", "command", "set", "set_args", msg.Contents())

	gsettings, err := storage.GetSettings(ctx, c.deps.GuildAPI(), msg.GuildID())
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

	argPairs := make([]argPair, 0, len(msg.Contents()))

	for _, arg := range msg.Contents() {
		if arg == "" {
			continue
		}

		argPairList := strings.SplitN(arg, "=", 2)
		if len(argPairList) != 2 {
			return r, fmt.Errorf("could not parse setting '%s'", arg)
		}

		ap := argPair{
			key: argPairList[0],
		}

		switch strings.ToLower(argPairList[0]) {
		case "adminrole":
			g, ok := c.deps.BotSession().Guild(msg.GuildID())
			if !ok {
				return r, errors.New("could not find guild to look up role")
			}

			parts := strings.Split(argPairList[1], ",")
			rids := make([]string, 0, len(parts))

			for _, rn := range parts {
				rn = strings.TrimSpace(rn)
				if rn == "" {
					continue
				}

				rid, ok := g.RoleWithName(rn)
				if !ok {
					return r, fmt.Errorf("could not find role with name '%s'", rn)
				}

				rids = append(rids, rid.ToString())
			}

			ap.val = strings.Join(rids, ",")

		default:
			ap.val = argPairList[1]
		}

		argPairs = append(argPairs, ap)
	}

	if err := c.setSettings(ctx, msg.GuildID(), argPairs); err != nil {
		return r, errors.Wrap(err, "could not set settings")
	}

	return c.listHandler(cmdhandler.NewWithContents(msg, ""))
}

func (c *ConfigCommands) setSettings(ctx context.Context, gid snowflake.Snowflake, aps []argPair) error {
	ctx, span := c.deps.Census().StartSpan(ctx, "configCommands.setHandler", "guild_id", gid.ToString())
	defer span.End()

	if len(aps) == 0 {
		return errors.New("no settings to save")
	}

	t, err := c.deps.GuildAPI().NewTransaction(ctx, true)
	if err != nil {
		return err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	bGuild, err := t.AddGuild(ctx, gid.ToString())
	if err != nil {
		return errors.Wrap(err, "unable to find guild")
	}

	s := bGuild.GetSettings(ctx)
	for _, ap := range aps {
		err = s.SetSettingString(ctx, ap.key, ap.val)
		if err != nil {
			return err
		}
	}
	bGuild.SetSettings(ctx, s)

	err = t.SaveGuild(ctx, bGuild)
	if err != nil {
		return errors.Wrap(err, "could not save guild settings")
	}

	err = t.Commit(ctx)
	return errors.Wrap(err, "could not save guild settings")
}
