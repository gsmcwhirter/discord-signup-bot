package commands

import (
	"github.com/gsmcwhirter/go-util/v8/parser"

	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
)

type ConfigCommands struct {
	preCommand string
	versionStr string
	deps       configDependencies
}

func (c *ConfigCommands) HandleInteraction(ix *cmdhandler.Interaction) (cmdhandler.Response, []cmdhandler.Response, error) {
	if ix.Data == nil {
		return nil, nil, cmdhandler.ErrMalformedInteraction
	}

	var sc string
	var opts []entity.ApplicationCommandInteractionOption

	for i := range ix.Data.Options {
		if ix.Data.Options[i].Type != entity.OptTypeSubCommand && ix.Data.Options[i].Type != entity.OptTypeSubCommandGroup {
			continue
		}

		sc = ix.Data.Options[i].Name
		opts = ix.Data.Options[i].Options
		break
	}

	switch sc {
	case "about":
		return c.aboutInteraction(ix, opts)
	case "debug":
		return c.debugInteraction(ix, opts)
	case "factory-reset":
		return c.resetInteraction(ix, opts)
	case "get":
		return c.getInteraction(ix, opts)
	case "list":
		return c.listInteraction(ix, opts)
	case "reset":
		return c.resetInteraction(ix, opts)
	case "set":
		return c.setInteraction(ix, opts)
	case "stats":
		return c.statsInteraction(ix, opts)
	case "adminrole":
		var arsc string
		var aropts []entity.ApplicationCommandInteractionOption

		for i := range opts {
			if opts[i].Type != entity.OptTypeSubCommand {
				continue
			}

			arsc = opts[i].Name
			aropts = opts[i].Options
			break
		}

		switch arsc {
		case "add":
			return c.adminroleAddInteraction(ix, aropts)
		case "clear":
			return c.adminroleClearInteraction(ix, aropts)
		case "refresh":
			return c.adminroleRefreshInteraction(ix, aropts)
		case "remove":
			return c.adminroleRemoveInteraction(ix, aropts)
		case "list":
			return c.adminroleListInteraction(ix, aropts)
		default:
			return nil, nil, parser.ErrUnknownCommand
		}
	default:
		return nil, nil, parser.ErrUnknownCommand
	}
}

func (c *ConfigCommands) HandleAutocomplete(ix *cmdhandler.Interaction) ([]entity.ApplicationCommandOptionChoice, error) {
	return nil, parser.ErrUnknownCommand
}

type stat struct {
	trials int
	open   int
	closed int
}

// ConfigCommandHandler creates a new command handler for !config-su commands
func ConfigCommandHandler(deps configDependencies, versionStr, preCommand string) (*ConfigCommands, *cmdhandler.CommandHandler, error) {
	p := parser.NewParser(parser.Options{
		CmdIndicator: "",
	})
	cc := &ConfigCommands{
		preCommand: preCommand,
		deps:       deps,
		versionStr: versionStr,
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

	return cc, ch, err
}

func (c *ConfigCommands) AttachToCommandHandler(ch *cmdhandler.CommandHandler) {
	ch.SetHandler("list", cmdhandler.NewMessageHandler(c.listHandler))
	ch.SetHandler("get", cmdhandler.NewMessageHandler(c.getHandler))
	ch.SetHandler("set", cmdhandler.NewMessageHandler(c.setHandler))
	ch.SetHandler("reset", cmdhandler.NewMessageHandler(c.resetHandler))
	ch.SetHandler("version", cmdhandler.NewMessageHandler(c.versionHandler))
	ch.SetHandler("website", cmdhandler.NewMessageHandler(c.websiteHandler))
	ch.SetHandler("discord", cmdhandler.NewMessageHandler(c.discordHandler))
	ch.SetHandler("stats", cmdhandler.NewMessageHandler(c.statsHandler))
}

func (c *ConfigCommands) GlobalCommands() []cmdhandler.InteractionCommandHandler {
	return nil
}

func (c *ConfigCommands) GuildCommands(gid snowflake.Snowflake) ([]cmdhandler.InteractionCommandHandler, error) {
	settings := []string{
		"controlsequence", // DEPRECATED
		"announcechannel",
		"adminchannel",
		"signupchannel",
		"announceto",
		"showaftersignup",
		"showafterwithdraw",
		"hidereactionsannounce",
		"hidereactionsshow",
		"adminrole",
		"messagecolor",
		"errorcolor",
	}

	settingOptions := make([]entity.ApplicationCommandOptionChoice, 0, len(settings))
	for _, s := range settings {
		settingOptions = append(settingOptions, entity.ApplicationCommandOptionChoice{
			Name:        s,
			ValueString: s,
			Type:        entity.OptTypeString,
		})
	}

	return []cmdhandler.InteractionCommandHandler{
		&InteractionCommandHandler{
			command: entity.ApplicationCommand{
				Type:        entity.CmdTypeChatInput,
				Name:        "config",
				Description: "Configure bot settings and behavior",
				Options: []entity.ApplicationCommandOption{
					{
						Type:        entity.OptTypeSubCommand,
						Name:        "debug",
						Description: "Debug settings",
					},
					{
						Type:        entity.OptTypeSubCommand,
						Name:        "about",
						Description: "Show bot information",
					},
					{
						Type:        entity.OptTypeSubCommand,
						Name:        "stats",
						Description: "Show statistics",
					},
					{
						Type:        entity.OptTypeSubCommand,
						Name:        "factory-reset",
						Description: "Reset all settings to their default values",
					},
					{
						Type:        entity.OptTypeSubCommand,
						Name:        "list",
						Description: "List all settings",
					},
					{
						Type:        entity.OptTypeSubCommand,
						Name:        "get",
						Description: "Get the value of a setting",
						Options: []entity.ApplicationCommandOption{
							{
								Type:        entity.OptTypeString,
								Name:        "setting_name",
								Description: "Name of the setting",
								Choices:     settingOptions,
								Required:    true,
							},
						},
					},
					{
						Type:        entity.OptTypeSubCommand,
						Name:        "set",
						Description: "Set the value of one or more settings",
						Options: []entity.ApplicationCommandOption{
							{
								Type:        entity.OptTypeString,
								Name:        "controlsequence",
								Description: "Control sequence for the original bot functionality (DEPRECATED)",
							},
							{
								Type:         entity.OptTypeChannel,
								Name:         "announcechannel",
								Description:  "Channel to post event announcements to, by default",
								ChannelTypes: []entity.ChannelType{entity.ChannelGuildText},
							},
							{
								Type:         entity.OptTypeChannel,
								Name:         "adminchannel",
								Description:  "Channel to listen for admin messages in",
								ChannelTypes: []entity.ChannelType{entity.ChannelGuildText},
							},
							{
								Type:         entity.OptTypeChannel,
								Name:         "signupchannel",
								Description:  "Channel to listed for signup messages in, by default",
								ChannelTypes: []entity.ChannelType{entity.ChannelGuildText},
							},
							{
								Type:        entity.OptTypeString,
								Name:        "announceto",
								Description: "String to include at the top of announcements (may include @here, etc)",
							},
							{
								Type:        entity.OptTypeBoolean,
								Name:        "showaftersignup",
								Description: "Whether or not to show event details after every signup",
							},
							{
								Type:        entity.OptTypeBoolean,
								Name:        "showafterwithdraw",
								Description: "Whether or not to show event details after every withdraw",
							},
							{
								Type:        entity.OptTypeBoolean,
								Name:        "hidereactionsannounce",
								Description: "Whether or not to hide reactions on event announcement messages",
							},
							{
								Type:        entity.OptTypeBoolean,
								Name:        "hidereactionsshow",
								Description: "Whether or not to hide reactions on event details messages",
							},
							{
								Type:        entity.OptTypeString,
								Name:        "messagecolor",
								Description: "Color code for standard messages",
							},
							{
								Type:        entity.OptTypeString,
								Name:        "errorcolor",
								Description: "Color code for error messages",
							},
						},
					},
					{
						Type:        entity.OptTypeSubCommandGroup,
						Name:        "adminrole",
						Description: "Manage admin roles",
						Options: []entity.ApplicationCommandOption{
							{
								Type:        entity.OptTypeSubCommand,
								Name:        "clear",
								Description: "Remove all admin roles",
							},
							{
								Type:        entity.OptTypeSubCommand,
								Name:        "list",
								Description: "List all admin roles",
							},
							{
								Type:        entity.OptTypeSubCommand,
								Name:        "refresh",
								Description: "Refresh permissions for all admin roles",
							},
							{
								Type:        entity.OptTypeSubCommand,
								Name:        "add",
								Description: "Add an admin role",
								Options: []entity.ApplicationCommandOption{
									{
										Type:        entity.OptTypeRole,
										Name:        "role",
										Description: "The role to add as an administrator",
										Required:    true,
									},
								},
							},
							{
								Type:        entity.OptTypeSubCommand,
								Name:        "remove",
								Description: "Remove an admin role",
								Options: []entity.ApplicationCommandOption{
									{
										Type:        entity.OptTypeRole,
										Name:        "role",
										Description: "The role to remove as an administrator",
										Required:    true,
									},
								},
							},
						},
					},
				},
				DefaultPermission: false,
			},
			handler: c,
		},
	}, nil
}

// ConfigDebugCommandHandler creates a new command handler for !config-su-debug commands
func ConfigDebugCommandHandler(deps configDependencies, preCommand string) (*cmdhandler.CommandHandler, error) {
	p := parser.NewParser(parser.Options{
		CmdIndicator: "",
	})
	cc := &ConfigCommands{
		preCommand: preCommand,
		deps:       deps,
	}

	ch, err := cmdhandler.NewCommandHandler(p, cmdhandler.Options{
		PreCommand:          preCommand,
		Placeholder:         "action",
		HelpOnEmptyCommands: true,
	})
	if err != nil {
		return nil, err
	}

	ch.SetHandler("factory-reset", cmdhandler.NewMessageHandler(cc.resetHandler))
	ch.SetHandler("info", cmdhandler.NewMessageHandler(cc.debugHandler))

	return ch, err
}
