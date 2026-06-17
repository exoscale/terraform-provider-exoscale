package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
)

func TestProviderSchema_DefaultLabels(t *testing.T) {
	t.Parallel()

	p := &ExoscaleProvider{}
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, resp)

	if _, ok := resp.Schema.Attributes[DefaultLabelsAttrName]; !ok {
		t.Fatalf("expected %q in provider schema", DefaultLabelsAttrName)
	}
}
