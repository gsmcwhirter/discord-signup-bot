package commands

import (
	"context"
	"strings"

	"github.com/gsmcwhirter/discord-bot-lib/v20/bot"
	"github.com/gsmcwhirter/discord-bot-lib/v20/bot/session"
	"github.com/gsmcwhirter/discord-bot-lib/v20/cmdhandler"
	"github.com/gsmcwhirter/go-util/v8/logging/level"
	"github.com/gsmcwhirter/go-util/v8/telemetry"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/reactions"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

type reactionDependencies interface {
	Logger() Logger
	TrialAPI() storage.TrialAPI
	GuildAPI() storage.GuildAPI
	BotSession() *session.Session
	Bot() *bot.DiscordBot
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
	if r.UserID().ToString() == h.deps.Bot().Config().ClientID {
		return nil, msghandler.ErrNoResponse
	}

	return h.signup(r)
}

func (h *reactionHandler) HandleReactionRemove(r reactions.Reaction) (cmdhandler.Response, error) {
	if r.UserID().ToString() == h.deps.Bot().Config().ClientID {
		return nil, msghandler.ErrNoResponse
	}

	return h.withdraw(r)
}

func (h *reactionHandler) getTrialNameForReaction(ctx context.Context, logger Logger, msg reactions.Reaction) (string, error) {
	msgInfo, err := h.deps.Bot().API().GetMessage(ctx, msg.ChannelID(), msg.MessageID())
	if err != nil {
		level.Error(logger).Err("could not get message information", err)
		return "", msghandler.ErrNoResponse
	}

	// was the message reacted to mine?
	if msgInfo.Author.IDSnowflake.ToString() != h.deps.Bot().Config().ClientID {
		level.Debug(logger).Message("reaction was not to our message")
		return "", msghandler.ErrNoResponse
	}

	// pull the trialName out of the message
	if len(msgInfo.Embeds) != 1 {
		level.Info(logger).Message("reaction was not to a message with a single embed", "num_embeds", len(msgInfo.Embeds))
		return "", msghandler.ErrNoResponse
	}

	if !strings.HasPrefix(msgInfo.Embeds[0].Footer.Text, "event:") {
		level.Info(logger).Message("reaction was not to a message with an event: footer", "footer_text", msgInfo.Embeds[0].Footer.Text)
		return "", msghandler.ErrNoResponse
	}

	trialName := strings.TrimSpace(msgInfo.Embeds[0].Footer.Text[6:])
	return trialName, nil
}
