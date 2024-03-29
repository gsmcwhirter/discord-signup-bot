package reactions

import "github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"

type Handler interface {
	HandleReactionAdd(Reaction) (cmdhandler.Response, error)
	HandleReactionRemove(Reaction) (cmdhandler.Response, error)
}
