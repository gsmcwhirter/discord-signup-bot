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
	return
}

func (g *boltGuild) SetSettings(s GuildSettings) {
	g.protoGuild.CommandIndicator = s.ControlSequence
}
