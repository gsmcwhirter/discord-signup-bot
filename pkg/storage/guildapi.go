package storage

//go:generate protoc --go_out=. --proto_path=. ./guildapi.proto

import (
	"context"
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v7/errors"
	"github.com/gsmcwhirter/go-util/v7/telemetry"
)

// ErrBadSetting is the error returned if an unknown setting is accessed
var ErrBadSetting = errors.New("bad setting")

// GuildSettings is the set of configuration settings for a guild
type GuildSettings struct {
	census            *telemetry.Census
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
func (s *GuildSettings) PrettyString(ctx context.Context) string {
	_, span := s.census.StartSpan(ctx, "GuildSettings.PrettyString")
	defer span.End()

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
func (s *GuildSettings) GetSettingString(ctx context.Context, name string) (string, error) {
	_, span := s.census.StartSpan(ctx, "GuildSettings.GetSettingString")
	defer span.End()

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
func (s *GuildSettings) SetSettingString(ctx context.Context, name, val string) error {
	_, span := s.census.StartSpan(ctx, "GuildSettings.SetSettingString")
	defer span.End()

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
	NewTransaction(ctx context.Context, writable bool) (GuildAPITx, error)
	AllGuilds(ctx context.Context) ([]string, error)
}

// GuildAPITx is the api for managing guild settings within a transaction
type GuildAPITx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error

	GetGuild(ctx context.Context, name string) (Guild, error)
	AddGuild(ctx context.Context, name string) (Guild, error)
	SaveGuild(ctx context.Context, guild Guild) error
}

// Guild is the api for managing guild settings for a particular guild
type Guild interface {
	GetName(ctx context.Context) string
	GetSettings(ctx context.Context) GuildSettings

	SetName(ctx context.Context, name string)
	SetSettings(ctx context.Context, s GuildSettings)

	Serialize(ctx context.Context) ([]byte, error)
}
