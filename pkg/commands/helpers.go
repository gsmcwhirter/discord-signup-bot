package commands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gsmcwhirter/discord-bot-lib/cmdhandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
	"github.com/pkg/errors"
)

// type suSorter struct {
// 	signups []storage.TrialSignup
// }

// func (s *suSorter) Len() int {
// 	return len(s.signups)
// }

// func (s *suSorter) Swap(i, j int) {
// 	s.signups[i], s.signups[j] = s.signups[j], s.signups[i]
// }

// func (s *suSorter) Less(i, j int) bool {
// 	return s.signups[i].GetName() < s.signups[j].GetName()
// }

func signupsForRole(role string, signups []storage.TrialSignup, sorted bool) []string {
	roleLower := strings.ToLower(role)
	t := make([]string, 0, len(signups))
	for _, s := range signups {
		if strings.ToLower(s.GetRole()) == roleLower {
			t = append(t, s.GetName())
		}
	}

	if sorted {
		sort.Strings(t)
	}

	return t
}

func roleCountByName(role string, roleCounts []storage.RoleCount) (storage.RoleCount, bool) {
	roleLower := strings.ToLower(role)
	for _, rc := range roleCounts {
		if strings.ToLower(rc.GetRole()) == roleLower {
			return rc, true
		}
	}

	return nil, false
}

func parseSettingDescriptionArgs(args string) (map[string]string, error) {
	argMap := map[string]string{}
	parts := strings.Split(strings.TrimSpace(args), " ")

	for i, pair := range parts {
		if pair == "" {
			continue
		}

		pairParts := strings.SplitN(pair, "=", 2)
		if len(pairParts) < 2 {
			return argMap, errors.New("could not parse arguments")
		}

		if strings.ToLower(pairParts[0]) == "description" {
			argMap[strings.ToLower(pairParts[0])] = fmt.Sprintf("%s %s", pairParts[1], strings.Join(parts[i+1:], " "))
			break
		}
		argMap[strings.ToLower(pairParts[0])] = pairParts[1]
	}

	return argMap, nil
}

func parseRolesString(args string) (map[string]uint64, error) {
	roleMap := map[string]uint64{}
	roles := strings.Split(strings.TrimSpace(args), ",")

	for _, roleStr := range roles {
		if roleStr == "" {
			continue
		}

		roleParts := strings.SplitN(roleStr, ":", 2)
		if len(roleParts) < 2 {
			return roleMap, errors.New("could not parse roles")
		}

		roleCt, err := strconv.Atoi(roleParts[1])
		if err != nil {
			return roleMap, err
		}
		roleMap[roleParts[0]] = uint64(roleCt)
	}

	return roleMap, nil
}

func formatTrialDisplay(trial storage.Trial, withState bool) *cmdhandler.EmbedResponse {
	r := &cmdhandler.EmbedResponse{}

	if withState {
		r.Title = fmt.Sprintf("__%s__ (%s)", trial.GetName(), string(trial.GetState()))
	} else {
		r.Title = fmt.Sprintf("__%s__", trial.GetName())
	}
	r.Description = trial.GetDescription()
	r.Fields = []cmdhandler.EmbedField{}

	overflowFields := []cmdhandler.EmbedField{}

	roleCounts := trial.GetRoleCounts() // already sorted by name
	signups := trial.GetSignups()

	for _, rc := range roleCounts {
		lowerRole := strings.ToLower(rc.GetRole())
		suNames := make([]string, 0, len(signups))
		ofNames := make([]string, 0, len(signups))
		for _, su := range signups {
			if strings.ToLower(su.GetRole()) != lowerRole {
				continue
			}

			if uint64(len(suNames)) < rc.GetCount() {
				suNames = append(suNames, su.GetName())
			} else {
				ofNames = append(ofNames, su.GetName())
			}
		}

		if len(suNames) > 0 {
			r.Fields = append(r.Fields, cmdhandler.EmbedField{
				Name: fmt.Sprintf("*%s* (%d/%d)", rc.GetRole(), len(suNames), rc.GetCount()),
				Val:  strings.Join(suNames, "\n") + "\n",
			})
		} else {
			r.Fields = append(r.Fields, cmdhandler.EmbedField{
				Name: fmt.Sprintf("*%s* (%d/%d)", rc.GetRole(), len(suNames), rc.GetCount()),
				Val:  "(empty)",
			})
		}

		if len(ofNames) > 0 {
			overflowFields = append(overflowFields, cmdhandler.EmbedField{
				Name: fmt.Sprintf("*Overflow %s* (%d)", rc.GetRole(), len(ofNames)),
				Val:  strings.Join(ofNames, "\n") + "\n",
			})
		}
	}

	r.Fields = append(r.Fields, overflowFields...)

	return r
}

func signupUser(trial storage.Trial, userMentionStr, role string) (overflow bool, err error) {
	roleCounts := trial.GetRoleCounts() // already sorted by name
	rc, known := roleCountByName(role, roleCounts)
	if !known {
		err = errors.New("unknown role")
		return
	}

	signups := trial.GetSignups()
	roleSignups := signupsForRole(role, signups, false)

	trial.AddSignup(userMentionStr, role)
	overflow = uint64(len(roleSignups)) >= rc.GetCount()

	return
}
