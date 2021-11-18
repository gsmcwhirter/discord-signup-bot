package main

import (
	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
)

type interaction struct {
	command entity.ApplicationCommand
	handler cmdhandler.InteractionHandler
}

func (i *interaction) Command() entity.ApplicationCommand     { return i.command }
func (i *interaction) Handler() cmdhandler.InteractionHandler { return i.handler }
