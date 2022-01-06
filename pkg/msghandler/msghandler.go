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

	"github.com/gsmcwhirter/discord-bot-lib/v23/bot"
	"github.com/gsmcwhirter/discord-bot-lib/v23/bot/session"
	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/etfapi"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/jsonapi"
	"github.com/gsmcwhirter/discord-bot-lib/v23/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v23/request"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
	"github.com/gsmcwhirter/discord-bot-lib/v23/wsapi"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/permissions"
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
	InteractionDispatcher() *cmdhandler.InteractionDispatcher
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
	InteractionSendAllowed() bool
	PermissionsManager() *permissions.Manager
	GuildCommandGenerator() func(snowflake.Snowflake) ([]cmdhandler.InteractionCommandHandler, error)
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

	interactionGuildAllowlist map[snowflake.Snowflake]bool
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

		interactionGuildAllowlist: map[snowflake.Snowflake]bool{
			snowflake.Snowflake(468646871133454357): true,
			snowflake.Snowflake(869634209550041128): true, // Daelinia
			snowflake.Snowflake(685674036214235175): true, // Daelinia
			snowflake.Snowflake(804137119085101137): true, // kingshart
			snowflake.Snowflake(649000389882150922): true, // OtterB
		},
	}

	return &h
}

func (h *handlers) ConnectToBot(b *bot.DiscordBot) {
	h.bot = b

	b.Dispatcher().AddHandler("MESSAGE_CREATE", h.handleMessage)
	b.Dispatcher().AddHandler("MESSAGE_REACTION_ADD", h.handleReactionAdd)
	b.Dispatcher().AddHandler("MESSAGE_REACTION_REMOVE", h.handleReactionRemove)
	b.Dispatcher().AddHandler("INTERACTION_CREATE", h.handleInteraction)

	b.Dispatcher().AddHandler("GUILD_CREATE", h.handleGuildCreate)
	b.Dispatcher().AddHandler("GUILD_UPDATE", h.handleGuildUpdate)
	b.Dispatcher().AddHandler("GUILD_ROLE_CREATE", h.handleGuildRoleCreate)
	b.Dispatcher().AddHandler("GUILD_ROLE_UPDATE", h.handleGuildRoleUpdate)
	b.Dispatcher().AddHandler("GUILD_ROLE_DELETE", h.handleGuildRoleDelete)
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

func (h *handlers) handleResponse(ctx context.Context, logger Logger, resp cmdhandler.Response, cid, gid snowflake.Snowflake, content string, allowSend bool, err error) {
	if err == ErrNoResponse || err == parser.ErrUnknownCommand {
		return
	}

	if err != nil {
		level.Error(logger).Err("error handling command", err, "contents", content)
		resp.IncludeError(err)
	}

	if resp.HasErrors() && resp.GetColor() == 0 {
		resp.SetColor(h.errorColor)
	}

	if !resp.HasErrors() && resp.GetColor() == 0 {
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
		if !allowSend {
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

func (h *handlers) handleInteractionResponse(ctx context.Context, logger Logger, resp cmdhandler.Response, ix *cmdhandler.Interaction, extras []cmdhandler.Response, err error) {
	if err == ErrNoResponse || err == parser.ErrUnknownCommand {
		return
	}

	if resp == nil {
		resp = &cmdhandler.SimpleResponse{}
	}

	if err != nil {
		level.Error(logger).Err("error handling interaction", err)
		resp.IncludeError(err)
		resp.SetEphemeral(true)
	}

	if resp.HasErrors() && resp.GetColor() == 0 {
		resp.SetColor(h.errorColor)
	}

	if !resp.HasErrors() && resp.GetColor() == 0 {
		resp.SetColor(h.successColor)
	}

	level.Info(logger).Message("sending interaction response", "resp", fmt.Sprintf("%+v", resp))

	splitResp := resp.Split()

	level.Info(logger).Message("sending interaction response split", "split_count", len(splitResp))

	for _, res := range splitResp {
		if !h.deps.InteractionSendAllowed() {
			level.Info(logger).Message("interaction response send disabled", "message_to_send", fmt.Sprintf("%#v", res.ToMessage()), "reactions", res.MessageReactions())
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

		err = h.bot.API().SendInteractionMessage(ctx, ix.IDSnowflake, ix.Token, res.ToMessage())
		if err != nil {
			level.Error(logger).Err("could not send interaction response", err)
			return
		}
	}

	reacts := resp.MessageReactions()
	if len(reacts) > 0 {
		sentMsg, err := h.bot.API().GetInteractionResponse(ctx, ix.ApplicationIDSnowflake, ix.Token)
		if err != nil {
			level.Error(logger).Err("could not retrieve message information prior to adding reactions", err)
			return
		}

		for _, reaction := range reacts {
			err = h.deps.ReactionsRateLimiter().Wait(ctx)
			if err != nil {
				level.Error(logger).Err("error waiting for ratelimiting for reaction", err)
				return
			}

			if ar, ok := h.deps.StatsHub().Get("reaction_sent"); ok {
				ar.Incr(1)
			}

			resp, err := h.bot.API().CreateReaction(ctx, ix.ChannelIDSnowflake, sentMsg.IDSnowflake, reaction)
			if err != nil {
				status := 0
				if resp != nil {
					status = resp.StatusCode
				}

				level.Error(logger).Err("could not add reaction", err, "status_code", status)
			}
		}
	}

	level.Info(logger).Message("successfully sent message(s) to interaction response", "interaction_id", ix.IDSnowflake, "message_ct", len(splitResp))

	for _, resp := range extras {
		cid := resp.Channel()
		if cid == 0 {
			cid = ix.ChannelID()
		}
		h.handleResponse(ctx, logger, resp, cid, ix.GuildID(), "", h.deps.InteractionSendAllowed(), nil)
	}
}

func (h *handlers) handleAutocompleteResponse(ctx context.Context, logger Logger, choices []entity.ApplicationCommandOptionChoice, ix *cmdhandler.Interaction, err error) {
	if err == ErrNoResponse || err == parser.ErrUnknownCommand {
		return
	}

	if err != nil {
		level.Error(logger).Err("error handling interaction", err)
	}

	level.Info(logger).Message("sending autocomplete response", "choices", fmt.Sprintf("%+v", choices))

	resp := jsonapi.InteractionAutocompleteResponse{
		Choices: choices,
	}

	if !h.deps.InteractionSendAllowed() {
		level.Info(logger).Message("interaction response send disabled", "message_to_send", fmt.Sprintf("%#v", resp))
		return
	}

	err = h.deps.MessageRateLimiter().Wait(ctx)
	if err != nil {
		level.Error(logger).Err("error waiting for ratelimiting", err)
		return
	}

	if ar, ok := h.deps.StatsHub().Get("autocompletes_sent"); ok {
		ar.Incr(1)
	}

	err = h.bot.API().SendInteractionAutocomplete(ctx, ix.IDSnowflake, ix.Token, resp)
	if err != nil {
		level.Error(logger).Err("could not send interaction response", err)
		return
	}

	level.Info(logger).Message("successfully sent choices to autocomplete response", "interaction_id", ix.IDSnowflake, "choices_ct", len(choices))
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

	m, err := entity.MessageFromElementMap(p.Contents())
	if err != nil {
		level.Error(logger).Err("error inflating message", err)
		return 0
	}

	if m.MessageType() != entity.DefaultMessage {
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

	h.handleResponse(req.Ctx, logger, resp, m.ChannelID(), gid, content, h.deps.SendAllowed(), err)

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

	r, err := entity.ReactionFromElementMap(p.Contents())
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

	h.handleResponse(req.Ctx, logger, resp, r.ChannelID(), gid, r.Emoji(), h.deps.SendAllowed(), err)

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

	r, err := entity.ReactionFromElementMap(p.Contents())
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

	h.handleResponse(req.Ctx, logger, resp, r.ChannelID(), gid, r.Emoji(), h.deps.SendAllowed(), err)

	return gid
}

func (h *handlers) handleInteraction(p bot.Payload, req wsapi.WSMessage, respChan chan<- wsapi.WSMessage) snowflake.Snowflake {
	ctx, span := h.deps.Census().StartSpan(req.Ctx, "handlers.handleInteraction")
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

	ix, err := entity.InteractionFromElementMap(p.Contents())
	if err != nil {
		level.Error(logger).Err("error inflating interaction", err)
		return 0
	}

	gid := ix.GuildIDSnowflake
	req.Ctx = request.WithGuildID(req.Ctx, gid)

	if ar, ok := h.deps.StatsHub().Get("interactions"); ok {
		ar.Incr(1)
	}

	msg := &cmdhandler.Interaction{
		Ctx:         req.Ctx,
		Interaction: ix,
	}

	switch ix.Type {
	case entity.InteractionApplicationCommand:
		return h.handleInteractionCommand(msg)
	case entity.InteractionAutocomplete:
		return h.handleInteractionAutocomplete(msg)
	default:
		level.Info(logger).Message("interaction was not a known type")
		return 0
	}
}

func (h *handlers) handleInteractionCommand(ix *cmdhandler.Interaction) snowflake.Snowflake {
	ctx, span := h.deps.Census().StartSpan(ix.Context(), "handlers.handleInteractionCommand")
	defer span.End()

	select {
	case <-ctx.Done():
		return 0
	default:
	}

	if ar, ok := h.deps.StatsHub().Get("interactions_command"); ok {
		ar.Incr(1)
	}

	logger := logging.WithMessage(ix, h.deps.Logger())

	// TODO: config and admin permissions
	level.Debug(logger).Message("admin not checked; processing as real message")
	resp, extras, err := h.deps.InteractionDispatcher().Dispatch(ix)

	h.handleInteractionResponse(ctx, logger, resp, ix, extras, err)

	return ix.GuildID()
}

func (h *handlers) handleInteractionAutocomplete(ix *cmdhandler.Interaction) snowflake.Snowflake {
	ctx, span := h.deps.Census().StartSpan(ix.Context(), "handlers.handleInteractionAutocomplete")
	defer span.End()

	select {
	case <-ctx.Done():
		return 0
	default:
	}

	if ar, ok := h.deps.StatsHub().Get("interactions_autocomplete"); ok {
		ar.Incr(1)
	}

	logger := logging.WithMessage(ix, h.deps.Logger())

	// TODO: do we need permissions here?
	choices, err := h.deps.InteractionDispatcher().Autocomplete(ix)

	h.handleAutocompleteResponse(ctx, logger, choices, ix, err)

	return ix.GuildID()
}

func (h *handlers) upsertGuildCommandsAndPermissions(ctx context.Context, gid snowflake.Snowflake) error {
	ctx, span := h.deps.Census().StartSpan(ctx, "handlers.upsertGuildCommandsAndPermissions", "gid", gid.ToString())
	defer span.End()

	if !h.interactionGuildAllowlist[gid] {
		return nil
	}

	gcmds, err := h.deps.GuildCommandGenerator()(gid)
	if err != nil {
		return errors.Wrap(err, "could not generate guild commands")
	}

	if err := h.deps.InteractionDispatcher().LearnGuildCommands(gid, gcmds); err != nil {
		return errors.Wrap(err, "could not LearnGuildCommands")
	}

	learned, err := h.bot.RegisterGuildCommands(ctx, gid, h.deps.InteractionDispatcher().GuildCommands()[gid])
	if err != nil {
		return errors.Wrap(err, "could not RegisterGuildCommands")
	}

	h.deps.PermissionsManager().SetGuildCommands(gid, learned)

	return errors.Wrap(h.upsertGuildCommandPermissions(ctx, gid), "could not upsertGuildCommandPermissions")
}

func (h *handlers) upsertGuildCommandPermissions(ctx context.Context, gid snowflake.Snowflake) error {
	ctx, span := h.deps.Census().StartSpan(ctx, "handlers.upsertGuildCommandPermissions", "gid", gid.ToString())
	defer span.End()

	if !h.interactionGuildAllowlist[gid] {
		return nil
	}

	return errors.Wrap(h.deps.PermissionsManager().RefreshPermissions(ctx, h.bot.Config().ClientID, gid), "could not RefreshPermissions")
}

var ErrGuildNotFound = errors.New("guild not found")

func (h *handlers) handleGuildCreate(p bot.Payload, req wsapi.WSMessage, respChan chan<- wsapi.WSMessage) snowflake.Snowflake {
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

	e, ok := p.Contents()["id"]
	if !ok {
		level.Error(logger).Err("could not handleGuildCreate", errors.Wrap(ErrGuildNotFound, "could not find guild id map element"))
		return 0
	}

	gid, err := etfapi.SnowflakeFromElement(e)
	if err != nil {
		level.Error(logger).Err("could not handleGuildCreate", errors.Wrap(err, "could not convert id to snowflake"))
		return 0
	}

	if err := h.upsertGuildCommandsAndPermissions(req.Ctx, gid); err != nil {
		level.Error(logger).Err("could not handleGuildCreate", errors.Wrap(err, "could not upsertGuildCommandsAndPermissions"))
	}

	return gid
}

func (h *handlers) handleGuildUpdate(p bot.Payload, req wsapi.WSMessage, respChan chan<- wsapi.WSMessage) snowflake.Snowflake {
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

	e, ok := p.Contents()["id"]
	if !ok {
		level.Error(logger).Err("could not handleGuildUpdate", errors.Wrap(ErrGuildNotFound, "could not find guild id map element"))
		return 0
	}

	gid, err := etfapi.SnowflakeFromElement(e)
	if err != nil {
		level.Error(logger).Err("could not handleGuildUpdate", errors.Wrap(err, "could not convert id to snowflake"))
		return 0
	}

	if err := h.upsertGuildCommandsAndPermissions(req.Ctx, gid); err != nil {
		level.Error(logger).Err("could not handleGuildUpdate", errors.Wrap(err, "could not upsertGuildCommandsAndPermissions"))
	}

	return gid
}

func (h *handlers) handleGuildRoleCreate(p bot.Payload, req wsapi.WSMessage, respChan chan<- wsapi.WSMessage) snowflake.Snowflake {
	return 0
}

func (h *handlers) handleGuildRoleUpdate(p bot.Payload, req wsapi.WSMessage, respChan chan<- wsapi.WSMessage) snowflake.Snowflake {
	return 0
}

func (h *handlers) handleGuildRoleDelete(p bot.Payload, req wsapi.WSMessage, respChan chan<- wsapi.WSMessage) snowflake.Snowflake {
	return 0
}
