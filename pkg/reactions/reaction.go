package reactions

import (
	"context"

	"github.com/gsmcwhirter/discord-bot-lib/v20/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v20/snowflake"
)

type Reaction interface {
	UserID() snowflake.Snowflake
	MessageID() snowflake.Snowflake
	ChannelID() snowflake.Snowflake
	GuildID() snowflake.Snowflake
	Emoji() string
	Context() context.Context
}

type reaction struct {
	ctx       context.Context
	userID    snowflake.Snowflake
	channelID snowflake.Snowflake
	messageID snowflake.Snowflake
	guildID   snowflake.Snowflake
	// member    GuildMember
	emoji string
}

// UserID returns the ID of the reactor
func (r *reaction) UserID() snowflake.Snowflake {
	return r.userID
}

// ChannelID returns the ID of the channel the reaction was to
func (r *reaction) ChannelID() snowflake.Snowflake {
	return r.channelID
}

// MessageID returns the ID of the message the reaction was to
func (r *reaction) MessageID() snowflake.Snowflake {
	return r.messageID
}

// GuildID returns the ID of the guild the reaction was to
func (r *reaction) GuildID() snowflake.Snowflake {
	return r.guildID
}

// Emoji returns the emoji of the reaction
// ChannelID returns the ID of the channel the reaction was to
func (r *reaction) Emoji() string {
	return r.emoji
}

func (r *reaction) Context() context.Context {
	return r.ctx
}

func (r *reaction) Contents() []string {
	return []string{r.emoji}
}

func (r *reaction) ContentErr() error {
	return nil
}

var (
	_ Reaction           = (*reaction)(nil)
	_ cmdhandler.Message = (*reaction)(nil)
)

func NewReaction(ctx context.Context, uid, mid, cid, gid snowflake.Snowflake, emoji string) Reaction {
	return &reaction{
		ctx:       ctx,
		userID:    uid,
		messageID: mid,
		channelID: cid,
		guildID:   gid,
		emoji:     emoji,
	}
}
