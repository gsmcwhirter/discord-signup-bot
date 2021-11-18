package commands

import (
	"strings"

	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
	"github.com/gsmcwhirter/go-util/v8/deferutil"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

func (c *AdminCommands) autocompleteOpenEvents(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption, focused entity.ApplicationCommandInteractionOption) ([]entity.ApplicationCommandOptionChoice, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "adminCommands.autocompleteOpenEvents", "guild_id", ix.GuildID().ToString())
	defer span.End()

	t, err := c.deps.TrialAPI().NewTransaction(ctx, ix.GuildID().ToString(), false)
	if err != nil {
		return nil, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	trials := t.GetTrials(ctx)

	typed := strings.ToLower(focused.ValueString)

	choices := make([]entity.ApplicationCommandOptionChoice, 0, len(trials))
	for _, trial := range trials {
		if trial.GetState(ctx) != storage.TrialStateOpen {
			continue
		}

		name := trial.GetName(ctx)
		nameLower := strings.ToLower(name)

		if !strings.Contains(nameLower, typed) {
			continue
		}

		choices = append(choices, entity.ApplicationCommandOptionChoice{
			Type:        entity.OptTypeString,
			Name:        name,
			ValueString: nameLower,
		})
	}

	return choices, nil
}

func (c *AdminCommands) autocompleteClosedEvents(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption, focused entity.ApplicationCommandInteractionOption) ([]entity.ApplicationCommandOptionChoice, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "adminCommands.autocompleteClosedEvents", "guild_id", ix.GuildID().ToString())
	defer span.End()

	t, err := c.deps.TrialAPI().NewTransaction(ctx, ix.GuildID().ToString(), false)
	if err != nil {
		return nil, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	trials := t.GetTrials(ctx)

	typed := strings.ToLower(focused.ValueString)

	choices := make([]entity.ApplicationCommandOptionChoice, 0, len(trials))
	for _, trial := range trials {
		if trial.GetState(ctx) != storage.TrialStateClosed {
			continue
		}

		name := trial.GetName(ctx)
		nameLower := strings.ToLower(name)

		if !strings.Contains(nameLower, typed) {
			continue
		}

		choices = append(choices, entity.ApplicationCommandOptionChoice{
			Type:        entity.OptTypeString,
			Name:        name,
			ValueString: nameLower,
		})
	}

	return choices, nil
}

func (c *AdminCommands) autocompleteAllEvents(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption, focused entity.ApplicationCommandInteractionOption) ([]entity.ApplicationCommandOptionChoice, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "adminCommands.autocompleteAllEvents", "guild_id", ix.GuildID().ToString())
	defer span.End()

	t, err := c.deps.TrialAPI().NewTransaction(ctx, ix.GuildID().ToString(), false)
	if err != nil {
		return nil, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	trials := t.GetTrials(ctx)

	typed := strings.ToLower(focused.ValueString)

	choices := make([]entity.ApplicationCommandOptionChoice, 0, len(trials))
	for _, trial := range trials {
		name := trial.GetName(ctx)
		nameLower := strings.ToLower(name)

		if !strings.Contains(nameLower, typed) {
			continue
		}

		choices = append(choices, entity.ApplicationCommandOptionChoice{
			Type:        entity.OptTypeString,
			Name:        name,
			ValueString: nameLower,
		})
	}

	return choices, nil
}

func (c *AdminCommands) autocompleteEventRoles(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption, focused entity.ApplicationCommandInteractionOption) ([]entity.ApplicationCommandOptionChoice, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "adminCommands.autocompleteEventRoles", "guild_id", ix.GuildID().ToString())
	defer span.End()

	var eventName string
	for i := range opts {
		if opts[i].Name == "event_name" {
			eventName = opts[i].ValueString
			break
		}
	}

	if eventName == "" {
		return nil, cmdhandler.ErrMalformedInteraction
	}

	t, err := c.deps.TrialAPI().NewTransaction(ctx, ix.GuildID().ToString(), false)
	if err != nil {
		return nil, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	trial, err := t.GetTrial(ctx, eventName)
	if err != nil {
		return nil, err
	}

	roles := trial.GetRoleCounts(ctx)
	typed := strings.ToLower(focused.ValueString)

	choices := make([]entity.ApplicationCommandOptionChoice, 0, len(roles))
	for _, rc := range roles {
		name := rc.GetRole(ctx)
		nameLower := strings.ToLower(name)

		if !strings.Contains(nameLower, typed) {
			continue
		}

		choices = append(choices, entity.ApplicationCommandOptionChoice{
			Type:        entity.OptTypeString,
			Name:        name,
			ValueString: nameLower,
		})
	}

	return choices, nil
}
