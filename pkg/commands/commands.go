package commands

import (
	"fmt"

	"github.com/gsmcwhirter/discord-bot-lib/v18/bot"
	"github.com/gsmcwhirter/discord-bot-lib/v18/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v18/etfapi"
	"github.com/gsmcwhirter/discord-bot-lib/v18/logging"
	"github.com/gsmcwhirter/go-util/v7/parser"
	"github.com/gsmcwhirter/go-util/v7/telemetry"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

type dependencies interface {
	Logger() logging.Logger
	TrialAPI() storage.TrialAPI
	GuildAPI() storage.GuildAPI
	BotSession() *etfapi.Session
	Bot() bot.DiscordBot
	Census() *telemetry.Census
}

// Options is the way to specify the command indicator string
type Options struct {
	CmdIndicator string
}

// RootCommands holds the commands at the root level
type userCommands struct {
	deps dependencies
}

// CommandHandler creates a new command handler for !list, !show, !signup, and !withdraw
func CommandHandler(deps dependencies, versionStr string, opts Options) (*cmdhandler.CommandHandler, error) {
	p := parser.NewParser(parser.Options{
		CmdIndicator: opts.CmdIndicator,
	})
	rh := userCommands{
		deps: deps,
	}

	ch, err := cmdhandler.NewCommandHandler(p, cmdhandler.Options{
		NoHelpOnUnknownCommands: true,
	})
	if err != nil {
		return nil, err
	}

	ch.SetHandler("list", cmdhandler.NewMessageHandler(rh.list))
	ch.SetHandler("show", cmdhandler.NewMessageHandler(rh.show))
	ch.SetHandler("signup", cmdhandler.NewMessageHandler(rh.signup))
	ch.SetHandler("su", cmdhandler.NewMessageHandler(rh.signup))
	ch.SetHandler("withdraw", cmdhandler.NewMessageHandler(rh.withdraw))
	ch.SetHandler("wd", cmdhandler.NewMessageHandler(rh.withdraw))

	return ch, nil
}

type configDependencies interface {
	Logger() logging.Logger
	GuildAPI() storage.GuildAPI
	TrialAPI() storage.TrialAPI
	BotSession() *etfapi.Session
	Bot() bot.DiscordBot
	Census() *telemetry.Census
}

// ConfigHandler creates a new command handler for !config-su
func ConfigHandler(deps configDependencies, versionStr string, opts Options) (*cmdhandler.CommandHandler, error) {
	p := parser.NewParser(parser.Options{
		CmdIndicator: opts.CmdIndicator,
	})

	ch, err := cmdhandler.NewCommandHandler(p, cmdhandler.Options{
		NoHelpOnUnknownCommands: true,
	})
	if err != nil {
		return nil, err
	}

	cch, err := ConfigCommandHandler(deps, versionStr, fmt.Sprintf("%sconfig-su", opts.CmdIndicator))
	if err != nil {
		return nil, err
	}

	ch.SetHandler("config-su", cch)

	// disable help for config
	ch.SetHandler("help", cmdhandler.NewMessageHandler(func(msg cmdhandler.Message) (cmdhandler.Response, error) {
		r := &cmdhandler.SimpleEmbedResponse{}
		return r, parser.ErrUnknownCommand
	}))

	return ch, nil
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
	Logger() logging.Logger
	GuildAPI() storage.GuildAPI
	TrialAPI() storage.TrialAPI
	BotSession() *etfapi.Session
	Bot() bot.DiscordBot
	Census() *telemetry.Census
}

// AdminHandler creates a new command handler for !admin
func AdminHandler(deps adminDependencies, versionStr string, opts Options) (*cmdhandler.CommandHandler, error) {
	p := parser.NewParser(parser.Options{
		CmdIndicator: opts.CmdIndicator,
	})

	ch, err := cmdhandler.NewCommandHandler(p, cmdhandler.Options{
		NoHelpOnUnknownCommands: true,
	})
	if err != nil {
		return nil, err
	}

	ach, err := AdminCommandHandler(deps, fmt.Sprintf("%sadmin", opts.CmdIndicator))
	if err != nil {
		return nil, err
	}

	ch.SetHandler("admin", ach)
	// disable help for admin
	ch.SetHandler("help", cmdhandler.NewMessageHandler(func(msg cmdhandler.Message) (cmdhandler.Response, error) {
		r := &cmdhandler.SimpleEmbedResponse{}
		return r, parser.ErrUnknownCommand
	}))

	return ch, nil
}
