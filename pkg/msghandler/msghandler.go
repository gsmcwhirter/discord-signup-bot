package msghandler

import (
	"fmt"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gsmcwhirter/discord-bot-lib/bot"
	"github.com/gsmcwhirter/discord-bot-lib/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/etfapi"
	"github.com/gsmcwhirter/discord-bot-lib/logging"
	"github.com/gsmcwhirter/discord-bot-lib/snowflake"
	"github.com/gsmcwhirter/discord-bot-lib/wsclient"
	"github.com/gsmcwhirter/go-util/parser"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

var errUnauthorized = errors.New("unauthorized")

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

func (h *handlers) ConnectToBot(bot bot.DiscordBot) {
	h.bot = bot

	bot.AddMessageHandler("MESSAGE_CREATE", h.handleMessage)
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

func (h *handlers) attemptConfigAndAdminHandlers(msg cmdhandler.Message, req wsclient.WSMessage, cmdIndicator string, content string) (resp cmdhandler.Response, err error) {
	logger := logging.WithMessage(msg, h.deps.Logger())

	s, err := storage.GetSettings(h.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		_ = level.Error(logger).Log("message", "could not retrieve guild settings", "err", err)
	}

	if !IsAdminAuthorized(logger, msg, s.AdminRole, h.deps.BotSession()) {
		_ = level.Info(logger).Log("message", "non-admin trying to config")

		err = errUnauthorized
		return
	}

	_ = level.Debug(logger).Log("message", "admin trying to config")
	cmdContent := h.deps.ConfigHandler().CommandIndicator() + strings.TrimPrefix(content, cmdIndicator)
	_ = level.Info(logger).Log("message", "processing command", "cmdContent", fmt.Sprintf("%q", cmdContent), "rawCmd", fmt.Sprintf("%q", content))
	resp, err = h.deps.ConfigHandler().HandleMessage(cmdhandler.NewWithContents(msg, cmdContent))

	if err == nil {
		return
	}

	if err != errUnauthorized && err != parser.ErrUnknownCommand {
		return
	}

	if !IsAdminChannel(logger, msg, s.AdminChannel, h.deps.BotSession()) {
		err = errUnauthorized
		return
	}

	_ = level.Debug(logger).Log("message", "admin trying to admin")
	cmdContent = h.deps.AdminHandler().CommandIndicator() + strings.TrimPrefix(content, cmdIndicator)
	resp, err = h.deps.AdminHandler().HandleMessage(cmdhandler.NewWithContents(msg, cmdContent))

	return
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
		_ = level.Error(logger).Log("message", "error inflating message", "err", err)
		return
	}

	if m.MessageType() != etfapi.DefaultMessage {
		_ = level.Info(logger).Log("message", "message was not a default type")
		return
	}

	content := m.ContentString()
	if len(content) == 0 {
		_ = level.Info(logger).Log("message", "message contents empty")
		return
	}

	gid := h.channelGuild(m.ChannelID())
	cmdIndicator := h.guildCommandIndicator(gid)

	if !strings.HasPrefix(content, cmdIndicator) {
		_ = level.Info(logger).Log("message", "not a command")
		return
	}

	content = strings.TrimSpace(content)

	msg := cmdhandler.NewSimpleMessage(req.Ctx, m.AuthorID(), gid, m.ChannelID(), m.ID(), "")
	logger = logging.WithMessage(msg, h.deps.Logger())
	resp, err := h.attemptConfigAndAdminHandlers(msg, req, cmdIndicator, content)

	if err != nil && (err == errUnauthorized || err == parser.ErrUnknownCommand) {
		_ = level.Debug(logger).Log("message", "admin not successful; processing as real message")
		cmdContent := h.deps.CommandHandler().CommandIndicator() + strings.TrimPrefix(content, cmdIndicator)
		resp, err = h.deps.CommandHandler().HandleMessage(cmdhandler.NewWithContents(msg, cmdContent))
	}

	if err == ErrNoResponse || err == parser.ErrUnknownCommand {
		return
	}

	if err != nil {
		_ = level.Error(logger).Log("message", "error handling command", "contents", content, "err", err)
		resp.IncludeError(err)
	}

	if resp.HasErrors() {
		resp.SetColor(h.errorColor)
	} else {
		resp.SetColor(h.successColor)
	}

	_ = level.Info(logger).Log("message", "sending message", "resp", fmt.Sprintf("%+v", resp))

	sendTo := resp.Channel()
	if sendTo == 0 {
		sendTo = m.ChannelID()
	}

	splitResp := resp.Split()

	_ = level.Info(logger).Log("message", "sending message split", "split_count", len(splitResp))

	for _, res := range splitResp {
		err = h.deps.MessageRateLimiter().Wait(req.Ctx)
		if err != nil {
			_ = level.Error(logger).Log("message", "error waiting for ratelimiting", "err", err)
			return
		}

		sendResp, body, err := h.bot.SendMessage(req.Ctx, sendTo, res.ToMessage())
		if err != nil {
			var bodyStr string
			if body != nil {
				bodyStr = string(body)
			}
			_ = level.Error(logger).Log("message", "could not send message", "err", err, "resp_body", bodyStr, "status_code", sendResp.StatusCode)
			return
		}
	}

	_ = level.Info(logger).Log("message", "successfully sent message(s) to channel", "channel_id", sendTo.ToString(), "message_ct", len(splitResp))
}
