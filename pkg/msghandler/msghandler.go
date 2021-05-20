package msghandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/logging/level"
	"github.com/gsmcwhirter/go-util/v8/parser"
	"github.com/gsmcwhirter/go-util/v8/telemetry"
	"golang.org/x/time/rate"

	"github.com/gsmcwhirter/discord-bot-lib/v19/bot"
	"github.com/gsmcwhirter/discord-bot-lib/v19/bot/session"
	"github.com/gsmcwhirter/discord-bot-lib/v19/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v19/dispatcher"
	"github.com/gsmcwhirter/discord-bot-lib/v19/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v19/request"
	"github.com/gsmcwhirter/discord-bot-lib/v19/snowflake"
	"github.com/gsmcwhirter/discord-bot-lib/v19/wsapi"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/reactions"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/stats"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

// ErrUnauthorized is the error a command handler should return if the user does
// not have permission to perform the requested action
var ErrUnauthorized = errors.New("unauthorized")

// ErrNoResponse is the error a command handler should return
// if the bot should not produce a response
var ErrNoResponse = errors.New("no response")

type dependencies interface {
	Logger() Logger
	GuildAPI() storage.GuildAPI
	CommandHandler() *cmdhandler.CommandHandler
	ConfigHandler() *cmdhandler.CommandHandler
	DebugHandler() *cmdhandler.CommandHandler
	AdminHandler() *cmdhandler.CommandHandler
	ReactionHandler() reactions.Handler
	MessageRateLimiter() *rate.Limiter
	ReactionsRateLimiter() *rate.Limiter
	BotSession() *session.Session
	Census() *telemetry.Census
	StatsHub() *stats.Hub
	SendAllowed() bool
}

// Handlers is the interface for a Handlers dependency that registers itself with a discrord bot
type Handlers interface {
	ConnectToBot(*bot.DiscordBot)
}

type handlers struct {
	bot                     *bot.DiscordBot
	deps                    dependencies
	defaultCommandIndicator string
	successColor            int
	errorColor              int
}

// Options provides a way to pass configuration to NewHandlers
type Options struct {
	DefaultCommandIndicator string
	SuccessColor            int
	ErrorColor              int
}

// NewHandlers creates a new Handlers object
func NewHandlers(deps dependencies, opts Options) Handlers {
	h := handlers{
		deps:                    deps,
		defaultCommandIndicator: opts.DefaultCommandIndicator,
		successColor:            opts.SuccessColor,
		errorColor:              opts.ErrorColor,
	}

	return &h
}

func (h *handlers) ConnectToBot(b *bot.DiscordBot) {
	h.bot = b

	b.AddMessageHandler("MESSAGE_CREATE", h.handleMessage)
	b.AddMessageHandler("MESSAGE_REACTION_ADD", h.handleReactionAdd)
	b.AddMessageHandler("MESSAGE_REACTION_REMOVE", h.handleReactionRemove)
}

func (h *handlers) channelGuild(cid snowflake.Snowflake) (gid snowflake.Snowflake) {
	gid, _ = h.deps.BotSession().GuildOfChannel(cid)
	return
}

func (h *handlers) guildCommandIndicator(ctx context.Context, gid snowflake.Snowflake) string {
	ctx, span := h.deps.Census().StartSpan(ctx, "handlers.guildCommandIndicator")
	defer span.End()

	if gid == 0 {
		return h.defaultCommandIndicator
	}

	s, err := storage.GetSettings(ctx, h.deps.GuildAPI(), gid)
	if err != nil {
		return h.defaultCommandIndicator
	}

	if s.ControlSequence == "" {
		return h.defaultCommandIndicator
	}

	return s.ControlSequence
}

func (h *handlers) attemptConfigAndAdminHandlers(msg cmdhandler.Message, cmdIndicator, content string) (cmdhandler.Response, error) {
	ctx, span := h.deps.Census().StartSpan(msg.Context(), "handlers.attemptConfigAndAdminHandlers", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	logger := logging.WithMessage(msg, h.deps.Logger())

	s, err := storage.GetSettings(msg.Context(), h.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		level.Error(logger).Err("could not retrieve guild settings", err)
	}

	if !IsAdminAuthorized(ctx, logger, msg, s.AdminRoles, h.deps.BotSession(), h.bot) {
		level.Info(logger).Message("non-admin trying to config")
		return nil, ErrUnauthorized
	}

	level.Debug(logger).Message("admin trying to config")

	level.Info(logger).Message("processing debug command", "cmdContent", fmt.Sprintf("%q", content))
	resp, err := h.deps.DebugHandler().HandleMessage(cmdhandler.NewWithContents(msg, content))
	if err == nil {
		return resp, nil
	}

	if e2, ok := err.(errors.Error); ok && e2.Unwrap() != nil {
		err = e2.Unwrap()
	}

	if err != ErrUnauthorized && err != parser.ErrUnknownCommand && err != parser.ErrNotACommand {
		return resp, err
	}

	cmdContent := h.deps.ConfigHandler().CommandIndicator() + strings.TrimPrefix(content, cmdIndicator)
	level.Info(logger).Message("processing command", "cmdContent", fmt.Sprintf("%q", cmdContent), "rawCmd", fmt.Sprintf("%q", content))
	resp, err = h.deps.ConfigHandler().HandleMessage(cmdhandler.NewWithContents(msg, cmdContent))

	if err == nil {
		return resp, nil
	}

	if err != ErrUnauthorized && err != parser.ErrUnknownCommand {
		return resp, err
	}

	level.Debug(logger).Message("admin trying to admin")
	cmdContent = h.deps.AdminHandler().CommandIndicator() + strings.TrimPrefix(content, cmdIndicator)
	return h.deps.AdminHandler().HandleMessage(cmdhandler.NewWithContents(msg, cmdContent))
}

func (h *handlers) handleResponse(ctx context.Context, logger Logger, resp cmdhandler.Response, cid, gid snowflake.Snowflake, content string, err error) {
	if err == ErrNoResponse || err == parser.ErrUnknownCommand {
		return
	}

	if err != nil {
		level.Error(logger).Err("error handling command", err, "contents", content)
		resp.IncludeError(err)
	}

	if resp.HasErrors() {
		resp.SetColor(h.errorColor)
	} else {
		resp.SetColor(h.successColor)
	}

	level.Info(logger).Message("sending message", "resp", fmt.Sprintf("%+v", resp))

	sendTo := resp.Channel()
	if sendTo == 0 {
		sendTo = cid
	}

	splitResp := resp.Split()

	level.Info(logger).Message("sending message split", "split_count", len(splitResp))

	for _, res := range splitResp {
		if !h.deps.SendAllowed() {
			level.Info(logger).Message("message send disabled", "message_to_send", fmt.Sprintf("%#v", res.ToMessage()), "reactions", res.MessageReactions())
			continue
		}

		err = h.deps.MessageRateLimiter().Wait(ctx)
		if err != nil {
			level.Error(logger).Err("error waiting for ratelimiting", err)
			return
		}

		if ar, ok := h.deps.StatsHub().Get("msg_sent"); ok {
			ar.Incr(1)
		}

		sentMsg, err := h.bot.API().SendMessage(ctx, sendTo, res.ToMessage())
		if err != nil {
			level.Error(logger).Err("could not send message", err)
			return
		}

		reacts := res.MessageReactions()
		for _, reaction := range reacts {
			err = h.deps.ReactionsRateLimiter().Wait(ctx)
			if err != nil {
				level.Error(logger).Err("error waiting for ratelimiting for reaction", err)
				return
			}

			if ar, ok := h.deps.StatsHub().Get("reaction_sent"); ok {
				ar.Incr(1)
			}

			resp, err := h.bot.API().CreateReaction(ctx, sendTo, sentMsg.IDSnowflake, reaction)
			if err != nil {
				status := 0
				if resp != nil {
					status = resp.StatusCode
				}

				level.Error(logger).Err("could not add reaction", err, "status_code", status)
			}
		}
	}

	level.Info(logger).Message("successfully sent message(s) to channel", "channel_id", sendTo.ToString(), "message_ct", len(splitResp))
}

func (h *handlers) handleMessage(p bot.Payload, req wsapi.WSMessage, respChan chan<- wsapi.WSMessage) snowflake.Snowflake {
	ctx, span := h.deps.Census().StartSpan(req.Ctx, "handlers.handleMessage")
	defer span.End()
	req.Ctx = ctx

	if h.bot == nil {
		return 0
	}

	select {
	case <-req.Ctx.Done():
		return 0
	default:
	}

	logger := logging.WithContext(req.Ctx, h.deps.Logger())

	m, err := dispatcher.MessageFromElementMap(p.Contents())
	if err != nil {
		level.Error(logger).Err("error inflating message", err)
		return 0
	}

	if m.MessageType() != dispatcher.DefaultMessage {
		level.Info(logger).Message("message was not a default type")
		return 0
	}

	gid := h.channelGuild(m.ChannelID())
	req.Ctx = request.WithGuildID(req.Ctx, gid)
	logger = logging.WithContext(req.Ctx, h.deps.Logger())

	content := m.ContentString()
	if content == "" {
		level.Info(logger).Message("message contents empty")
		return gid
	}

	cmdIndicator := h.guildCommandIndicator(req.Ctx, gid)

	if !strings.HasPrefix(content, cmdIndicator) && !strings.HasPrefix(content, "!config-su-debug") {
		level.Info(logger).Message("not a command")
		return gid
	}

	if ar, ok := h.deps.StatsHub().Get("msgs"); ok {
		ar.Incr(1)
	}

	content = strings.TrimSpace(content)

	msg := cmdhandler.NewSimpleMessage(req.Ctx, m.AuthorID(), gid, m.ChannelID(), m.ID(), "")
	logger = logging.WithMessage(msg, h.deps.Logger())
	resp, err := h.attemptConfigAndAdminHandlers(msg, cmdIndicator, content)

	if err != nil && (err == ErrUnauthorized || err == parser.ErrUnknownCommand) {
		level.Debug(logger).Message("admin not successful; processing as real message")
		cmdContent := h.deps.CommandHandler().CommandIndicator() + strings.TrimPrefix(content, cmdIndicator)
		resp, err = h.deps.CommandHandler().HandleMessage(cmdhandler.NewWithContents(msg, cmdContent))
	}

	h.handleResponse(req.Ctx, logger, resp, m.ChannelID(), gid, content, err)

	return gid
}

func (h *handlers) handleReactionAdd(p bot.Payload, req wsapi.WSMessage, respChan chan<- wsapi.WSMessage) snowflake.Snowflake {
	ctx, span := h.deps.Census().StartSpan(req.Ctx, "handlers.handleReactionAdd")
	defer span.End()
	req.Ctx = ctx

	if h.bot == nil {
		return 0
	}

	select {
	case <-req.Ctx.Done():
		return 0
	default:
	}

	logger := logging.WithContext(req.Ctx, h.deps.Logger())

	r, err := dispatcher.ReactionFromElementMap(p.Contents())
	if err != nil {
		level.Error(logger).Err("error inflating reaction", err)
		return 0
	}

	gid := h.channelGuild(r.ChannelID())
	req.Ctx = request.WithGuildID(req.Ctx, gid)
	reaction := reactions.NewReaction(req.Ctx, r.UserID(), r.MessageID(), r.ChannelID(), r.GuildID(), r.Emoji())
	logger = reactions.LoggerWithReaction(reaction, h.deps.Logger())

	if ar, ok := h.deps.StatsHub().Get("reactions"); ok {
		ar.Incr(1)
	}

	resp, err := h.deps.ReactionHandler().HandleReactionAdd(reaction)

	h.handleResponse(req.Ctx, logger, resp, r.ChannelID(), gid, r.Emoji(), err)

	return gid
}

func (h *handlers) handleReactionRemove(p bot.Payload, req wsapi.WSMessage, respChan chan<- wsapi.WSMessage) snowflake.Snowflake {
	ctx, span := h.deps.Census().StartSpan(req.Ctx, "handlers.handleReactionAdd")
	defer span.End()
	req.Ctx = ctx

	if h.bot == nil {
		return 0
	}

	select {
	case <-req.Ctx.Done():
		return 0
	default:
	}

	logger := logging.WithContext(req.Ctx, h.deps.Logger())

	r, err := dispatcher.ReactionFromElementMap(p.Contents())
	if err != nil {
		level.Error(logger).Err("error inflating reaction", err)
		return 0
	}

	gid := h.channelGuild(r.ChannelID())
	req.Ctx = request.WithGuildID(req.Ctx, gid)
	reaction := reactions.NewReaction(req.Ctx, r.UserID(), r.MessageID(), r.ChannelID(), r.GuildID(), r.Emoji())
	logger = reactions.LoggerWithReaction(reaction, h.deps.Logger())

	if ar, ok := h.deps.StatsHub().Get("reactions"); ok {
		ar.Incr(1)
	}

	resp, err := h.deps.ReactionHandler().HandleReactionRemove(reaction)

	h.handleResponse(req.Ctx, logger, resp, r.ChannelID(), gid, r.Emoji(), err)

	return gid
}
