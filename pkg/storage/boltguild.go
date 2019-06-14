package storage

import (
	"github.com/golang/protobuf/proto"
)

type boltGuild struct {
	protoGuild *ProtoGuild
}

func (g *boltGuild) GetName() string {
	return g.protoGuild.Name
}

func (g *boltGuild) SetName(name string) {
	g.protoGuild.Name = name
}

func (g *boltGuild) Serialize() ([]byte, error) {
	return proto.Marshal(g.protoGuild)
}

func (g *boltGuild) GetSettings() GuildSettings {
	s := GuildSettings{
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

func (g *boltGuild) SetSettings(s GuildSettings) {
	g.protoGuild.CommandIndicator = s.ControlSequence
	g.protoGuild.AnnounceChannel = s.AnnounceChannel
	g.protoGuild.AdminChannel = s.AdminChannel
	g.protoGuild.SignupChannel = s.SignupChannel
	g.protoGuild.AnnounceTo = s.AnnounceTo
	g.protoGuild.AdminRole = s.AdminRole

	g.protoGuild.ShowAfterSignup = s.ShowAfterSignup == "true"
	g.protoGuild.ShowAfterWithdraw = s.ShowAfterWithdraw == "true"
}
