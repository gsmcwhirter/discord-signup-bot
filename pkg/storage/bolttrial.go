package storage

import (
	"sort"

	"github.com/golang/protobuf/proto"
)

const (
	signupCanceled string = "canceled"
	signupOk              = "ok"
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

func (b *boltTrial) GetAnnounceChannel() string {
	return b.protoTrial.AnnounceChannel
}

func (b *boltTrial) GetSignupChannel() string {
	return b.protoTrial.SignupChannel
}

func (b *boltTrial) GetState() TrialState {
	return TrialState(b.protoTrial.State)
}

func (b *boltTrial) GetSignups() []TrialSignup {
	s := make([]TrialSignup, 0, len(b.protoTrial.Signups))
	for _, ps := range b.protoTrial.Signups {
		if ps.State == signupCanceled {
			continue
		}

		s = append(s, &boltTrialSignup{
			name: ps.Name,
		})
	}

	return s
}

func (b *boltTrial) GetRoleCounts() []RoleCount {
	s := make([]RoleCount, 0, len(b.protoTrial.RoleCounts))

	rcNames := make([]string, 0, len(b.protoTrial.RoleCounts))
	for rName := range b.protoTrial.RoleCounts {
		rcNames = append(rcNames, rName)
	}
	sort.Strings(rcNames)

	for _, rName := range rcNames {
		rCt := b.protoTrial.RoleCounts[rName]
		s = append(s, &boltRoleCount{
			role:  rName,
			count: rCt,
		})
	}

	return s
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

func (b *boltTrial) SetSignupChannel(val string) {
	b.protoTrial.SignupChannel = val
}

func (b *boltTrial) SetState(state TrialState) {
	b.protoTrial.State = string(state)
}

func (b *boltTrial) AddSignup(name, role string) {
	s := b.GetSignups()
	for _, su := range s {
		if su.GetName() == name && su.GetRole() != role {
			b.RemoveSignup(name)
			break
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
		if b.protoTrial.Signups[i].Name == name {
			b.protoTrial.Signups[i].State = signupCanceled
		}
	}
}

func (b *boltTrial) SetRoleCount(name string, ct uint64) {
	if b.protoTrial.RoleCounts == nil {
		b.protoTrial.RoleCounts = map[string]uint64{}
	}

	b.protoTrial.RoleCounts[name] = ct
}

func (b *boltTrial) RemoveRole(name string) {
	if b.protoTrial.RoleCounts == nil {
		return
	}

	if _, ok := b.protoTrial.RoleCounts[name]; !ok {
		return
	}

	delete(b.protoTrial.RoleCounts, name)
}

func (b *boltTrial) Serialize() (out []byte, err error) {
	out, err = proto.Marshal(b.protoTrial)
	return
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
}

func (b *boltRoleCount) GetRole() string {
	return b.role
}

func (b *boltRoleCount) GetCount() uint64 {
	return b.count
}
