package storage

//go:generate protoc --go_out=. --proto_path=. ./guildapi.proto

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// ErrBadSetting is the error returned if an unknown setting is accessed
var ErrBadSetting = errors.New("bad setting")

// GuildSettings is the set of configuration settings for a guild
type GuildSettings struct {
	ControlSequence   string
	AnnounceChannel   string
	SignupChannel     string
	AdminChannel      string
	AnnounceTo        string
	ShowAfterSignup   string
	ShowAfterWithdraw string
	AdminRole         string
}

// PrettyString returns a multi-line string describing the settings
func (s *GuildSettings) PrettyString() string {
	return fmt.Sprintf(`
GuildSettings:

	- ControlSequence: '%[2]s',
	- AnnounceChannel: '#%[3]s',
	- SignupChannel: '#%[4]s',
	- AdminChannel: '#%[5]s',
	- AnnounceTo: '%[6]s', 
	- ShowAfterSignup: '%[7]s',
	- ShowAfterWithdraw: '%[8]s',
	- AdminRole: '<@&%[9]s>',

	`, "```", s.ControlSequence, s.AnnounceChannel, s.SignupChannel, s.AdminChannel, s.AnnounceTo, s.ShowAfterSignup, s.ShowAfterWithdraw, s.AdminRole)
}

// GetSettingString gets the value of a setting
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
	case "adminrole":
		return s.AdminRole, nil
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

// SetSettingString sets the value of a setting
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
	case "adminrole":
		s.AdminRole = val
		return nil
	default:
		return ErrBadSetting
	}
}

// GuildAPI is the api for managing guild settings transactions
type GuildAPI interface {
	NewTransaction(writable bool) (GuildAPITx, error)
	AllGuilds() ([]string, error)
}

// GuildAPITx is the api for managing guild settings within a transaction
type GuildAPITx interface {
	Commit() error
	Rollback() error

	GetGuild(name string) (Guild, error)
	AddGuild(name string) (Guild, error)
	SaveGuild(guild Guild) error
}

// Guild is the api for managing guild settings for a particular guild
type Guild interface {
	GetName() string
	GetSettings() GuildSettings

	SetName(name string)
	SetSettings(s GuildSettings)

	Serialize() ([]byte, error)
}
