package storage

import (
	"context"

	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/telemetry"
)

type guildData struct {
	Name              string
	CommandIndicator  string
	AnnounceChannel   string
	AdminChannel      string
	SignupChannel     string
	AnnounceTo        string
	ShowAfterSignup   bool
	ShowAfterWithdraw bool
	MessageColor      string
	ErrorColor        string

	AdminRoles []string
}

type pgGuild struct {
	data   guildData
	census *telemetry.Census
}

var _ Guild = (*pgGuild)(nil)

func (g *pgGuild) GetName(ctx context.Context) string {
	return g.data.Name
}

func (g *pgGuild) SetName(ctx context.Context, name string) {
	g.data.Name = name
}

func (g *pgGuild) Serialize(ctx context.Context) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (g *pgGuild) GetSettings(ctx context.Context) GuildSettings {
	_, span := g.census.StartSpan(ctx, "guild.GetSettings")
	defer span.End()

	s := GuildSettings{
		census:          g.census,
		ControlSequence: g.data.CommandIndicator,
		AnnounceChannel: g.data.AnnounceChannel,
		AdminChannel:    g.data.AdminChannel,
		SignupChannel:   g.data.SignupChannel,
		AnnounceTo:      g.data.AnnounceTo,
		AdminRoles:      g.data.AdminRoles,
		MessageColor:    g.data.MessageColor,
		ErrorColor:      g.data.ErrorColor,
	}

	if g.data.ShowAfterSignup {
		s.ShowAfterSignup = "true"
	} else {
		s.ShowAfterSignup = "false"
	}

	if g.data.ShowAfterWithdraw {
		s.ShowAfterWithdraw = "true"
	} else {
		s.ShowAfterWithdraw = "false"
	}
	return s
}

func (g *pgGuild) SetSettings(ctx context.Context, s GuildSettings) {
	_, span := g.census.StartSpan(ctx, "guild.SetSettings")
	defer span.End()

	g.data.CommandIndicator = s.ControlSequence
	g.data.AnnounceChannel = s.AnnounceChannel
	g.data.AdminChannel = s.AdminChannel
	g.data.SignupChannel = s.SignupChannel
	g.data.AnnounceTo = s.AnnounceTo
	g.data.MessageColor = s.MessageColor
	g.data.ErrorColor = s.ErrorColor
	g.data.AdminRoles = s.AdminRoles

	g.data.ShowAfterSignup = s.ShowAfterSignup == "true"
	g.data.ShowAfterWithdraw = s.ShowAfterWithdraw == "true"
}
