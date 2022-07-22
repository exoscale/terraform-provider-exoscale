package exoscale

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/go-cty/cty"
)

func Test_validateString(t *testing.T) {
	type args struct {
		s    string
		i    interface{}
		path cty.Path
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid type",
			args: args{
				s:    "test",
				i:    42,
				path: cty.GetAttrPath("test"),
			},
			wantErr: true,
		},
		{
			name: "invalid value",
			args: args{
				s:    "test",
				i:    "lolnope",
				path: cty.GetAttrPath("test"),
			},
			wantErr: true,
		},
		{
			name: "ok",
			args: args{
				s:    "test",
				i:    "test",
				path: cty.GetAttrPath("test"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diags := validateString(tt.args.s)(tt.args.i, tt.args.path); diags.HasError() != tt.wantErr {
				t.Errorf("validateString() diags = %v, wantErr %v", diags, tt.wantErr)
				return
			}
		})
	}
}

func Test_validatePortRange(t *testing.T) {
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
			args:    args{i: "0", k: "ports"},
			wantErr: true,
		},
		{
			args: args{i: "1", k: "ports"},
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
			_, es := validatePortRange(tt.args.i, tt.args.k)
			if (len(es) > 0) != tt.wantErr {
				t.Errorf("validatePortRange() error = %v, wantErr %v", es, tt.wantErr)
				return
			}
		})
	}
}

func Test_validateComputeInstanceType(t *testing.T) {
	type args struct {
		i    interface{}
		path cty.Path
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid type",
			args: args{
				i:    42,
				path: cty.GetAttrPath("test"),
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			args: args{
				i:    "lolnope",
				path: cty.GetAttrPath("test"),
			},
			wantErr: true,
		},
		{
			name: "ok",
			args: args{
				i:    "standard.medium",
				path: cty.GetAttrPath("test"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diags := validateComputeInstanceType(tt.args.i, tt.args.path); diags.HasError() != tt.wantErr {
				t.Errorf("validateComputeInstanceType() diags = %v, wantErr %v", diags, tt.wantErr)
			}
		})
	}
}

var testPemCertificateFormatRegex = regexp.MustCompile(fmt.Sprintf(`^-----BEGIN CERTIFICATE-----\n(.|\s)+\n-----END CERTIFICATE-----\n$`))
