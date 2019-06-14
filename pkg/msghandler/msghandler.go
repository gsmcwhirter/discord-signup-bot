package msghandler

import (
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v3/errors"
	log "github.com/gsmcwhirter/go-util/v3/logging"
	"github.com/gsmcwhirter/go-util/v3/logging/level"
	"github.com/gsmcwhirter/go-util/v3/parser"
	"golang.org/x/time/rate"

	"github.com/gsmcwhirter/discord-bot-lib/v7/bot"
	"github.com/gsmcwhirter/discord-bot-lib/v7/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v7/etfapi"
	"github.com/gsmcwhirter/discord-bot-lib/v7/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v7/snowflake"
	"github.com/gsmcwhirter/discord-bot-lib/v7/wsclient"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

// ErrUnauthorized is the error a command handler should return if the user does
// not have permission to perform the requested action
var ErrUnauthorized = errors.New("unauthorized")

// ErrNoResponse is the error a command handler should return
// if the bot should not produce a response
var ErrNoResponse = errors.New("no response")

type dependencies interface {
	Logger() log.Logger
	GuildAPI() storage.GuildAPI
	CommandHandler() *cmdhandler.CommandHandler
	ConfigHandler() *cmdhandler.CommandHandler
	AdminHandler() *cmdhandler.CommandHandler
	MessageRateLimiter() *rate.Limiter
	BotSession() *etfapi.Session
}

// Handlers is the interface for a Handlers dependency that registers itself with a discrord bot
type Handlers interface {
	ConnectToBot(bot.DiscordBot)
}

type handlers struct {
	bot                     bot.DiscordBot
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

func (h *handlers) ConnectToBot(b bot.DiscordBot) {
	h.bot = b

	b.AddMessageHandler("MESSAGE_CREATE", h.handleMessage)
}

func (h *handlers) channelGuild(cid snowflake.Snowflake) (gid snowflake.Snowflake) {
	gid, _ = h.deps.BotSession().GuildOfChannel(cid)
	return
}

func (h *handlers) guildCommandIndicator(gid snowflake.Snowflake) string {
	if gid == 0 {
		return h.defaultCommandIndicator
	}

	s, err := storage.GetSettings(h.deps.GuildAPI(), gid)
	if err != nil {
		return h.defaultCommandIndicator
	}

	if s.ControlSequence == "" {
		return h.defaultCommandIndicator
	}

	return s.ControlSequence
}

func (h *handlers) attemptConfigAndAdminHandlers(msg cmdhandler.Message, req wsclient.WSMessage, cmdIndicator, content string) (cmdhandler.Response, error) {
	logger := logging.WithMessage(msg, h.deps.Logger())

	s, err := storage.GetSettings(h.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		level.Error(logger).Err("could not retrieve guild settings", err)
	}

	if !IsAdminAuthorized(logger, msg, s.AdminRole, h.deps.BotSession()) {
		level.Info(logger).Message("non-admin trying to config")
		return nil, ErrUnauthorized
	}

	level.Debug(logger).Message("admin trying to config")
	cmdContent := h.deps.ConfigHandler().CommandIndicator() + strings.TrimPrefix(content, cmdIndicator)
	level.Info(logger).Message("processing command", "cmdContent", fmt.Sprintf("%q", cmdContent), "rawCmd", fmt.Sprintf("%q", content))
	resp, err := h.deps.ConfigHandler().HandleMessage(cmdhandler.NewWithContents(msg, cmdContent))

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

func (h *handlers) handleMessage(p *etfapi.Payload, req wsclient.WSMessage, respChan chan<- wsclient.WSMessage) {
	if h.bot == nil {
		return
	}

	select {
	case <-req.Ctx.Done():
		return
	default:
	}

	logger := logging.WithContext(req.Ctx, h.deps.Logger())

	m, err := etfapi.MessageFromElementMap(p.Data)
	if err != nil {
		level.Error(logger).Err("error inflating message", err)
		return
	}

	if m.MessageType() != etfapi.DefaultMessage {
		level.Info(logger).Message("message was not a default type")
		return
	}

	content := m.ContentString()
	if content == "" {
		level.Info(logger).Message("message contents empty")
		return
	}

	gid := h.channelGuild(m.ChannelID())
	cmdIndicator := h.guildCommandIndicator(gid)

	if !strings.HasPrefix(content, cmdIndicator) {
		level.Info(logger).Message("not a command")
		return
	}

	content = strings.TrimSpace(content)

	msg := cmdhandler.NewSimpleMessage(req.Ctx, m.AuthorID(), gid, m.ChannelID(), m.ID(), "")
	logger = logging.WithMessage(msg, h.deps.Logger())
	resp, err := h.attemptConfigAndAdminHandlers(msg, req, cmdIndicator, content)

	if err != nil && (err == ErrUnauthorized || err == parser.ErrUnknownCommand) {
		level.Debug(logger).Message("admin not successful; processing as real message")
		cmdContent := h.deps.CommandHandler().CommandIndicator() + strings.TrimPrefix(content, cmdIndicator)
		resp, err = h.deps.CommandHandler().HandleMessage(cmdhandler.NewWithContents(msg, cmdContent))
	}

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
		sendTo = m.ChannelID()
	}

	splitResp := resp.Split()

	level.Info(logger).Message("sending message split", "split_count", len(splitResp))

	for _, res := range splitResp {
		err = h.deps.MessageRateLimiter().Wait(req.Ctx)
		if err != nil {
			level.Error(logger).Err("error waiting for ratelimiting", err)
			return
		}

		sendResp, body, err := h.bot.SendMessage(req.Ctx, sendTo, res.ToMessage())
		if err != nil {
			var bodyStr string
			if body != nil {
				bodyStr = string(body)
			}
			level.Error(logger).Err("could not send message", err, "resp_body", bodyStr, "status_code", sendResp.StatusCode)
			return
		}
	}

	level.Info(logger).Message("successfully sent message(s) to channel", "channel_id", sendTo.ToString(), "message_ct", len(splitResp))
}
