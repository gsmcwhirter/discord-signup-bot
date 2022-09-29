package storage

import (
	"reflect"
	"testing"
)

func Test_diffStringSlices(t *testing.T) {
	t.Parallel()

	type args struct {
		a []string
		b []string
	}
	tests := []struct {
		name  string
		args  args
		wantA []string
		wantB []string
	}{
		{
			name: "a empty",
			args: args{
				a: nil,
				b: []string{"a", "b"},
			},
			wantA: []string{},
			wantB: []string{"a", "b"},
		},
		{
			name: "b empty",
			args: args{
				a: []string{"a", "b"},
				b: nil,
			},
			wantA: []string{"a", "b"},
			wantB: []string{},
		},
		{
			name: "some each",
			args: args{
				a: []string{"a", "b"},
				b: []string{"b", "c"},
			},
			wantA: []string{"a"},
			wantB: []string{"c"},
		},
		{
			name: "order mixing",
			args: args{
				a: []string{"d", "b", "a"},
				b: []string{"c", "a", "d"},
			},
			wantA: []string{"b"},
			wantB: []string{"c"},
		},
		{
			name: "duplicates",
			args: args{
				a: []string{"f", "d", "b", "a", "d", "e", "d"},
				b: []string{"c", "a", "d", "a", "c", "e", "e", "a"},
			},
			wantA: []string{"b", "f"},
			wantB: []string{"c"},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotA, gotB := diffStringSlices(tt.args.a, tt.args.b)
			if !reflect.DeepEqual(gotA, tt.wantA) {
				t.Errorf("diffStringSlices() gotA = %v, wantA %v", gotA, tt.wantA)
			}
			if !reflect.DeepEqual(gotB, tt.wantB) {
				t.Errorf("diffStringSlices() gotB = %v, wantB %v", gotB, tt.wantB)
			}
		})
	}
}
