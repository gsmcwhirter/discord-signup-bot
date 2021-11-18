package commands

import (
	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
)

type InteractionCommandHandler struct {
	command      entity.ApplicationCommand
	handler      cmdhandler.InteractionHandler
	autocomplete cmdhandler.AutocompleteHandler
}

var _ cmdhandler.InteractionCommandHandler = (*InteractionCommandHandler)(nil)

func (i *InteractionCommandHandler) Command() entity.ApplicationCommand     { return i.command }
func (i *InteractionCommandHandler) Handler() cmdhandler.InteractionHandler { return i.handler }
func (i *InteractionCommandHandler) AutocompleteHandler() cmdhandler.AutocompleteHandler {
	return i.autocomplete
}
