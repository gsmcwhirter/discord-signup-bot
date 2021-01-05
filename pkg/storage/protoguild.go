package storage

import (
	"context"
	"strings"

	"github.com/gsmcwhirter/go-util/v7/telemetry"
	"google.golang.org/protobuf/proto"
)

type protoGuild struct {
	protoGuild *ProtoGuild
	census     *telemetry.Census
}

var _ Guild = (*protoGuild)(nil)

func (g *protoGuild) GetName(ctx context.Context) string {
	return g.protoGuild.Name
}

func (g *protoGuild) SetName(ctx context.Context, name string) {
	g.protoGuild.Name = name
}

func (g *protoGuild) Serialize(ctx context.Context) ([]byte, error) {
	_, span := g.census.StartSpan(ctx, "protoGuild.Serialize")
	defer span.End()

	return proto.Marshal(g.protoGuild)
}

func (g *protoGuild) GetSettings(ctx context.Context) GuildSettings {
	_, span := g.census.StartSpan(ctx, "protoGuild.GetSettings")
	defer span.End()

	s := GuildSettings{
		census:          g.census,
		ControlSequence: g.protoGuild.CommandIndicator,
		AnnounceChannel: g.protoGuild.AnnounceChannel,
		AdminChannel:    g.protoGuild.AdminChannel,
		SignupChannel:   g.protoGuild.SignupChannel,
		AnnounceTo:      g.protoGuild.AnnounceTo,
	}

	if g.protoGuild.AdminRole != "" {
		s.AdminRoles = strings.Split(g.protoGuild.AdminRole, ",")
	}

	if g.protoGuild.ShowAfterSignup {
		s.ShowAfterSignup = "true"
	} else {
		s.ShowAfterSignup = "false"
	}

	if g.protoGuild.ShowAfterWithdraw {
		s.ShowAfterWithdraw = "true"
	} else {
		s.ShowAfterWithdraw = "false"
	}
	return s
}

func (g *protoGuild) SetSettings(ctx context.Context, s GuildSettings) {
	_, span := g.census.StartSpan(ctx, "protoGuild.SetSettings")
	defer span.End()

	g.protoGuild.CommandIndicator = s.ControlSequence
	g.protoGuild.AnnounceChannel = s.AnnounceChannel
	g.protoGuild.AdminChannel = s.AdminChannel
	g.protoGuild.SignupChannel = s.SignupChannel
	g.protoGuild.AnnounceTo = s.AnnounceTo
	g.protoGuild.AdminRole = strings.Join(s.AdminRoles, ",")

	g.protoGuild.ShowAfterSignup = s.ShowAfterSignup == "true"
	g.protoGuild.ShowAfterWithdraw = s.ShowAfterWithdraw == "true"
}
