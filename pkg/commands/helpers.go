package commands

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gsmcwhirter/discord-bot-lib/v20/bot"
	"github.com/gsmcwhirter/discord-bot-lib/v20/bot/session"
	"github.com/gsmcwhirter/discord-bot-lib/v20/cmdhandler"
	"github.com/gsmcwhirter/go-util/v8/errors"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

type Logger = interface {
	Log(keyvals ...interface{}) error
	Message(string, ...interface{})
	Err(string, error, ...interface{})
	Printf(string, ...interface{})
}

var ErrUnknownRole = errors.New("unknown role")

var (
	isAdminAuthorized = msghandler.IsAdminAuthorized
	isAdminChannel    = msghandler.IsAdminChannel
)

func isSignupChannel(ctx context.Context, logger Logger, msg msghandler.MessageLike, signupChannel, adminChannel string, adminRoles []string, sess *session.Session, b *bot.DiscordBot) bool {
	if msghandler.IsSignupChannel(msg, signupChannel, sess) {
		return true
	}

	if !isAdminChannel(logger, msg, adminChannel, sess) {
		return false
	}

	return isAdminAuthorized(ctx, logger, msg, adminRoles, sess, b)
}

func signupsForRole(ctx context.Context, role string, signups []storage.TrialSignup, sorted bool) []string {
	roleLower := strings.ToLower(role)
	t := make([]string, 0, len(signups))
	for _, s := range signups {
		if strings.ToLower(s.GetRole(ctx)) == roleLower {
			t = append(t, s.GetName(ctx))
		}
	}

	if sorted {
		sort.Strings(t)
	}

	return t
}

func roleCountByName(ctx context.Context, role string, roleCounts []storage.RoleCount) (storage.RoleCount, bool) {
	roleLower := strings.ToLower(role)
	for _, rc := range roleCounts {
		if strings.ToLower(rc.GetRole(ctx)) == roleLower {
			return rc, true
		}
	}

	return nil, false
}

func parseSettingDescriptionArgs(args []string) (map[string]string, error) {
	argMap := map[string]string{}

	for _, pair := range args {
		if pair == "" {
			continue
		}

		pairParts := strings.SplitN(pair, "=", 2)
		if len(pairParts) < 2 {
			return argMap, errors.New("could not parse arguments")
		}

		// if strings.ToLower(pairParts[0]) == "description" {
		// 	argMap[strings.ToLower(pairParts[0])] = fmt.Sprintf("%s %s", pairParts[1], strings.Join(parts[i+1:], " "))
		// 	break
		// }

		argMap[strings.ToLower(pairParts[0])] = pairParts[1]
	}

	return argMap, nil
}

type roleCtEmo struct {
	role string
	ct   uint64
	emo  string
}

func parseRolesString(args string) ([]roleCtEmo, error) {
	roles := strings.Split(strings.TrimSpace(args), ",")
	roleEmoCt := make([]roleCtEmo, 0, len(roles))

	for _, roleStr := range roles {
		if roleStr == "" {
			continue
		}

		roleParts := strings.SplitN(roleStr, ":", 3)
		if len(roleParts) < 2 {
			return roleEmoCt, errors.New("could not parse roles")
		}

		roleCt, err := strconv.Atoi(roleParts[1])
		if err != nil {
			return roleEmoCt, err
		}

		var emo string
		if len(roleParts) == 3 {
			emo = roleParts[2]
		}

		roleEmoCt = append(roleEmoCt, roleCtEmo{
			role: roleParts[0],
			ct:   uint64(roleCt),
			emo:  emo,
		})
	}

	return roleEmoCt, nil
}

func getTrialRoleSignups(ctx context.Context, signups []storage.TrialSignup, rc storage.RoleCount) ([]string, []string) {
	lowerRole := strings.ToLower(rc.GetRole(ctx))
	suNames := make([]string, 0, len(signups))
	ofNames := make([]string, 0, len(signups))
	for _, su := range signups {
		if strings.ToLower(su.GetRole(ctx)) != lowerRole {
			continue
		}

		if uint64(len(suNames)) < rc.GetCount(ctx) {
			suNames = append(suNames, su.GetName(ctx))
		} else {
			ofNames = append(ofNames, su.GetName(ctx))
		}
	}

	return suNames, ofNames
}

func formatTrialDisplay(ctx context.Context, trial storage.Trial, withState bool) *cmdhandler.EmbedResponse {
	r := &cmdhandler.EmbedResponse{}

	if withState {
		r.Title = fmt.Sprintf("__%s__ (%s)", trial.GetName(ctx), string(trial.GetState(ctx)))
	} else {
		r.Title = fmt.Sprintf("__%s__", trial.GetName(ctx))
	}
	r.Description = trial.GetDescription(ctx)
	r.Fields = []cmdhandler.EmbedField{}

	overflowFields := []cmdhandler.EmbedField{}

	roleCounts := trial.GetRoleCounts(ctx) // already sorted by name
	signups := trial.GetSignups(ctx)

	emojis := make([]string, 0, len(roleCounts))

	for _, rc := range roleCounts {
		suNames, ofNames := getTrialRoleSignups(ctx, signups, rc)

		emoji := rc.GetEmoji(ctx)

		if len(suNames) > 0 {
			r.Fields = append(r.Fields, cmdhandler.EmbedField{
				Name: fmt.Sprintf("*%s* %s (%d/%d)", rc.GetRole(ctx), emoji, len(suNames), rc.GetCount(ctx)),
				Val:  strings.Join(suNames, "\n") + "\n_ _\n",
			})
		} else {
			r.Fields = append(r.Fields, cmdhandler.EmbedField{
				Name: fmt.Sprintf("*%s* %s (%d/%d)", rc.GetRole(ctx), emoji, len(suNames), rc.GetCount(ctx)),
				Val:  "(empty)\n_ _\n",
			})
		}

		if len(ofNames) > 0 {
			overflowFields = append(overflowFields, cmdhandler.EmbedField{
				Name: fmt.Sprintf("*Overflow %s* %s (%d)", rc.GetRole(ctx), emoji, len(ofNames)),
				Val:  strings.Join(ofNames, "\n") + "\n_ _\n",
			})
		}

		if emoji != "" {
			emojis = append(emojis, emoji)
		}
	}

	r.Fields = append(r.Fields, overflowFields...)

	if !trial.HideReactionsShow(ctx) {
		r.Reactions = emojis
	}
	r.FooterText = fmt.Sprintf("event:%s", trial.GetName(ctx))

	return r
}

func signupUser(ctx context.Context, trial storage.Trial, userMentionStr, role string) (bool, error) {
	roleCounts := trial.GetRoleCounts(ctx) // already sorted by name
	rc, known := roleCountByName(ctx, role, roleCounts)
	if !known {
		return false, ErrUnknownRole
	}

	trial.AddSignup(ctx, userMentionStr, role)

	signups := trial.GetSignups(ctx)
	roleSignups := signupsForRole(ctx, role, signups, false)

	overflow := uint64(len(roleSignups)) > rc.GetCount(ctx)

	return overflow, nil
}

func colorToInt(c string) (int, error) {
	if c == "" {
		return 0, nil
	}

	c = strings.ToLower(c)
	v, err := strconv.ParseInt(c, 16, 64)
	if err != nil {
		return 0, errors.Wrap(err, "could not understand color")
	}

	return int(v), nil
}
