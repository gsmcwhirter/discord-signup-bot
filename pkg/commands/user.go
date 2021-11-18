package commands

import (
	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
)

// RootCommands holds the commands at the root level
type UserCommands struct {
	deps dependencies
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

// func (c *UserCommands) commandForGuild(ctx context.Context, gid snowflake.Snowflake) (cmd entity.ApplicationCommand, err error) {
// 	var eventNames []entity.ApplicationCommandOptionChoice

// 	tx, err := c.deps.TrialAPI().NewTransaction(ctx, gid.ToString(), false)
// 	if err != nil {
// 		return cmd, errors.Wrap(err, "could not open db transaction")
// 	}

// 	trials := tx.GetTrials(ctx)
// 	eventNames = make([]entity.ApplicationCommandOptionChoice, 0, len(trials))
// 	for _, t := range trials {
// 		name := t.GetName(ctx)
// 		eventNames = append(eventNames, entity.ApplicationCommandOptionChoice{
// 			Name:        name,
// 			ValueString: name,
// 		})
// 	}

// 	return entity.ApplicationCommand{
// 		Type:        entity.CmdTypeChatInput,
// 		Name:        "admin",
// 		Description: "Controls administrative functions of the bot",
// 		Options: []entity.ApplicationCommandOption{
// 			{
// 				Type:        entity.OptTypeSubCommand,
// 				Name:        "announce",
// 				Description: "Create an event announcement",
// 				Options: []entity.ApplicationCommandOption{
// 					{
// 						Type:        entity.OptTypeString,
// 						Name:        "event_name",
// 						Description: "Name of the event to announce",
// 						Choices:     eventNames,
// 						Required:    true,
// 					},
// 					{
// 						Type:        entity.OptTypeChannel,
// 						Name:        "announce_channel",
// 						Description: "The channel to announce into (omit for default)",
// 						Required:    false,
// 					},
// 				},
// 			},
// 			{
// 				Type:        entity.OptTypeSubCommand,
// 				Name:        "clear",
// 				Description: "Clear the signups from an event",
// 				Options: []entity.ApplicationCommandOption{
// 					{
// 						Type:        entity.OptTypeString,
// 						Name:        "event_name",
// 						Description: "Name of the event to clear",
// 						Choices:     eventNames,
// 						Required:    true,
// 					},
// 				},
// 			},
// 			{
// 				Type:        entity.OptTypeSubCommand,
// 				Name:        "close",
// 				Description: "Close an event",
// 				Options: []entity.ApplicationCommandOption{
// 					{
// 						Type:        entity.OptTypeString,
// 						Name:        "event_name",
// 						Description: "Name of the event to close",
// 						Choices:     eventNames,
// 						Required:    true,
// 					},
// 				},
// 			},
// 			// {
// 			// 	Type: entity.OptTypeSubCommand,
// 			// 	Name: "create",
// 			// 	Description: "Create a new event",
// 			// 	Options: []entity.ApplicationCommandOption{
// 			// 		{
// 			// 			Type:        entity.OptTypeString,
// 			// 			Name:        "event_name",
// 			// 			Description: "Name of the event to announce",
// 			// 			Choices:     eventNames,
// 			// 			Required:    true,
// 			// 		},
// 			// 	},
// 			// },
// 			{
// 				Type:        entity.OptTypeSubCommand,
// 				Name:        "debug",
// 				Description: "Debug an event",
// 				Options: []entity.ApplicationCommandOption{
// 					{
// 						Type:        entity.OptTypeString,
// 						Name:        "event_name",
// 						Description: "Name of the event to debug",
// 						Choices:     eventNames,
// 						Required:    true,
// 					},
// 				},
// 			},
// 			{
// 				Type:        entity.OptTypeSubCommand,
// 				Name:        "delete",
// 				Description: "Delete an event",
// 				Options: []entity.ApplicationCommandOption{
// 					{
// 						Type:        entity.OptTypeString,
// 						Name:        "event_name",
// 						Description: "Name of the event to delete",
// 						Choices:     eventNames,
// 						Required:    true,
// 					},
// 				},
// 			},
// 			// {
// 			// 	Type: entity.OptTypeSubCommand,
// 			// 	Name: "edit",
// 			// 	Description: "Clear the signups from an event",
// 			// 	Options: []entity.ApplicationCommandOption{
// 			// 		{
// 			// 			Type:        entity.OptTypeString,
// 			// 			Name:        "event_name",
// 			// 			Description: "Name of the event to announce",
// 			// 			Choices:     eventNames,
// 			// 			Required:    true,
// 			// 		},
// 			// 	},
// 			// },
// 			{
// 				Type:        entity.OptTypeSubCommand,
// 				Name:        "grouping",
// 				Description: "Clear the signups from an event",
// 				Options: []entity.ApplicationCommandOption{
// 					{
// 						Type:        entity.OptTypeString,
// 						Name:        "event_name",
// 						Description: "Name of the event to announce",
// 						Choices:     eventNames,
// 						Required:    true,
// 					},
// 					{
// 						Type:        entity.OptTypeString,
// 						Name:        "grouping_message",
// 						Description: "A message to include with the grouping notification",
// 						Required:    false,
// 					},
// 				},
// 			},
// 			{
// 				Type:        entity.OptTypeSubCommand,
// 				Name:        "list",
// 				Description: "List all events",
// 			},
// 			{
// 				Type:        entity.OptTypeSubCommand,
// 				Name:        "open",
// 				Description: "Open an event",
// 				Options: []entity.ApplicationCommandOption{
// 					{
// 						Type:        entity.OptTypeString,
// 						Name:        "event_name",
// 						Description: "Name of the event to open",
// 						Choices:     eventNames,
// 						Required:    true,
// 					},
// 				},
// 			},
// 			{
// 				Type:        entity.OptTypeSubCommand,
// 				Name:        "show",
// 				Description: "Show details for an event",
// 				Options: []entity.ApplicationCommandOption{
// 					{
// 						Type:        entity.OptTypeString,
// 						Name:        "event_name",
// 						Description: "Name of the event to show",
// 						Choices:     eventNames,
// 						Required:    true,
// 					},
// 				},
// 			},
// 			{
// 				Type:        entity.OptTypeSubCommand,
// 				Name:        "signup",
// 				Description: "Sign a user up for an event",
// 				Options: []entity.ApplicationCommandOption{
// 					{
// 						Type:        entity.OptTypeString,
// 						Name:        "event_name",
// 						Description: "Name of the event",
// 						Choices:     eventNames,
// 						Required:    true,
// 					},
// 					{
// 						Type:        entity.OptTypeUser,
// 						Name:        "user",
// 						Description: "User to sign up",
// 						Required:    true,
// 					},
// 					{
// 						Type:        entity.OptTypeString,
// 						Name:        "role",
// 						Description: "Role to sign up for",
// 						Required:    true,
// 					},
// 				},
// 			},
// 			{
// 				Type:        entity.OptTypeSubCommand,
// 				Name:        "withdraw",
// 				Description: "Withdraw a user from an event",
// 				Options: []entity.ApplicationCommandOption{
// 					{
// 						Type:        entity.OptTypeString,
// 						Name:        "event_name",
// 						Description: "Name of the event",
// 						Choices:     eventNames,
// 						Required:    true,
// 					},
// 					{
// 						Type:        entity.OptTypeUser,
// 						Name:        "user",
// 						Description: "User to sign up",
// 						Required:    true,
// 					},
// 				},
// 			},
// 		},
// 		DefaultPermission: true,
// 	}, nil
// }

func (c *UserCommands) GuildCommands(gid snowflake.Snowflake) ([]cmdhandler.InteractionCommandHandler, error) {
	// ctx := context.TODO()
	// command, err := c.commandForGuild(ctx, gid)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "could not get command for guild")
	// }

	// return []cmdhandler.InteractionCommandHandler{
	// 	&InteractionCommandHandler{
	// 		command: command,
	// 		handler: c,
	// 	},
	// }, nil

	return nil, nil
}
