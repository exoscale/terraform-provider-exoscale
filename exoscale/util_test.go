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

func TestIsDNSName(t *testing.T) {
	validNames := []string{
		"example.com",
		"sub.example.com",
		"test-domain.org",
		"a.b.c.d.example.net",
		"localhost",
		"example.com.", // FQDN with trailing dot
		"1host.com",    // starts with number
		"host1.com",    // ends with number
		"996b5d08-a6d3-49eb-9b26-c8021ce2b024.sks-ch-gva-2.exo.io", // UUID label
	}

	invalidNames := []string{
		"",             // empty
		"-example.com", // starts with hyphen
		"example-.com", // label ends with hyphen
		"192.168.1.1",  // IP address
		"example..com", // double dot
		"ex@mple.com",  // invalid character
		"verylonglabelverylonglabelverylonglabelverylonglabelverylonglabel.com", // label too long (>63 chars)
	}

	// Test valid names
	for _, name := range validNames {
		t.Run(fmt.Sprintf("valid_%s", name), func(t *testing.T) {
			warnings, errs := isDNSName(name, "test_field")
			if len(errs) != 0 {
				t.Errorf("Expected no errors for valid DNS name %s, got: %v", name, errs)
			}
			if len(warnings) != 0 {
				t.Errorf("Expected no warnings for valid DNS name %s, got: %v", name, warnings)
			}
		})
	}

	// Test invalid names
	for _, name := range invalidNames {
		t.Run(fmt.Sprintf("invalid_%s", name), func(t *testing.T) {
			warnings, errs := isDNSName(name, "test_field")
			if len(errs) == 0 {
				t.Errorf("Expected errors for invalid DNS name %s, got none", name)
			}
			if len(warnings) != 0 {
				t.Errorf("Expected no warnings for invalid DNS name %s, got: %v", name, warnings)
			}
		})
	}

	// Test non-string input
	t.Run("non_string_input", func(t *testing.T) {
		warnings, errs := isDNSName(123, "test_field")
		if len(errs) == 0 {
			t.Error("Expected error for non-string input, got none")
		}
		if len(warnings) != 0 {
			t.Errorf("Expected no warnings for non-string input, got: %v", warnings)
		}
	})
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
