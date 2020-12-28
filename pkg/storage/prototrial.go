package storage

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/gsmcwhirter/go-util/v7/telemetry"
	"google.golang.org/protobuf/proto"
)

const (
	signupCanceled string = "canceled"
	signupOk       string = "ok"
)

type protoTrial struct {
	protoTrial *ProtoTrial
	census     *telemetry.Census
}

func (b *protoTrial) GetName(ctx context.Context) string {
	_, span := b.census.StartSpan(ctx, "protoTrial.GetName")
	defer span.End()
	return b.protoTrial.Name
}

func (b *protoTrial) GetDescription(ctx context.Context) string {
	_, span := b.census.StartSpan(ctx, "protoTrial.GetDescription")
	defer span.End()
	return b.protoTrial.Description
}

func (b *protoTrial) GetAnnounceTo(ctx context.Context) string {
	_, span := b.census.StartSpan(ctx, "protoTrial.GetAnnounceTo")
	defer span.End()
	return b.protoTrial.AnnounceTo
}

func (b *protoTrial) GetAnnounceChannel(ctx context.Context) string {
	_, span := b.census.StartSpan(ctx, "protoTrial.GetAnnounceChannel")
	defer span.End()
	return b.protoTrial.AnnounceChannel
}

func (b *protoTrial) GetSignupChannel(ctx context.Context) string {
	_, span := b.census.StartSpan(ctx, "protoTrial.GetSignupChannel")
	defer span.End()
	return b.protoTrial.SignupChannel
}

func (b *protoTrial) GetState(ctx context.Context) TrialState {
	_, span := b.census.StartSpan(ctx, "protoTrial.GetState")
	defer span.End()
	return TrialState(b.protoTrial.State)
}

func (b *protoTrial) getSignups(ctx context.Context, raw bool) []TrialSignup {
	_, span := b.census.StartSpan(ctx, "protoTrial.getSignups")
	defer span.End()

	s := make([]TrialSignup, 0, len(b.protoTrial.Signups))
	for _, ps := range b.protoTrial.Signups {
		if ps.State == signupCanceled {
			continue
		}

		name := ps.Name
		if !raw {
			name = userMentionOverflowFix(name)
		}

		s = append(s, &protoTrialSignup{
			name:   name,
			role:   ps.Role,
			census: b.census,
		})
	}

	return s
}

func (b *protoTrial) GetSignups(ctx context.Context) []TrialSignup {
	ctx, span := b.census.StartSpan(ctx, "protoTrial.GetSignups")
	defer span.End()

	return b.getSignups(ctx, false)
}

func (b *protoTrial) GetRoleCounts(ctx context.Context) []RoleCount {
	ctx, span := b.census.StartSpan(ctx, "protoTrial.GetRoleCounts")
	defer span.End()

	b.migrateRoleCounts(ctx)

	s := RoleCountSlice(make([]RoleCount, 0, len(b.protoTrial.RoleCountMap)))
	rcNames := make([]string, 0, len(b.protoTrial.RoleCountMap))

	for rName := range b.protoTrial.RoleCountMap {
		rcNames = append(rcNames, rName)
	}
	sort.Strings(rcNames)

	ord := b.GetRoleOrder(ctx)
	ordMap := map[string]int{}
	for i, r := range ord {
		ordMap[r] = i
	}

	for i, rName := range rcNames {
		idx, ok := ordMap[strings.ToLower(rName)]
		if !ok {
			idx = len(ord) + i
		}

		r := b.protoTrial.RoleCountMap[rName]
		s = append(s, &boltRoleCount{
			role:   r.Name,
			count:  r.Count,
			emoji:  r.Emoji,
			census: b.census,
			index:  idx,
		})
	}

	sort.Sort(s)

	return []RoleCount(s)
}

func (b *protoTrial) GetRoleOrder(ctx context.Context) []string {
	_, span := b.census.StartSpan(ctx, "protoTrial.GetRoleOrder")
	defer span.End()

	return b.protoTrial.RoleSortOrder
}

func (b *protoTrial) PrettyRoleOrder(ctx context.Context) string {
	ctx, span := b.census.StartSpan(ctx, "protoTrial.PrettyRoleOrder")
	defer span.End()

	ord := b.GetRoleOrder(ctx)
	return strings.Join(ord, ", ")
}

func (b *protoTrial) PrettyRoles(ctx context.Context, indent string) string {
	ctx, span := b.census.StartSpan(ctx, "protoTrial.PrettyRoles")
	defer span.End()

	rcs := b.GetRoleCounts(ctx)
	lines := make([]string, 0, len(rcs))

	for _, rc := range rcs {
		lines = append(lines, fmt.Sprintf("%s%s: %d", rc.GetEmoji(ctx), rc.GetRole(ctx), rc.GetCount(ctx)))
	}

	return strings.Join(lines, "\n"+indent)
}

func (b *protoTrial) PrettySettings(ctx context.Context) string {
	ctx, span := b.census.StartSpan(ctx, "protoTrial.PrettySettings")
	defer span.End()

	return fmt.Sprintf(`
Event settings:
%[1]s
	- State: '%[5]s',
	- AnnounceChannel: '#%[2]s',
	- SignupChannel: '#%[3]s',
	- AnnounceTo: '%[4]s', 
	- RoleOrder: '%[8]s',
	- Roles:
		%[6]s
%[1]s

Description:
%[1]s
%[7]s

%[1]s`, "", b.GetAnnounceChannel(ctx), b.GetSignupChannel(ctx), b.GetAnnounceTo(ctx), b.GetState(ctx), b.PrettyRoles(ctx, "		"), b.GetDescription(ctx), b.PrettyRoleOrder(ctx))
}

func (b *protoTrial) SetName(ctx context.Context, name string) {
	_, span := b.census.StartSpan(ctx, "protoTrial.SetName")
	defer span.End()
	b.protoTrial.Name = name
}

func (b *protoTrial) SetDescription(ctx context.Context, d string) {
	_, span := b.census.StartSpan(ctx, "protoTrial.SetDescription")
	defer span.End()
	b.protoTrial.Description = d
}

func (b *protoTrial) SetAnnounceChannel(ctx context.Context, val string) {
	_, span := b.census.StartSpan(ctx, "protoTrial.SetAnnounceChannel")
	defer span.End()
	b.protoTrial.AnnounceChannel = val
}

func (b *protoTrial) SetAnnounceTo(ctx context.Context, val string) {
	_, span := b.census.StartSpan(ctx, "protoTrial.SetAnnounceTo")
	defer span.End()
	b.protoTrial.AnnounceTo = val
}

func (b *protoTrial) SetSignupChannel(ctx context.Context, val string) {
	_, span := b.census.StartSpan(ctx, "protoTrial.SetSignupChannel")
	defer span.End()
	b.protoTrial.SignupChannel = val
}

func (b *protoTrial) SetState(ctx context.Context, state TrialState) {
	_, span := b.census.StartSpan(ctx, "protoTrial.SetState")
	defer span.End()
	b.protoTrial.State = string(state)
}

func isSameUser(dbName, argName string) bool {
	return dbName == argName || userMentionOverflowFix(dbName) == argName
}

func (b *protoTrial) AddSignup(ctx context.Context, name, role string) {
	ctx, span := b.census.StartSpan(ctx, "protoTrial.AddSignup")
	defer span.End()

	lowerRole := strings.ToLower(role)
	s := b.getSignups(ctx, true)
	for _, su := range s {
		suName := su.GetName(ctx)
		suRole := su.GetRole(ctx)
		if isSameUser(suName, name) && strings.ToLower(suRole) != lowerRole {
			b.RemoveSignup(ctx, suName)
			break
		}

		if isSameUser(suName, name) && strings.ToLower(suRole) == lowerRole {
			return
		}
	}

	b.protoTrial.Signups = append(b.protoTrial.Signups, &ProtoTrialSignup{
		Name:  name,
		Role:  role,
		State: signupOk,
	})
}

func (b *protoTrial) RemoveSignup(ctx context.Context, name string) {
	_, span := b.census.StartSpan(ctx, "protoTrial.RemoveSignup")
	defer span.End()

	for i := 0; i < len(b.protoTrial.Signups); i++ {
		if isSameUser(b.protoTrial.Signups[i].Name, name) {
			b.protoTrial.Signups[i].State = signupCanceled
		}
	}
}

func (b *protoTrial) ClearSignups(ctx context.Context) {
	_, span := b.census.StartSpan(ctx, "protoTrial.ClearSignups")
	defer span.End()

	b.protoTrial.Signups = nil
}

func (b *protoTrial) SetRoleCount(ctx context.Context, name, emoji string, ct uint64) {
	ctx, span := b.census.StartSpan(ctx, "protoTrial.SetRoleCount")
	defer span.End()

	b.migrateRoleCounts(ctx)

	if b.protoTrial.RoleCountMap == nil {
		b.protoTrial.RoleCountMap = map[string]*ProtoRoleCount{}
	}

	lowerName := strings.ToLower(name)
	prc, ok := b.protoTrial.RoleCountMap[lowerName]
	if !ok {
		prc = new(ProtoRoleCount)
	}

	prc.Name = name
	prc.Emoji = emoji
	prc.Count = ct

	b.protoTrial.RoleCountMap[lowerName] = prc
}

func (b *protoTrial) RemoveRole(ctx context.Context, name string) {
	ctx, span := b.census.StartSpan(ctx, "protoTrial.RemoveRole")
	defer span.End()

	lowerName := strings.ToLower(name)
	if b.protoTrial.RoleCounts == nil && b.protoTrial.RoleCountMap == nil {
		return
	}

	b.migrateRoleCounts(ctx)

	if _, ok := b.protoTrial.RoleCountMap[lowerName]; !ok {
		return
	}

	delete(b.protoTrial.RoleCountMap, lowerName)
}

func (b *protoTrial) SetRoleOrder(ctx context.Context, ord []string) {
	_, span := b.census.StartSpan(ctx, "protoTrial.SetRoleOrder")
	defer span.End()

	b.protoTrial.RoleSortOrder = nil
	for _, role := range ord {
		b.protoTrial.RoleSortOrder = append(b.protoTrial.RoleSortOrder, strings.ToLower(role))
	}
}

func (b *protoTrial) Serialize(ctx context.Context) (out []byte, err error) {
	_, span := b.census.StartSpan(ctx, "protoTrial.Serialize")
	defer span.End()

	out, err = proto.Marshal(b.protoTrial)
	return
}

func (b *protoTrial) migrateRoleCounts(ctx context.Context) {
	_, span := b.census.StartSpan(ctx, "protoTrial.migrateRoleCounts")
	defer span.End()

	if b.protoTrial.RoleCountMap != nil {
		return
	}

	b.protoTrial.RoleCountMap = map[string]*ProtoRoleCount{}
	for k, v := range b.protoTrial.RoleCounts {
		prc := new(ProtoRoleCount)
		prc.Name = k
		prc.Count = v
		prc.Emoji = ""
		b.protoTrial.RoleCountMap[strings.ToLower(k)] = prc
	}
}

type protoTrialSignup struct {
	name   string
	role   string
	census *telemetry.Census
}

func (b *protoTrialSignup) GetName(ctx context.Context) string {
	_, span := b.census.StartSpan(ctx, "protoTrialSignup.GetName")
	defer span.End()

	return b.name
}

func (b *protoTrialSignup) GetRole(ctx context.Context) string {
	_, span := b.census.StartSpan(ctx, "protoTrialSignup.GetRole")
	defer span.End()

	return b.role
}

type RoleCountSlice []RoleCount

func (s RoleCountSlice) Len() int {
	return len(s)
}

func (s RoleCountSlice) Less(i, j int) bool {
	return s[i].Index() < s[j].Index()
}

func (s RoleCountSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type boltRoleCount struct {
	role   string
	count  uint64
	emoji  string
	census *telemetry.Census
	index  int
}

func (b *boltRoleCount) GetRole(ctx context.Context) string {
	_, span := b.census.StartSpan(ctx, "boltRoleCount.GetRole")
	defer span.End()

	return b.role
}

func (b *boltRoleCount) GetCount(ctx context.Context) uint64 {
	_, span := b.census.StartSpan(ctx, "boltRoleCount.GetCount")
	defer span.End()

	return b.count
}

func (b *boltRoleCount) GetEmoji(ctx context.Context) string {
	_, span := b.census.StartSpan(ctx, "boltRoleCount.GetEmoji")
	defer span.End()

	return b.emoji
}

func (b *boltRoleCount) Index() int {
	return b.index
}
