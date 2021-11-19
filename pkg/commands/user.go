package commands

import (
	"fmt"

	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/parser"
)

// RootCommands holds the commands at the root level
type UserCommands struct {
	deps dependencies
}

func (c *UserCommands) HandleInteraction(ix *cmdhandler.Interaction) (cmdhandler.Response, []cmdhandler.Response, error) {
	if ix.Data == nil {
		return nil, nil, cmdhandler.ErrMalformedInteraction
	}

	sc := ix.Data.Name
	opts := ix.Data.Options

	_ = opts

	switch sc {
	case "list":
		return c.listInteraction(ix, opts)
	case "myevents":
		return c.myEventsInteraction(ix, opts)
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

func (c *UserCommands) Autocomplete(ix *cmdhandler.Interaction) ([]entity.ApplicationCommandOptionChoice, error) {
	if ix.Data == nil {
		return nil, cmdhandler.ErrMalformedInteraction
	}

	sc := ix.Data.Name
	opts := ix.Data.Options
	focused, err := findFocusedOption(opts)
	if err != nil {
		return nil, errors.Wrap(err, "could not find focused option")
	}

	scKind := fmt.Sprintf("%s:%s", sc, focused.Name)

	switch scKind {
	case "show:event_name":
		return c.autocompleteOpenEvents(ix, opts, focused)
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

func (c *UserCommands) AttachToCommandHandler(ch *cmdhandler.CommandHandler) {
	ch.SetHandler("list", cmdhandler.NewMessageHandler(c.listHandler))
	ch.SetHandler("myevents", cmdhandler.NewMessageHandler(c.myEventsHandler))
	ch.SetHandler("show", cmdhandler.NewMessageHandler(c.showHandler))
	ch.SetHandler("signup", cmdhandler.NewMessageHandler(c.signupHandler))
	ch.SetHandler("su", cmdhandler.NewMessageHandler(c.signupHandler))
	ch.SetHandler("withdraw", cmdhandler.NewMessageHandler(c.withdrawHandler))
	ch.SetHandler("wd", cmdhandler.NewMessageHandler(c.withdrawHandler))
}

func (c *UserCommands) GlobalCommands() []cmdhandler.InteractionCommandHandler {
	return nil
}

func (c *UserCommands) GuildCommands(gid snowflake.Snowflake) ([]cmdhandler.InteractionCommandHandler, error) {
	return []cmdhandler.InteractionCommandHandler{
		&InteractionCommandHandler{
			command: entity.ApplicationCommand{
				Type:              entity.CmdTypeChatInput,
				Name:              "list",
				Description:       "List all open events",
				DefaultPermission: true,
			},
			handler:      c,
			autocomplete: c,
		},
		&InteractionCommandHandler{
			command: entity.ApplicationCommand{
				Type:              entity.CmdTypeChatInput,
				Name:              "myevents",
				Description:       "List my currently signed-up-for events",
				DefaultPermission: true,
			},
			handler:      c,
			autocomplete: c,
		},
		&InteractionCommandHandler{
			command: entity.ApplicationCommand{
				Type:        entity.CmdTypeChatInput,
				Name:        "show",
				Description: "Show details of the requested event",
				Options: []entity.ApplicationCommandOption{
					{
						Type:         entity.OptTypeString,
						Name:         "event_name",
						Description:  "The name of the event to show",
						Required:     true,
						Autocomplete: true,
					},
				},
				DefaultPermission: true,
			},
			handler:      c,
			autocomplete: c,
		},
		&InteractionCommandHandler{
			command: entity.ApplicationCommand{
				Type:        entity.CmdTypeChatInput,
				Name:        "signup",
				Description: "Sign up for an event",
				Options: []entity.ApplicationCommandOption{
					{
						Type:         entity.OptTypeString,
						Name:         "event_name",
						Description:  "The name of the event to sign up for",
						Required:     true,
						Autocomplete: true,
					},
					{
						Type:         entity.OptTypeString,
						Name:         "role",
						Description:  "The role to sign up for",
						Required:     true,
						Autocomplete: true,
					},
				},
				DefaultPermission: true,
			},
			handler:      c,
			autocomplete: c,
		},
		&InteractionCommandHandler{
			command: entity.ApplicationCommand{
				Type:        entity.CmdTypeChatInput,
				Name:        "withdraw",
				Description: "Withdraw from an event",
				Options: []entity.ApplicationCommandOption{
					{
						Type:         entity.OptTypeString,
						Name:         "event_name",
						Description:  "The name of the event to withdraw from",
						Required:     true,
						Autocomplete: true,
					},
				},
				DefaultPermission: true,
			},
			handler:      c,
			autocomplete: c,
		},
	}, nil
}
