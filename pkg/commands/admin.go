package commands

import (
	"context"
	"fmt"

	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/parser"

	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
)

// ErrGuildNotFound is the error returned when a guild is not known about
// in a BotSession
var ErrGuildNotFound = errors.New("guild not found")

type AdminCommands struct {
	preCommand string
	deps       adminDependencies
}

// AdminCommandHandler creates a new command handler for !admin commands
func AdminCommandHandler(deps adminDependencies, preCommand string) (*AdminCommands, *cmdhandler.CommandHandler, error) {
	p := parser.NewParser(parser.Options{
		CmdIndicator: "",
	})
	cc := &AdminCommands{
		preCommand: preCommand,
		deps:       deps,
	}

	ch, err := cmdhandler.NewCommandHandler(p, cmdhandler.Options{
		PreCommand:          preCommand,
		Placeholder:         "action",
		HelpOnEmptyCommands: true,
	})
	if err != nil {
		return cc, nil, err
	}

	cc.AttachToCommandHandler(ch)

	return cc, ch, nil
}

func (c *AdminCommands) HandleInteraction(ix *cmdhandler.Interaction) (cmdhandler.Response, []cmdhandler.Response, error) {
	if ix.Data == nil {
		return nil, nil, cmdhandler.ErrMalformedInteraction
	}

	var sc string
	var opts []entity.ApplicationCommandInteractionOption

	for i := range ix.Data.Options {
		if ix.Data.Options[i].Type != entity.OptTypeSubCommand {
			continue
		}

		sc = ix.Data.Options[i].Name
		opts = ix.Data.Options[i].Options
	}

	switch sc {
	case "announce":
		return c.announceInteraction(ix, opts)
	case "clear":
		return c.clearInteraction(ix, opts)
	case "close":
		return c.closeInteraction(ix, opts)
	case "create":
		return c.createInteraction(ix, opts)
	case "debug":
		return c.debugInteraction(ix, opts)
	case "delete":
		return c.deleteInteraction(ix, opts)
	case "edit":
		return c.editInteraction(ix, opts)
	case "grouping":
		return c.groupingInteraction(ix, opts)
	case "list":
		return c.listInteraction(ix, opts)
	case "open":
		return c.openInteraction(ix, opts)
	case "show":
		return c.showInteraction(ix, opts)
	case "signup":
		return c.signupInteraction(ix, opts)
	case "withdraw":
		return c.withdrawInteraction(ix, opts)
	default:
		return nil, nil, parser.ErrUnknownCommand
	}
}

var ErrNoFocus = errors.New("no focus")

func findFocusedOption(opts []entity.ApplicationCommandInteractionOption) (focused entity.ApplicationCommandInteractionOption, err error) {
	for i := range opts {
		if opts[i].Focused {
			return opts[i], nil
		}

		if opts[i].Type == entity.OptTypeSubCommand || opts[i].Type == entity.OptTypeSubCommandGroup {
			focused, err = findFocusedOption(opts[i].Options)
			if err != ErrNoFocus {
				return focused, err
			}
		}
	}

	return focused, ErrNoFocus
}

func (c *AdminCommands) Autocomplete(ix *cmdhandler.Interaction) ([]entity.ApplicationCommandOptionChoice, error) {
	if ix.Data == nil {
		return nil, cmdhandler.ErrMalformedInteraction
	}

	var sc string
	var opts []entity.ApplicationCommandInteractionOption
	var focused entity.ApplicationCommandInteractionOption
	var err error

	for i := range ix.Data.Options {
		if ix.Data.Options[i].Type != entity.OptTypeSubCommand {
			continue
		}

		sc = ix.Data.Options[i].Name
		opts = ix.Data.Options[i].Options

		focused, err = findFocusedOption(opts)
		if err != nil {
			return nil, errors.Wrap(err, "could not find focused option")
		}
	}

	scKind := fmt.Sprintf("%s:%s", sc, focused.Name)

	switch scKind {
	case "announce:event_name":
		return c.autocompleteOpenEvents(ix, opts, focused)
	case "clear:event_name":
		return c.autocompleteAllEvents(ix, opts, focused)
	case "close:event_name":
		return c.autocompleteOpenEvents(ix, opts, focused)
	case "debug:event_name":
		return c.autocompleteAllEvents(ix, opts, focused)
	case "delete:event_name":
		return c.autocompleteAllEvents(ix, opts, focused)
	case "edit:event_name":
		return c.autocompleteAllEvents(ix, opts, focused)
	case "grouping:event_name":
		return c.autocompleteOpenEvents(ix, opts, focused)
	case "open:event_name":
		return c.autocompleteClosedEvents(ix, opts, focused)
	case "show:event_name":
		return c.autocompleteAllEvents(ix, opts, focused)
	case "signup:event_name":
		return c.autocompleteOpenEvents(ix, opts, focused)
	case "signup:role":
		return c.autocompleteEventRoles(ix, opts, focused)
	case "withdraw:event_name":
		return c.autocompleteOpenEvents(ix, opts, focused)
	default:
		return nil, parser.ErrUnknownCommand
	}
}

func (c *AdminCommands) AttachToCommandHandler(ch *cmdhandler.CommandHandler) {
	ch.SetHandler("list", cmdhandler.NewMessageHandler(c.listHandler))
	ch.SetHandler("create", cmdhandler.NewMessageHandler(c.createHandler))
	ch.SetHandler("edit", cmdhandler.NewMessageHandler(c.editHandler))
	ch.SetHandler("open", cmdhandler.NewMessageHandler(c.openHandler))
	ch.SetHandler("close", cmdhandler.NewMessageHandler(c.closeHandler))
	ch.SetHandler("delete", cmdhandler.NewMessageHandler(c.deleteHandler))
	ch.SetHandler("announce", cmdhandler.NewMessageHandler(c.announceHandler))
	ch.SetHandler("grouping", cmdhandler.NewMessageHandler(c.groupingHandler))
	ch.SetHandler("signup", cmdhandler.NewMessageHandler(c.signupHandler))
	ch.SetHandler("su", cmdhandler.NewMessageHandler(c.signupHandler))
	ch.SetHandler("withdraw", cmdhandler.NewMessageHandler(c.withdrawHandler))
	ch.SetHandler("wd", cmdhandler.NewMessageHandler(c.withdrawHandler))
	ch.SetHandler("clear", cmdhandler.NewMessageHandler(c.clearHandler))
	ch.SetHandler("show", cmdhandler.NewMessageHandler(c.showHandler))
	ch.SetHandler("debug", cmdhandler.NewMessageHandler(c.debugHandler))
}

func (c *AdminCommands) GlobalCommands() []cmdhandler.InteractionCommandHandler {
	return nil
}

func (c *AdminCommands) commandForGuild(ctx context.Context, gid snowflake.Snowflake) (cmd entity.ApplicationCommand, err error) {
	return entity.ApplicationCommand{
		Type:        entity.CmdTypeChatInput,
		Name:        "admin",
		Description: "Controls administrative functions of the bot",
		Options: []entity.ApplicationCommandOption{
			{
				Type:        entity.OptTypeSubCommand,
				Name:        "announce",
				Description: "Create an event announcement",
				Options: []entity.ApplicationCommandOption{
					{
						Type:        entity.OptTypeString,
						Name:        "event_name",
						Description: "Name of the event to announce",
						// Choices:     eventNames,
						Required:     true,
						Autocomplete: true,
					},
					{
						Type:        entity.OptTypeString,
						Name:        "announce_message",
						Description: "A message to include at the top of the announcement",
						Required:    false,
					},
					{
						Type:         entity.OptTypeChannel,
						Name:         "announce_channel",
						Description:  "The channel to announce into (omit for default)",
						Required:     false,
						ChannelTypes: []entity.ChannelType{entity.ChannelGuildText},
					},
				},
			},
			{
				Type:        entity.OptTypeSubCommand,
				Name:        "clear",
				Description: "Clear the signups from an event",
				Options: []entity.ApplicationCommandOption{
					{
						Type:        entity.OptTypeString,
						Name:        "event_name",
						Description: "Name of the event to clear",
						// Choices:     eventNames,
						Required:     true,
						Autocomplete: true,
					},
				},
			},
			{
				Type:        entity.OptTypeSubCommand,
				Name:        "close",
				Description: "Close an event",
				Options: []entity.ApplicationCommandOption{
					{
						Type:        entity.OptTypeString,
						Name:        "event_name",
						Description: "Name of the event to close",
						// Choices:     eventNames,
						Required:     true,
						Autocomplete: true,
					},
				},
			},
			{
				Type:        entity.OptTypeSubCommand,
				Name:        "create",
				Description: "Create a new event",
				Options: []entity.ApplicationCommandOption{
					{
						Type:        entity.OptTypeString,
						Name:        "event_name",
						Description: "Name of the event to create",
						Required:    true,
					},
					{
						Type:        entity.OptTypeString,
						Name:        "roles",
						Description: "Roles for the event (comma-separated list of NAME:COUNT[:EMOJI])",
						Required:    true,
					},
					{
						Type:        entity.OptTypeString,
						Name:        "time",
						Description: "When the event will occur",
					},
					{
						Type:        entity.OptTypeString,
						Name:        "description",
						Description: "Description of the event",
					},
					{
						Type:         entity.OptTypeChannel,
						Name:         "announcechannel",
						Description:  "Channel to announce the event to (omit for default)",
						ChannelTypes: []entity.ChannelType{entity.ChannelGuildText},
					},
					{
						Type:         entity.OptTypeChannel,
						Name:         "signupchannel",
						Description:  "Channel to allow signups in (omit for default)",
						ChannelTypes: []entity.ChannelType{entity.ChannelGuildText},
					},
					{
						Type:        entity.OptTypeString,
						Name:        "announceto",
						Description: "Who to tag when the event is announced (omit for @everyone)",
					},
					{
						Type:        entity.OptTypeBoolean,
						Name:        "hidereactionsannounce",
						Description: "Hide the reactions when the event is announced",
					},
					{
						Type:        entity.OptTypeBoolean,
						Name:        "hidereactionsshow",
						Description: "Hide the reactions when the event is shown",
					},
					{
						Type:        entity.OptTypeString,
						Name:        "roleorder",
						Description: "Order to display the event roles (omit or set to empty for alphabetical)",
					},
				},
			},
			{
				Type:        entity.OptTypeSubCommand,
				Name:        "debug",
				Description: "Debug an event",
				Options: []entity.ApplicationCommandOption{
					{
						Type:        entity.OptTypeString,
						Name:        "event_name",
						Description: "Name of the event to debug",
						// Choices:     eventNames,
						Required:     true,
						Autocomplete: true,
					},
				},
			},
			{
				Type:        entity.OptTypeSubCommand,
				Name:        "delete",
				Description: "Delete an event",
				Options: []entity.ApplicationCommandOption{
					{
						Type:        entity.OptTypeString,
						Name:        "event_name",
						Description: "Name of the event to delete",
						// Choices:     eventNames,
						Required:     true,
						Autocomplete: true,
					},
				},
			},
			{
				Type:        entity.OptTypeSubCommand,
				Name:        "edit",
				Description: "Edit the details of an event",
				Options: []entity.ApplicationCommandOption{
					{
						Type:        entity.OptTypeString,
						Name:        "event_name",
						Description: "Name of the event to edit",
						// Choices:     eventNames,
						Required:     true,
						Autocomplete: true,
					},
					{
						Type:        entity.OptTypeString,
						Name:        "roles",
						Description: "Roles for the event (comma-separated list of NAME:COUNT[:EMOJI])",
					},
					{
						Type:        entity.OptTypeString,
						Name:        "time",
						Description: "When the event will occur",
					},
					{
						Type:        entity.OptTypeString,
						Name:        "description",
						Description: "Description of the event",
					},
					{
						Type:         entity.OptTypeChannel,
						Name:         "announcechannel",
						Description:  "Channel to announce the event to (omit for default)",
						ChannelTypes: []entity.ChannelType{entity.ChannelGuildText},
					},
					{
						Type:         entity.OptTypeChannel,
						Name:         "signupchannel",
						Description:  "Channel to allow signups in (omit for default)",
						ChannelTypes: []entity.ChannelType{entity.ChannelGuildText},
					},
					{
						Type:        entity.OptTypeString,
						Name:        "announceto",
						Description: "Who to tag when the event is announced (omit for @everyone)",
					},
					{
						Type:        entity.OptTypeBoolean,
						Name:        "hidereactionsannounce",
						Description: "Hide the reactions when the event is announced",
					},
					{
						Type:        entity.OptTypeBoolean,
						Name:        "hidereactionsshow",
						Description: "Hide the reactions when the event is shown",
					},
					{
						Type:        entity.OptTypeString,
						Name:        "roleorder",
						Description: "Order to display the event roles (omit or set to empty for alphabetical)",
					},
				},
			},
			{
				Type:        entity.OptTypeSubCommand,
				Name:        "grouping",
				Description: "Announce the start of an event",
				Options: []entity.ApplicationCommandOption{
					{
						Type:        entity.OptTypeString,
						Name:        "event_name",
						Description: "Name of the event to announce",
						// Choices:     eventNames,
						Required:     true,
						Autocomplete: true,
					},
					{
						Type:        entity.OptTypeString,
						Name:        "grouping_message",
						Description: "A message to include with the grouping notification",
						Required:    false,
					},
					{
						Type:         entity.OptTypeChannel,
						Name:         "grouping_channel",
						Description:  "The channel to announce into (omit or default)",
						Required:     false,
						ChannelTypes: []entity.ChannelType{entity.ChannelGuildText},
					},
				},
			},
			{
				Type:        entity.OptTypeSubCommand,
				Name:        "list",
				Description: "List all events",
			},
			{
				Type:        entity.OptTypeSubCommand,
				Name:        "open",
				Description: "Open an event",
				Options: []entity.ApplicationCommandOption{
					{
						Type:        entity.OptTypeString,
						Name:        "event_name",
						Description: "Name of the event to open",
						// Choices:     eventNames,
						Required:     true,
						Autocomplete: true,
					},
				},
			},
			{
				Type:        entity.OptTypeSubCommand,
				Name:        "show",
				Description: "Show details for an event",
				Options: []entity.ApplicationCommandOption{
					{
						Type:        entity.OptTypeString,
						Name:        "event_name",
						Description: "Name of the event to show",
						// Choices:     eventNames,
						Required:     true,
						Autocomplete: true,
					},
				},
			},
			{
				Type:        entity.OptTypeSubCommand,
				Name:        "signup",
				Description: "Sign a user up for an event",
				Options: []entity.ApplicationCommandOption{
					{
						Type:        entity.OptTypeString,
						Name:        "event_name",
						Description: "Name of the event",
						// Choices:     eventNames,
						Required:     true,
						Autocomplete: true,
					},
					{
						Type:        entity.OptTypeUser,
						Name:        "user",
						Description: "User to sign up",
						Required:    true,
					},
					{
						Type:         entity.OptTypeString,
						Name:         "role",
						Description:  "Role to sign up for",
						Required:     true,
						Autocomplete: true,
					},
				},
			},
			{
				Type:        entity.OptTypeSubCommand,
				Name:        "withdraw",
				Description: "Withdraw a user from an event",
				Options: []entity.ApplicationCommandOption{
					{
						Type:        entity.OptTypeString,
						Name:        "event_name",
						Description: "Name of the event",
						// Choices:     eventNames,
						Required:     true,
						Autocomplete: true,
					},
					{
						Type:        entity.OptTypeUser,
						Name:        "user",
						Description: "User to withdraw",
						Required:    true,
					},
				},
			},
		},
		DefaultPermission: true,
	}, nil
}

func (c *AdminCommands) GuildCommands(gid snowflake.Snowflake) ([]cmdhandler.InteractionCommandHandler, error) {
	ctx := context.TODO()
	command, err := c.commandForGuild(ctx, gid)
	if err != nil {
		return nil, errors.Wrap(err, "could not get command for guild")
	}

	return []cmdhandler.InteractionCommandHandler{
		&InteractionCommandHandler{
			command:      command,
			handler:      c,
			autocomplete: c,
		},
	}, nil
}
