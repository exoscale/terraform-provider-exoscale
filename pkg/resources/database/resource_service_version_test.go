package database_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/exoscale/terraform-provider-exoscale/pkg/resources/database"
)

// TestVersionAttributeSchema checks that the version attribute on each
// database service type has the IsMajorVersionValidator wired up.
// No API credentials are required to run this test.
func TestVersionAttributeSchema(t *testing.T) {
	t.Parallel()

	const wantValidatorDesc = "major version number"

	services := []struct {
		name   string
		schema schema.SingleNestedAttribute
	}{
		{"opensearch", database.ResourceOpensearchSchema},
		{"mysql", database.ResourceMysqlSchema},
		{"pg", database.ResourcePgSchema},
	}

	for _, svc := range services {
		t.Run(svc.name, func(t *testing.T) {
			t.Parallel()

			attr, ok := svc.schema.Attributes["version"]
			if !ok {
				t.Fatal("version attribute not found in schema")
			}

			strAttr, ok := attr.(schema.StringAttribute)
			if !ok {
				t.Fatal("version is not a StringAttribute")
			}

			if !strAttr.IsOptional() {
				t.Error("version should be Optional")
			}
			if !strAttr.IsComputed() {
				t.Error("version should be Computed")
			}

			ctx := context.Background()
			var found bool
			for _, v := range strAttr.Validators {
				if strings.Contains(v.Description(ctx), wantValidatorDesc) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s.version is missing the IsMajorVersionValidator", svc.name)
			}
		})
	}
}
