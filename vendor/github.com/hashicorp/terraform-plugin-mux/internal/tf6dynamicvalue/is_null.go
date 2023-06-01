package tf6dynamicvalue

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// IsNull returns true if the given *tfprotov6.DynamicValue is nil or
// represents a null value.
func IsNull(schema *tfprotov6.Schema, dynamicValue *tfprotov6.DynamicValue) (bool, error) {
	if dynamicValue == nil {
		return true, nil
	}

	// Panic prevention
	if schema == nil {
		return false, fmt.Errorf("unable to unmarshal DynamicValue: missing Type")
	}

	tfValue, err := dynamicValue.Unmarshal(schema.ValueType())

	if err != nil {
		return false, fmt.Errorf("unable to unmarshal DynamicValue: %w", err)
	}

	return tfValue.IsNull(), nil
}
