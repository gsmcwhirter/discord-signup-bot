package commands

import (
	"fmt"

	"github.com/gsmcwhirter/discord-bot-lib/v23/bot"
	"github.com/gsmcwhirter/discord-bot-lib/v23/bot/session"
	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/go-util/v8/parser"
	"github.com/gsmcwhirter/go-util/v8/telemetry"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/permissions"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/stats"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

type dependencies interface {
	Logger() Logger
	TrialAPI() storage.TrialAPI
	GuildAPI() storage.GuildAPI
	BotSession() *session.Session
	Bot() *bot.DiscordBot
	Census() *telemetry.Census
}

// Options is the way to specify the command indicator string
type Options struct {
	CmdIndicator string
}

// CommandHandler creates a new command handler for !list, !show, !signup, and !withdraw
func CommandHandler(deps dependencies, versionStr string, opts Options) (*UserCommands, *cmdhandler.CommandHandler, error) {
	p := parser.NewParser(parser.Options{
		CmdIndicator: opts.CmdIndicator,
	})
	uc := &UserCommands{
		deps: deps,
	}

	ch, err := cmdhandler.NewCommandHandler(p, cmdhandler.Options{
		NoHelpOnUnknownCommands: true,
	})
	if err != nil {
		return nil, nil, err
	}

	uc.AttachToCommandHandler(ch)

	return uc, ch, nil
}

type configDependencies interface {
	Logger() Logger
	GuildAPI() storage.GuildAPI
	TrialAPI() storage.TrialAPI
	BotSession() *session.Session
	Bot() *bot.DiscordBot
	Census() *telemetry.Census
	StatsHub() *stats.Hub
	PermissionsManager() *permissions.Manager
}

// ConfigHandler creates a new command handler for !config-su
func ConfigHandler(deps configDependencies, versionStr string, opts Options) (*ConfigCommands, *cmdhandler.CommandHandler, error) {
	p := parser.NewParser(parser.Options{
		CmdIndicator: opts.CmdIndicator,
	})

	ch, err := cmdhandler.NewCommandHandler(p, cmdhandler.Options{
		NoHelpOnUnknownCommands: true,
	})
	if err != nil {
		return nil, nil, err
	}

	cc, cch, err := ConfigCommandHandler(deps, versionStr, fmt.Sprintf("%sconfig-su", opts.CmdIndicator))
	if err != nil {
		return nil, nil, err
	}

	ch.SetHandler("config-su", cch)

	// disable help for config
	ch.SetHandler("help", cmdhandler.NewMessageHandler(func(msg cmdhandler.Message) (cmdhandler.Response, error) {
		r := &cmdhandler.SimpleEmbedResponse{}
		return r, parser.ErrUnknownCommand
	}))

	return cc, ch, nil
}

func ConfigDebugHandler(deps configDependencies) (*cmdhandler.CommandHandler, error) {
	p := parser.NewParser(parser.Options{
		CmdIndicator: "!", // yes, hard-code this here
	})

	ch, err := cmdhandler.NewCommandHandler(p, cmdhandler.Options{
		NoHelpOnUnknownCommands: false,
	})
	if err != nil {
		return nil, err
	}

	cch, err := ConfigDebugCommandHandler(deps, "!config-su-debug")
	if err != nil {
		return nil, err
	}

	ch.SetHandler("config-su-debug", cch)

	// disable help for config
	ch.SetHandler("help", cmdhandler.NewMessageHandler(func(msg cmdhandler.Message) (cmdhandler.Response, error) {
		r := &cmdhandler.SimpleEmbedResponse{}
		return r, parser.ErrUnknownCommand
	}))

	return ch, nil
}

type adminDependencies interface {
	Logger() Logger
	GuildAPI() storage.GuildAPI
	TrialAPI() storage.TrialAPI
	BotSession() *session.Session
	Bot() *bot.DiscordBot
	Census() *telemetry.Census
}

// AdminHandler creates a new command handler for !admin
func AdminHandler(deps adminDependencies, versionStr string, opts Options) (*AdminCommands, *cmdhandler.CommandHandler, error) {
	p := parser.NewParser(parser.Options{
		CmdIndicator: opts.CmdIndicator,
	})

	ch, err := cmdhandler.NewCommandHandler(p, cmdhandler.Options{
		NoHelpOnUnknownCommands: true,
	})
	if err != nil {
		return nil, nil, err
	}

	ac, ach, err := AdminCommandHandler(deps, fmt.Sprintf("%sadmin", opts.CmdIndicator))
	if err != nil {
		return nil, nil, err
	}

	ch.SetHandler("admin", ach)
	// disable help for admin
	ch.SetHandler("help", cmdhandler.NewMessageHandler(func(msg cmdhandler.Message) (cmdhandler.Response, error) {
		r := &cmdhandler.SimpleEmbedResponse{}
		return r, parser.ErrUnknownCommand
	}))

	return ac, ch, nil
}
