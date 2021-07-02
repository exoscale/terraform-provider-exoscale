package exoscale

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// ValidateString validates that the given field is a string and matches the expected value
func ValidateString(str string) schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(func(i interface{}, k string) (s []string, es []error) {
		value, ok := i.(string)
		if !ok {
			es = append(es, fmt.Errorf("expected type of %s to be string", k))
			return
		}

		if value != str {
			es = append(es, fmt.Errorf("string %q doesn't match expected value %q", value, str))
			return
		}

		return
	})
}

// ValidateStringNot validates that the given field is a string that doesn't match the specified value.
func ValidateStringNot(str string) schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(func(i interface{}, k string) (s []string, es []error) {
		value, ok := i.(string)
		if !ok {
			es = append(es, fmt.Errorf("expected type of %s to be string", k))
			return
		}

		if value == str {
			es = append(es, fmt.Errorf("string %q match expected value %q", value, str))
			return
		}

		return
	})
}

// ValidatePortRange validates that the given field contains a port range.
func ValidatePortRange(i interface{}, k string) (s []string, es []error) {
	value, ok := i.(string)
	if !ok {
		es = append(es, fmt.Errorf("expected type of %s to be string", k))
		return
	}

	ports := strings.Split(value, "-")
	if len(ports) > 2 {
		es = append(es, fmt.Errorf("expected a range of at most two values, got %d", len(ports)))
		return
	}

	ps := make([]uint16, len(ports))
	for i, port := range ports {
		p, err := strconv.ParseUint(port, 10, 16)
		if err != nil {
			es = append(es, err)
			return
		}

		ps[i] = uint16(p)
	}

	if len(ports) == 2 {
		if ps[0] >= ps[1] {
			es = append(es, fmt.Errorf("port range should be ordered, %s < %s", ports[0], ports[1]))
		}
	}

	return
}
