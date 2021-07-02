package exoscale

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
)

func TestValidateStringKo(t *testing.T) {
	f := ValidateString("exoscale")
	diags := f("not exoscale", cty.GetAttrPath("test_property"))
	if !diags.HasError() {
		t.Error("an error was expected")
	}
}

func TestValidateStringOk(t *testing.T) {
	f := ValidateString("exoscale")
	diags := f("exoscale", cty.GetAttrPath("test_property"))
	if diags.HasError() {
		t.Errorf("no errors were expected, got %v", diags)
	}
}

func TestValidatePortRange(t *testing.T) {
	type args struct {
		i interface{}
		k string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			args:    args{i: "-1", k: "ports"},
			wantErr: true,
		},
		{
			args:    args{i: "22-22", k: "ports"},
			wantErr: true,
		},
		{
			args:    args{i: "8000-7000", k: "ports"},
			wantErr: true,
		},
		{
			args:    args{i: "65536", k: "ports"},
			wantErr: true,
		},
		{
			args: args{i: "0", k: "ports"},
		},
		{
			args: args{i: "22", k: "ports"},
		},
		{
			args: args{i: "8000-8080", k: "ports"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, es := ValidatePortRange(tt.args.i, tt.args.k)
			if (len(es) > 0) != tt.wantErr {
				t.Errorf("ValidatePortRange() error = %v, wantErr %v", es, tt.wantErr)
				return
			}
		})
	}
}
