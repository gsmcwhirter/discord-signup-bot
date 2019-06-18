package storage

import (
	fmt "fmt"
	"sort"
	"strings"

	"github.com/golang/protobuf/proto"
)

const (
	signupCanceled string = "canceled"
	signupOk       string = "ok"
)

type boltTrial struct {
	protoTrial *ProtoTrial
}

func (b *boltTrial) GetName() string {
	return b.protoTrial.Name
}

func (b *boltTrial) GetDescription() string {
	return b.protoTrial.Description
}

func (b *boltTrial) GetAnnounceTo() string {
	return b.protoTrial.AnnounceTo
}

func (b *boltTrial) GetAnnounceChannel() string {
	return b.protoTrial.AnnounceChannel
}

func (b *boltTrial) GetSignupChannel() string {
	return b.protoTrial.SignupChannel
}

func (b *boltTrial) GetState() TrialState {
	return TrialState(b.protoTrial.State)
}

func (b *boltTrial) getSignups(raw bool) []TrialSignup {
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
			name: name,
			role: ps.Role,
		})
	}

	return s
}

func (b *boltTrial) GetSignups() []TrialSignup {
	return b.getSignups(false)
}

func (b *boltTrial) GetRoleCounts() []RoleCount {
	b.migrateRoleCounts()

	s := make([]RoleCount, 0, len(b.protoTrial.RoleCountMap))
	rcNames := make([]string, 0, len(b.protoTrial.RoleCountMap))

	for rName := range b.protoTrial.RoleCountMap {
		rcNames = append(rcNames, rName)
	}
	sort.Strings(rcNames)

	for _, rName := range rcNames {
		r := b.protoTrial.RoleCountMap[rName]
		s = append(s, &boltRoleCount{
			role:  r.Name,
			count: r.Count,
			emoji: r.Emoji,
		})
	}

	return s
}

func (b *boltTrial) PrettyRoles(indent string) string {
	rcs := b.GetRoleCounts()
	lines := make([]string, 0, len(rcs))

	for _, rc := range rcs {
		lines = append(lines, fmt.Sprintf("%s%s: %d", rc.GetEmoji(), rc.GetRole(), rc.GetCount()))
	}

	return strings.Join(lines, "\n"+indent)
}

func (b *boltTrial) PrettySettings() string {
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

	`, b.GetAnnounceChannel(), b.GetSignupChannel(), b.GetAnnounceTo(), b.GetState(), b.PrettyRoles("    "), b.GetDescription())
}

func (b *boltTrial) SetName(name string) {
	b.protoTrial.Name = name
}

func (b *boltTrial) SetDescription(d string) {
	b.protoTrial.Description = d
}

func (b *boltTrial) SetAnnounceChannel(val string) {
	b.protoTrial.AnnounceChannel = val
}

func (b *boltTrial) SetAnnounceTo(val string) {
	b.protoTrial.AnnounceTo = val
}

func (b *boltTrial) SetSignupChannel(val string) {
	b.protoTrial.SignupChannel = val
}

func (b *boltTrial) SetState(state TrialState) {
	b.protoTrial.State = string(state)
}

func isSameUser(dbName, argName string) bool {
	return dbName == argName || userMentionOverflowFix(dbName) == argName
}

func (b *boltTrial) AddSignup(name, role string) {
	lowerRole := strings.ToLower(role)
	s := b.getSignups(true)
	for _, su := range s {
		if isSameUser(su.GetName(), name) && strings.ToLower(su.GetRole()) != lowerRole {
			b.RemoveSignup(su.GetName())
			break
		}

		if isSameUser(su.GetName(), name) && strings.ToLower(su.GetRole()) == lowerRole {
			return
		}
	}

	b.protoTrial.Signups = append(b.protoTrial.Signups, &ProtoTrialSignup{
		Name:  name,
		Role:  role,
		State: signupOk,
	})
}

func (b *boltTrial) RemoveSignup(name string) {
	for i := 0; i < len(b.protoTrial.Signups); i++ {
		if isSameUser(b.protoTrial.Signups[i].Name, name) {
			b.protoTrial.Signups[i].State = signupCanceled
		}
	}
}

func (b *boltTrial) ClearSignups() {
	b.protoTrial.Signups = nil
}

func (b *boltTrial) SetRoleCount(name, emoji string, ct uint64) {
	b.migrateRoleCounts()

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

func (b *boltTrial) RemoveRole(name string) {
	lowerName := strings.ToLower(name)
	if b.protoTrial.RoleCounts == nil && b.protoTrial.RoleCountMap == nil {
		return
	}

	b.migrateRoleCounts()

	if _, ok := b.protoTrial.RoleCountMap[lowerName]; !ok {
		return
	}

	delete(b.protoTrial.RoleCountMap, lowerName)
}

func (b *boltTrial) Serialize() (out []byte, err error) {
	out, err = proto.Marshal(b.protoTrial)
	return
}

func (b *boltTrial) migrateRoleCounts() {
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
	name string
	role string
}

func (b *boltTrialSignup) GetName() string {
	return b.name
}

func (b *boltTrialSignup) GetRole() string {
	return b.role
}

type boltRoleCount struct {
	role  string
	count uint64
	emoji string
}

func (b *boltRoleCount) GetRole() string {
	return b.role
}

func (b *boltRoleCount) GetCount() uint64 {
	return b.count
}

func (b *boltRoleCount) GetEmoji() string {
	return b.emoji
}
