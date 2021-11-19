package commands

import (
	"fmt"

	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
	"github.com/gsmcwhirter/discord-bot-lib/v23/logging"
	"github.com/gsmcwhirter/go-util/v8/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

func (c *ConfigCommands) aboutInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "configCommands.aboutInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling config interaction", "command", "about")

	gsettings, err := storage.GetSettings(ctx, c.deps.GuildAPI(), ix.GuildID())
	if err != nil {
		return r, nil, err
	}

	okColor, err := colorToInt(gsettings.MessageColor)
	if err != nil {
		return r, nil, err
	}

	r.SetColor(okColor)

	if !isAdminChannel(logger, ix, gsettings.AdminChannel, c.deps.BotSession()) {
		level.Info(logger).Message("command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, nil, msghandler.ErrUnauthorized
	}

	r.Description = fmt.Sprintf("Version %s\nDiscord: https://discord.gg/BgkvvbN\nWebsite: https://www.evogames.org/bots/eso-signup-bot/\n", c.versionStr)

	return r, nil, nil
}
