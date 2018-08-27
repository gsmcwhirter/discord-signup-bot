package commands

import (
	"fmt"
	"strings"

	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"

	"github.com/gsmcwhirter/discord-bot-lib/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/logging"
	"github.com/gsmcwhirter/go-util/deferutil"
	"github.com/gsmcwhirter/go-util/parser"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

type configCommands struct {
	preCommand string
	versionStr string
	deps       configDependencies
}

func (c *configCommands) version(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To:          cmdhandler.UserMentionString(msg.UserID()),
		Description: c.versionStr,
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling configCommand", "command", "version")

	return r, nil
}

func (c *configCommands) list(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling configCommand", "command", "list")

	t, err := c.deps.GuildAPI().NewTransaction(false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	bGuild, err := t.AddGuild(msg.GuildID().ToString())
	if err != nil {
		return r, errors.Wrap(err, "unable to find guild")
	}

	s := bGuild.GetSettings()
	r.Description = s.PrettyString()
	return r, nil
}

func (c *configCommands) get(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	settingName := strings.TrimSpace(msg.Contents())

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling adminCommand", "command", "get", "setting_name", settingName)

	t, err := c.deps.GuildAPI().NewTransaction(false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	bGuild, err := t.AddGuild(msg.GuildID().ToString())
	if err != nil {
		return r, errors.Wrap(err, "unable to find guild")
	}

	s := bGuild.GetSettings()
	sVal, err := s.GetSettingString(settingName)
	if err != nil {
		return r, fmt.Errorf("'%s' is not the name of a setting", settingName)
	}

	r.Description = fmt.Sprintf("```\n%s: '%s'\n```", settingName, sVal)
	return r, nil
}

type argPair struct {
	key, val string
}

func (c *configCommands) set(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	args := strings.TrimSpace(msg.Contents())

	argList := strings.Split(args, " ")

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling adminCommand", "command", "set", "set_args", argList)

	argPairs := make([]argPair, 0, len(argList))

	for _, arg := range argList {
		if arg == "" {
			continue
		}

		argPairList := strings.SplitN(arg, "=", 2)
		if len(argPairList) != 2 {
			return r, fmt.Errorf("could not parse setting '%s'", arg)
		}
		argPairs = append(argPairs, argPair{
			key: argPairList[0],
			val: argPairList[1],
		})
	}

	if len(argPairs) == 0 {
		return r, errors.New("no settings to save")
	}

	t, err := c.deps.GuildAPI().NewTransaction(true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	bGuild, err := t.AddGuild(msg.GuildID().ToString())
	if err != nil {
		return r, errors.Wrap(err, "unable to find guild")
	}

	s := bGuild.GetSettings()
	for _, ap := range argPairs {
		err = s.SetSettingString(ap.key, ap.val)
		if err != nil {
			return r, err
		}
	}
	bGuild.SetSettings(s)

	err = t.SaveGuild(bGuild)
	if err != nil {
		return r, errors.Wrap(err, "could not save guild settings")
	}

	err = t.Commit()
	if err != nil {
		return r, errors.Wrap(err, "could not save guild settings")
	}

	return c.list(cmdhandler.NewWithContents(msg, ""))
}

func (c *configCommands) reset(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling adminCommand", "command", "reset")

	t, err := c.deps.GuildAPI().NewTransaction(true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	bGuild, err := t.AddGuild(msg.GuildID().ToString())
	if err != nil {
		return r, errors.Wrap(err, "unable to find or add guild")
	}

	s := storage.GuildSettings{}
	bGuild.SetSettings(s)

	err = t.SaveGuild(bGuild)
	if err != nil {
		return r, errors.Wrap(err, "could not save guild settings")
	}

	err = t.Commit()
	if err != nil {
		return r, errors.Wrap(err, "could not save guild settings")
	}

	return c.list(msg)
}

// ConfigCommandHandler creates a new command handler for !config-su commands
func ConfigCommandHandler(deps configDependencies, versionStr, preCommand string) (*cmdhandler.CommandHandler, error) {
	p := parser.NewParser(parser.Options{
		CmdIndicator: " ",
	})
	cc := configCommands{
		preCommand: preCommand,
		deps:       deps,
		versionStr: versionStr,
	}

	ch, err := cmdhandler.NewCommandHandler(p, cmdhandler.Options{
		PreCommand:          preCommand,
		Placeholder:         "action",
		HelpOnEmptyCommands: true,
	})
	if err != nil {
		return nil, err
	}

	ch.SetHandler("list", cmdhandler.NewMessageHandler(cc.list))
	ch.SetHandler("get", cmdhandler.NewMessageHandler(cc.get))
	ch.SetHandler("set", cmdhandler.NewMessageHandler(cc.set))
	ch.SetHandler("reset", cmdhandler.NewMessageHandler(cc.reset))
	ch.SetHandler("version", cmdhandler.NewMessageHandler(cc.version))

	return ch, err
}
