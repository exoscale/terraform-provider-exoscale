package exoscale

import (
	"context"
	"fmt"
	"os"
	"testing"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	v3 "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/egoscale/v3/credentials"
)

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

func Test_defaultString(t *testing.T) {
	type args struct {
		v   *string
		def string
	}

	var (
		testValue        = "test"
		testDefaultValue = "default"
	)

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "nil pointer",
			args: args{
				v:   nil,
				def: testDefaultValue,
			},
			want: testDefaultValue,
		},
		{
			name: "non-nil pointer",
			args: args{
				v:   &testValue,
				def: testDefaultValue,
			},
			want: testValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := defaultString(tt.args.v, tt.args.def); got != tt.want {
				t.Errorf("defaultString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_defaultInt64(t *testing.T) {
	type args struct {
		v   *int64
		def int64
	}

	var (
		testValue        int64 = 1
		testDefaultValue int64 = 2
	)

	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "nil pointer",
			args: args{
				v:   nil,
				def: testDefaultValue,
			},
			want: testDefaultValue,
		},
		{
			name: "non-nil pointer",
			args: args{
				v:   &testValue,
				def: testDefaultValue,
			},
			want: testValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := defaultInt64(tt.args.v, tt.args.def); got != tt.want {
				t.Errorf("defaultInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_defaultBool(t *testing.T) {
	type args struct {
		v   *bool
		def bool
	}

	var (
		testValue        = true
		testDefaultValue = true
	)

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nil pointer",
			args: args{
				v:   nil,
				def: testDefaultValue,
			},
			want: testDefaultValue,
		},
		{
			name: "non-nil pointer",
			args: args{
				v:   &testValue,
				def: testDefaultValue,
			},
			want: testValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := defaultBool(tt.args.v, tt.args.def); got != tt.want {
				t.Errorf("defaultBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func GetTemplateIDByName(templateName string) (*string, error) {
	client := getClient(testAccProvider.Meta())
	ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testEnvironment, testZoneName))

	templates, err := client.ListTemplates(ctx, testZoneName, egoscale.ListTemplatesWithVisibility("public"))
	if err != nil {
		return nil, fmt.Errorf("error retrieving templates: %s", err)
	}

	for _, template := range templates {
		if *template.Name == templateName {
			return template.ID, nil
		}
	}

	return nil, fmt.Errorf("unable to find template: %s", templateName)
}

// backported from testutils.go for cyclical import reasons
func APIClientV3() (*v3.Client, error) {
	creds := credentials.NewStaticCredentials(
		os.Getenv("EXOSCALE_API_KEY"),
		os.Getenv("EXOSCALE_API_SECRET"),
	)

	client, err := v3.NewClient(creds)
	if err != nil {
		return nil, err
	}
	return client, nil
}
