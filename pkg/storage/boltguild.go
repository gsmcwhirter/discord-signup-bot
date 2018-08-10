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

func (g *boltGuild) Serialize() (out []byte, err error) {
	out, err = proto.Marshal(g.protoGuild)
	return
}

func (g *boltGuild) GetSettings() (s GuildSettings) {
	s.ControlSequence = g.protoGuild.CommandIndicator
	s.AnnounceChannel = g.protoGuild.AnnounceChannel
	s.AdminChannel = g.protoGuild.AdminChannel
	s.SignupChannel = g.protoGuild.SignupChannel
	s.AnnounceTo = g.protoGuild.AnnounceTo

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
	return
}

func (g *boltGuild) SetSettings(s GuildSettings) {
	g.protoGuild.CommandIndicator = s.ControlSequence
	g.protoGuild.AnnounceChannel = s.AnnounceChannel
	g.protoGuild.AdminChannel = s.AdminChannel
	g.protoGuild.SignupChannel = s.SignupChannel
	g.protoGuild.AnnounceTo = s.AnnounceTo

	g.protoGuild.ShowAfterSignup = s.ShowAfterSignup == "true"
	g.protoGuild.ShowAfterWithdraw = s.ShowAfterWithdraw == "true"
}
