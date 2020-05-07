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

type boltTrial struct {
	protoTrial *ProtoTrial
	census     *telemetry.Census
}

func (b *boltTrial) GetName(ctx context.Context) string {
	return b.protoTrial.Name
}

func (b *boltTrial) GetDescription(ctx context.Context) string {
	return b.protoTrial.Description
}

func (b *boltTrial) GetAnnounceTo(ctx context.Context) string {
	return b.protoTrial.AnnounceTo
}

func (b *boltTrial) GetAnnounceChannel(ctx context.Context) string {
	return b.protoTrial.AnnounceChannel
}

func (b *boltTrial) GetSignupChannel(ctx context.Context) string {
	return b.protoTrial.SignupChannel
}

func (b *boltTrial) GetState(ctx context.Context) TrialState {
	return TrialState(b.protoTrial.State)
}

func (b *boltTrial) getSignups(ctx context.Context, raw bool) []TrialSignup {
	_, span := b.census.StartSpan(ctx, "boltTrial.getSignups")
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

		s = append(s, &boltTrialSignup{
			name:   name,
			role:   ps.Role,
			census: b.census,
		})
	}

	return s
}

func (b *boltTrial) GetSignups(ctx context.Context) []TrialSignup {
	ctx, span := b.census.StartSpan(ctx, "boltTrial.GetSignups")
	defer span.End()

	return b.getSignups(ctx, false)
}

func (b *boltTrial) GetRoleCounts(ctx context.Context) []RoleCount {
	ctx, span := b.census.StartSpan(ctx, "boltTrial.GetRoleCounts")
	defer span.End()

	b.migrateRoleCounts(ctx)

	s := make([]RoleCount, 0, len(b.protoTrial.RoleCountMap))
	rcNames := make([]string, 0, len(b.protoTrial.RoleCountMap))

	for rName := range b.protoTrial.RoleCountMap {
		rcNames = append(rcNames, rName)
	}
	sort.Strings(rcNames)

	for _, rName := range rcNames {
		r := b.protoTrial.RoleCountMap[rName]
		s = append(s, &boltRoleCount{
			role:   r.Name,
			count:  r.Count,
			emoji:  r.Emoji,
			census: b.census,
		})
	}

	return s
}

func (b *boltTrial) PrettyRoles(ctx context.Context, indent string) string {
	ctx, span := b.census.StartSpan(ctx, "boltTrial.PrettyRoles")
	defer span.End()

	rcs := b.GetRoleCounts(ctx)
	lines := make([]string, 0, len(rcs))

	for _, rc := range rcs {
		lines = append(lines, fmt.Sprintf("%s%s: %d", rc.GetEmoji(ctx), rc.GetRole(ctx), rc.GetCount(ctx)))
	}

	return strings.Join(lines, "\n"+indent)
}

func (b *boltTrial) PrettySettings(ctx context.Context) string {
	ctx, span := b.census.StartSpan(ctx, "boltTrial.PrettySettings")
	defer span.End()

	return fmt.Sprintf(`
Event settings:

	- State: '%[4]s',
	- AnnounceChannel: '#%[1]s',
	- SignupChannel: '#%[2]s',
	- AnnounceTo: '%[3]s', 
	- Roles:
		%[5]s

Description:
%s

	`, b.GetAnnounceChannel(ctx), b.GetSignupChannel(ctx), b.GetAnnounceTo(ctx), b.GetState(ctx), b.PrettyRoles(ctx, "    "), b.GetDescription(ctx))
}

func (b *boltTrial) SetName(ctx context.Context, name string) {
	b.protoTrial.Name = name
}

func (b *boltTrial) SetDescription(ctx context.Context, d string) {
	b.protoTrial.Description = d
}

func (b *boltTrial) SetAnnounceChannel(ctx context.Context, val string) {
	b.protoTrial.AnnounceChannel = val
}

func (b *boltTrial) SetAnnounceTo(ctx context.Context, val string) {
	b.protoTrial.AnnounceTo = val
}

func (b *boltTrial) SetSignupChannel(ctx context.Context, val string) {
	b.protoTrial.SignupChannel = val
}

func (b *boltTrial) SetState(ctx context.Context, state TrialState) {
	b.protoTrial.State = string(state)
}

func isSameUser(dbName, argName string) bool {
	return dbName == argName || userMentionOverflowFix(dbName) == argName
}

func (b *boltTrial) AddSignup(ctx context.Context, name, role string) {
	ctx, span := b.census.StartSpan(ctx, "boltTrial.AddSignup")
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

func (b *boltTrial) RemoveSignup(ctx context.Context, name string) {
	_, span := b.census.StartSpan(ctx, "boltTrial.RemoveSignup")
	defer span.End()

	for i := 0; i < len(b.protoTrial.Signups); i++ {
		if isSameUser(b.protoTrial.Signups[i].Name, name) {
			b.protoTrial.Signups[i].State = signupCanceled
		}
	}
}

func (b *boltTrial) ClearSignups(ctx context.Context) {
	_, span := b.census.StartSpan(ctx, "boltTrial.ClearSignups")
	defer span.End()

	b.protoTrial.Signups = nil
}

func (b *boltTrial) SetRoleCount(ctx context.Context, name, emoji string, ct uint64) {
	ctx, span := b.census.StartSpan(ctx, "boltTrial.SetRoleCount")
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

func (b *boltTrial) RemoveRole(ctx context.Context, name string) {
	ctx, span := b.census.StartSpan(ctx, "boltTrial.RemoveRole")
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

func (b *boltTrial) Serialize(ctx context.Context) (out []byte, err error) {
	_, span := b.census.StartSpan(ctx, "boltTrial.Serialize")
	defer span.End()

	out, err = proto.Marshal(b.protoTrial)
	return
}

func (b *boltTrial) migrateRoleCounts(ctx context.Context) {
	_, span := b.census.StartSpan(ctx, "boltTrial.migrateRoleCounts")
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

type boltTrialSignup struct {
	name   string
	role   string
	census *telemetry.Census
}

func (b *boltTrialSignup) GetName(ctx context.Context) string {
	return b.name
}

func (b *boltTrialSignup) GetRole(ctx context.Context) string {
	return b.role
}

type boltRoleCount struct {
	role   string
	count  uint64
	emoji  string
	census *telemetry.Census
}

func (b *boltRoleCount) GetRole(ctx context.Context) string {
	return b.role
}

func (b *boltRoleCount) GetCount(ctx context.Context) uint64 {
	return b.count
}

func (b *boltRoleCount) GetEmoji(ctx context.Context) string {
	return b.emoji
}
