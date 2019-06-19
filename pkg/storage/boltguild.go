package storage

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/gsmcwhirter/go-util/v4/census"
)

type boltGuild struct {
	protoGuild *ProtoGuild
	census     *census.OpenCensus
}

func (g *boltGuild) GetName(ctx context.Context) string {
	return g.protoGuild.Name
}

func (g *boltGuild) SetName(ctx context.Context, name string) {
	g.protoGuild.Name = name
}

func (g *boltGuild) Serialize(ctx context.Context) ([]byte, error) {
	_, span := g.census.StartSpan(ctx, "boltGuild.Serialize")
	defer span.End()

	return proto.Marshal(g.protoGuild)
}

func (g *boltGuild) GetSettings(ctx context.Context) GuildSettings {
	_, span := g.census.StartSpan(ctx, "boltGuild.GetSettings")
	defer span.End()

	s := GuildSettings{
		census:          g.census,
		ControlSequence: g.protoGuild.CommandIndicator,
		AnnounceChannel: g.protoGuild.AnnounceChannel,
		AdminChannel:    g.protoGuild.AdminChannel,
		SignupChannel:   g.protoGuild.SignupChannel,
		AnnounceTo:      g.protoGuild.AnnounceTo,
		AdminRole:       g.protoGuild.AdminRole,
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

func (g *boltGuild) SetSettings(ctx context.Context, s GuildSettings) {
	_, span := g.census.StartSpan(ctx, "boltGuild.SetSettings")
	defer span.End()

	g.protoGuild.CommandIndicator = s.ControlSequence
	g.protoGuild.AnnounceChannel = s.AnnounceChannel
	g.protoGuild.AdminChannel = s.AdminChannel
	g.protoGuild.SignupChannel = s.SignupChannel
	g.protoGuild.AnnounceTo = s.AnnounceTo
	g.protoGuild.AdminRole = s.AdminRole

	g.protoGuild.ShowAfterSignup = s.ShowAfterSignup == "true"
	g.protoGuild.ShowAfterWithdraw = s.ShowAfterWithdraw == "true"
}
