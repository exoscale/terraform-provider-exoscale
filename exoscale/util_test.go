package exoscale

import "testing"

func Test_in(t *testing.T) {
	type args struct {
		list []string
		v    string
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			args: args{
				list: []string{"a", "b", "c"},
				v:    "a",
			},
			want: true,
		},
		{
			args: args{
				list: []string{"a", "b", "c"},
				v:    "z",
			},
			want: false,
		},
		{
			args: args{
				list: []string{"a", "b", "c"},
				v:    "",
			},
			want: false,
		},
		{
			args: args{
				list: nil,
				v:    "a",
			},
			want: false,
		},
		{
			args: args{
				list: nil,
				v:    "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := in(tt.args.list, tt.args.v); got != tt.want {
				t.Errorf("in() = %v, want %v", got, tt.want)
			}
		})
	}
}
