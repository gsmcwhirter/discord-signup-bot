package msghandler

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gsmcwhirter/discord-bot-lib/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/discordapi"
	"github.com/gsmcwhirter/discord-bot-lib/discordapi/etfapi"
	"github.com/gsmcwhirter/discord-bot-lib/discordapi/session"
	"github.com/gsmcwhirter/discord-bot-lib/logging"
	"github.com/gsmcwhirter/discord-bot-lib/snowflake"
	"github.com/gsmcwhirter/discord-bot-lib/wsclient"
	"github.com/gsmcwhirter/go-util/parser"
	"golang.org/x/time/rate"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

var errUnauthorized = errors.New("unauthorized")

type dependencies interface {
	Logger() log.Logger
	GuildAPI() storage.GuildAPI
	CommandHandler() *cmdhandler.CommandHandler
	ConfigHandler() *cmdhandler.CommandHandler
	AdminHandler() *cmdhandler.CommandHandler
	MessageRateLimiter() *rate.Limiter
	BotSession() *session.Session
}

// Handlers TODOC
type Handlers interface {
	ConnectToBot(discordapi.DiscordBot)
}

type handlers struct {
	bot                     discordapi.DiscordBot
	deps                    dependencies
	defaultCommandIndicator string
	successColor            int
	errorColor              int
}

// Options TODOC
type Options struct {
	DefaultCommandIndicator string
	SuccessColor            int
	ErrorColor              int
}

// NewHandlers TODOC
func NewHandlers(deps dependencies, opts Options) Handlers {
	h := handlers{
		deps: deps,
		defaultCommandIndicator: opts.DefaultCommandIndicator,
		successColor:            opts.SuccessColor,
		errorColor:              opts.ErrorColor,
	}

	return &h
}

func (h *handlers) ConnectToBot(bot discordapi.DiscordBot) {
	h.bot = bot

	bot.AddMessageHandler("MESSAGE_CREATE", h.handleMessage)
}

func (h *handlers) channelGuild(cid snowflake.Snowflake) (gid snowflake.Snowflake) {
	gid, _ = h.deps.BotSession().GuildOfChannel(cid)
	return
}

func (h *handlers) channelName(cid snowflake.Snowflake) (name string) {
	name, _ = h.deps.BotSession().ChannelName(cid)
	return
}

func (h *handlers) guildCommandIndicator(gid snowflake.Snowflake) string {
	if gid == 0 {
		return h.defaultCommandIndicator
	}

	s, err := storage.GetSettings(h.deps.GuildAPI(), gid.ToString())
	if err != nil {
		return h.defaultCommandIndicator
	}

	if s.ControlSequence == "" {
		return h.defaultCommandIndicator
	}

	return s.ControlSequence
}

func (h *handlers) guildAdminChannelID(gid snowflake.Snowflake) (snowflake.Snowflake, bool) {
	if gid == 0 {
		return 0, false
	}

	s, err := storage.GetSettings(h.deps.GuildAPI(), gid.ToString())
	if err != nil {
		return 0, false
	}

	g, err := h.deps.BotSession().Guild(gid)
	if err != nil {
		return 0, false
	}

	return g.ChannelWithName(s.AdminChannel)
}

func (h *handlers) attemptConfigAndAdminHandlers(msg *cmdhandler.SimpleMessage, req wsclient.WSMessage, cmdIndicator string, content string, m etfapi.Message, gid snowflake.Snowflake) (resp cmdhandler.Response, err error) {
	// TODO: check auth
	if !h.deps.BotSession().IsGuildAdmin(gid, m.AuthorID()) {
		_ = level.Debug(logging.WithContext(req.Ctx, h.deps.Logger())).Log("message", "non-admin trying to config", "author_id", m.AuthorID().ToString(), "guild_id", gid.ToString())

		err = errUnauthorized
		return
	}

	_ = level.Debug(logging.WithContext(req.Ctx, h.deps.Logger())).Log("message", "admin trying to config", "author_id", m.AuthorID().ToString(), "guild_id", gid.ToString())
	cmdContent := h.deps.ConfigHandler().CommandIndicator() + strings.TrimPrefix(content, cmdIndicator)
	resp, err = h.deps.ConfigHandler().HandleLine(cmdhandler.NewWithContents(msg, cmdContent))

	if err == nil {
		return
	}

	if err != errUnauthorized && err != parser.ErrUnknownCommand {
		return
	}

	adminChannelID, ok := h.guildAdminChannelID(gid)
	if !ok {
		err = errUnauthorized
		return
	}

	if m.ChannelID() != adminChannelID {
		err = errUnauthorized
		return
	}

	cmdContent = h.deps.AdminHandler().CommandIndicator() + strings.TrimPrefix(content, cmdIndicator)
	resp, err = h.deps.AdminHandler().HandleLine(cmdhandler.NewWithContents(msg, cmdContent))

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
		_ = level.Debug(logger).Log("message", "message was not a default type")
		return
	}

	content := m.ContentString()
	if len(content) == 0 {
		_ = level.Debug(logger).Log("message", "message contents empty")
		return
	}

	gid := h.channelGuild(m.ChannelID())
	cmdIndicator := h.guildCommandIndicator(gid)

	if !strings.HasPrefix(content, cmdIndicator) {
		_ = level.Debug(logger).Log("message", "not a command")
		return
	}

	msg := cmdhandler.NewSimpleMessage(m.AuthorID(), gid, m.ChannelID(), m.ID(), "")

	_ = level.Info(logger).Log("message", "attempting to handle command")
	resp, err := h.attemptConfigAndAdminHandlers(msg, req, cmdIndicator, content, m, gid)

	if err != nil && (err == errUnauthorized || err == parser.ErrUnknownCommand) {
		cmdContent := h.deps.CommandHandler().CommandIndicator() + strings.TrimPrefix(content, cmdIndicator)
		resp, err = h.deps.CommandHandler().HandleLine(cmdhandler.NewWithContents(msg, cmdContent))
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

	err = h.deps.MessageRateLimiter().Wait(req.Ctx)
	if err != nil {
		_ = level.Error(logger).Log("message", "error waiting for ratelimiting", "err", err)
		return
	}

	_ = level.Debug(logger).Log("message", "sending message", "marshaler", fmt.Sprintf("%+v", resp.ToMessage()), "resp", fmt.Sprintf("%+v", resp))

	sendTo := resp.Channel()
	if sendTo == 0 {
		sendTo = m.ChannelID()
	}

	sendResp, body, err := h.bot.SendMessage(req.Ctx, sendTo, resp.ToMessage())
	if err != nil {
		_ = level.Error(logger).Log("message", "could not send message", "err", err, "resp_body", string(body), "status_code", sendResp.StatusCode)
		return
	}

	_ = level.Info(logger).Log("message", "successfully sent message to channel", "channel_id", sendTo.ToString())

	return
}
