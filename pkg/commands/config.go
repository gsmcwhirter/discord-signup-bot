package commands

import (
	"github.com/gsmcwhirter/discord-bot-lib/cmdhandler"
	"github.com/gsmcwhirter/go-util/parser"
)

type configCommands struct {
	preCommand string
	versionStr string
	deps       configDependencies
}

type stat struct {
	trials int
	open   int
	closed int
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
