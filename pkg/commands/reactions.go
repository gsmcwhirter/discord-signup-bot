package commands

import (
	"github.com/gsmcwhirter/discord-bot-lib/v18/bot"
	"github.com/gsmcwhirter/discord-bot-lib/v18/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v18/etfapi"
	"github.com/gsmcwhirter/discord-bot-lib/v18/logging"
	"github.com/gsmcwhirter/go-util/v7/telemetry"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/reactions"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

type reactionDependencies interface {
	Logger() logging.Logger
	TrialAPI() storage.TrialAPI
	GuildAPI() storage.GuildAPI
	BotSession() *etfapi.Session
	Bot() bot.DiscordBot
	Census() *telemetry.Census
}

type reactionHandler struct {
	deps reactionDependencies
}

var _ reactions.Handler = (*reactionHandler)(nil)

func NewReactionHandler(deps reactionDependencies) reactions.Handler {
	return &reactionHandler{
		deps: deps,
	}
}

func (h *reactionHandler) HandleReactionAdd(r reactions.Reaction) (cmdhandler.Response, error) {
	return h.signup(r)
}

func (h *reactionHandler) HandleReactionRemove(r reactions.Reaction) (cmdhandler.Response, error) {
	return nil, msghandler.ErrNoResponse
}
