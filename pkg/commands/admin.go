package commands

import (
	"github.com/gsmcwhirter/go-util/v2/parser"
	"github.com/pkg/errors"

	"github.com/gsmcwhirter/discord-bot-lib/v6/cmdhandler"
)

// ErrGuildNotFound is the error returned when a guild is not known about
// in a BotSession
var ErrGuildNotFound = errors.New("guild not found")

type adminCommands struct {
	preCommand string
	deps       adminDependencies
}

// AdminCommandHandler creates a new command handler for !admin commands
func AdminCommandHandler(deps adminDependencies, preCommand string) (*cmdhandler.CommandHandler, error) {
	p := parser.NewParser(parser.Options{
		CmdIndicator: "",
	})
	cc := adminCommands{
		preCommand: preCommand,
		deps:       deps,
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
	ch.SetHandler("create", cmdhandler.NewMessageHandler(cc.create))
	ch.SetHandler("edit", cmdhandler.NewMessageHandler(cc.edit))
	ch.SetHandler("open", cmdhandler.NewMessageHandler(cc.open))
	ch.SetHandler("close", cmdhandler.NewMessageHandler(cc.close))
	ch.SetHandler("delete", cmdhandler.NewMessageHandler(cc.delete))
	ch.SetHandler("announce", cmdhandler.NewMessageHandler(cc.announce))
	ch.SetHandler("grouping", cmdhandler.NewMessageHandler(cc.grouping))
	ch.SetHandler("signup", cmdhandler.NewMessageHandler(cc.signup))
	ch.SetHandler("su", cmdhandler.NewMessageHandler(cc.signup))
	ch.SetHandler("withdraw", cmdhandler.NewMessageHandler(cc.withdraw))
	ch.SetHandler("wd", cmdhandler.NewMessageHandler(cc.withdraw))
	ch.SetHandler("clear", cmdhandler.NewMessageHandler(cc.clear))
	ch.SetHandler("show", cmdhandler.NewMessageHandler(cc.show))

	return ch, nil
}
