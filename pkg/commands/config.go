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

	return r, msg.ContentErr()
}

func (c *configCommands) website(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To:          cmdhandler.UserMentionString(msg.UserID()),
		Description: "https://www.evogames.org/bots/eso-signup-bot/",
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling configCommand", "command", "website")

	return r, msg.ContentErr()
}

func (c *configCommands) discord(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To:          cmdhandler.UserMentionString(msg.UserID()),
		Description: "https://discord.gg/BgkvvbN",
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling configCommand", "command", "discord")

	return r, msg.ContentErr()
}

func (c *configCommands) list(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling configCommand", "command", "list")

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

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

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling adminCommand", "command", "get", "args", msg.Contents())

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 1 {
		return r, errors.New("missing setting name")
	}

	if len(msg.Contents()) > 1 {
		return r, errors.New("too many arguments")
	}

	settingName := strings.TrimSpace(msg.Contents()[0])

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

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling adminCommand", "command", "set", "set_args", msg.Contents())

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
			rid, ok := g.RoleWithName(argPairList[1])
			if !ok {
				return r, fmt.Errorf("could not find role with name '%s'", argPairList[1])
			}

			ap.val = rid.ToString()
		default:
			ap.val = argPairList[1]
		}

		argPairs = append(argPairs, ap)
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

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

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

type stat struct {
	trials int
	open   int
	closed int
}

func (c *configCommands) collectStats(gid string) (stat, error) {
	s := stat{}

	t, err := c.deps.TrialAPI().NewTransaction(gid, false)
	if err != nil {
		return s, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	trials := t.GetTrials()

	for _, trial := range trials {
		s.trials++
		if trial.GetState() == storage.TrialStateClosed {
			s.closed++
		} else {
			s.open++
		}
	}

	return s, nil
}

func (c *configCommands) stats(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling adminCommand", "command", "stats")

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	allGuilds, err := c.deps.GuildAPI().AllGuilds()
	if err != nil {
		return r, err
	}

	s := stat{}

	for _, guild := range allGuilds {
		stat, err := c.collectStats(guild)
		if err != nil {
			return r, err
		}

		s.trials += stat.trials
		s.open += stat.open
		s.closed += stat.closed
	}

	r.Description = fmt.Sprintf("Total guilds: %d\nTotal events: %d\nCurrently open: %d\nCurrently closed: %d\n", len(allGuilds), s.trials, s.open, s.closed)
	return r, nil
}

// ConfigCommandHandler creates a new command handler for !config-su commands
func ConfigCommandHandler(deps configDependencies, versionStr, preCommand string) (*cmdhandler.CommandHandler, error) {
	p := parser.NewParser(parser.Options{
		CmdIndicator: "",
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
	ch.SetHandler("website", cmdhandler.NewMessageHandler(cc.website))
	ch.SetHandler("discord", cmdhandler.NewMessageHandler(cc.discord))
	ch.SetHandler("stats", cmdhandler.NewMessageHandler(cc.stats))

	return ch, err
}
