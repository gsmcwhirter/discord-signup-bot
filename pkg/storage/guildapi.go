package storage

//go:generate protoc --go_out=. --proto_path=. ./guildapi.proto

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// ErrBadSetting TODOC
var ErrBadSetting = errors.New("bad setting")

// GuildSettings TODOC
type GuildSettings struct {
	ControlSequence   string
	AnnounceChannel   string
	SignupChannel     string
	AdminChannel      string
	AnnounceTo        string
	ShowAfterSignup   string
	ShowAfterWithdraw string
}

// PrettyString TODOC
func (s *GuildSettings) PrettyString() string {
	return fmt.Sprintf(`
%[1]s
GuildSettings{
	ControlSequence: '%[2]s',
	AnnounceChannel: '#%[3]s',
	SignupChannel: '#%[4]s',
	AdminChannel: '#%[5]s',
	AnnounceTo: '%[6]s', 
	ShowAfterSignup: '%[7]s',
	ShowAfterWithdraw: '%[8]s',
}
%[1]s
	`, "```", s.ControlSequence, s.AnnounceChannel, s.SignupChannel, s.AdminChannel, s.AnnounceTo, s.ShowAfterSignup, s.ShowAfterWithdraw)
}

// GetSettingString TODOC
func (s *GuildSettings) GetSettingString(name string) (string, error) {
	switch strings.ToLower(name) {
	case "controlsequence":
		return s.ControlSequence, nil
	case "announcechannel":
		return s.AnnounceChannel, nil
	case "adminchannel":
		return s.AdminChannel, nil
	case "signupchannel":
		return s.SignupChannel, nil
	case "announceto":
		return s.AnnounceTo, nil
	case "showaftersignup":
		return s.ShowAfterSignup, nil
	case "showafterwithdraw":
		return s.ShowAfterWithdraw, nil
	default:
		return "", ErrBadSetting
	}
}

func normalizeTrueFalseString(val string) (string, error) {
	switch strings.ToLower(val) {
	case "yes", "true", "ok", "1", "+", "t", "on":
		return "true", nil
	case "", "no", "false", "not ok", "0", "-", "f", "off":
		return "false", nil
	default:
		return val, errors.New("could not understand option value")
	}
}

// SetSettingString TODOC
func (s *GuildSettings) SetSettingString(name, val string) error {
	switch strings.ToLower(name) {
	case "controlsequence":
		s.ControlSequence = val
		return nil
	case "announcechannel":
		s.AnnounceChannel = strings.TrimLeft(val, "#")
		return nil
	case "adminchannel":
		s.AdminChannel = strings.TrimLeft(val, "#")
		return nil
	case "signupchannel":
		s.SignupChannel = strings.TrimLeft(val, "#")
		return nil
	case "announceto":
		s.AnnounceTo = val
		return nil
	case "showaftersignup":
		v, err := normalizeTrueFalseString(val)
		if err != nil {
			return errors.Wrap(err, "could not set ShowAfterSignup")
		}
		s.ShowAfterSignup = v
		return nil
	case "showafterwithdraw":
		v, err := normalizeTrueFalseString(val)
		if err != nil {
			return errors.Wrap(err, "could not set ShowAfterWithdraw")
		}
		s.ShowAfterWithdraw = v
		return nil
	default:
		return ErrBadSetting
	}
}

// GuildAPI TODOC
type GuildAPI interface {
	NewTransaction(writable bool) (GuildAPITx, error)
}

// GuildAPITx TODOC
type GuildAPITx interface {
	Commit() error
	Rollback() error

	GetGuild(name string) (Guild, error)
	AddGuild(name string) (Guild, error)
	SaveGuild(guild Guild) error
}

// Guild TODOC
type Guild interface {
	GetName() string
	GetSettings() GuildSettings

	SetName(name string)
	SetSettings(s GuildSettings)

	Serialize() ([]byte, error)
}
