package exoscale

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

// ValidateString validates that the given field is a string and matches the expected value
func ValidateString(str string) schema.SchemaValidateFunc {
	return func(i interface{}, k string) (s []string, es []error) {
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
	}
}

// ValidateStringNot validates that the given field is a string that doesn't match the specified value.
func ValidateStringNot(str string) schema.SchemaValidateFunc {
	return func(i interface{}, k string) (s []string, es []error) {
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
	}
}

// ValidateRegexp validates that the given field is a string and matches the regexp
func ValidateRegexp(pattern string) schema.SchemaValidateFunc {
	return func(i interface{}, k string) (s []string, es []error) {
		value, ok := i.(string)
		if !ok {
			es = append(es, fmt.Errorf("expected type of %s to be string", k))
			return
		}

		r := regexp.MustCompile(pattern)
		if !r.MatchString(value) {
			es = append(es, fmt.Errorf("string %q doesn't match regular expression value %q", value, pattern))
			return
		}

		return
	}
}

// ValidateUUID validates that the given field is a UUID
func ValidateUUID() schema.SchemaValidateFunc {
	return func(i interface{}, k string) (s []string, es []error) {
		value, ok := i.(string)
		if !ok {
			es = append(es, fmt.Errorf("expected type of %s to be string", k))
			return
		}

		if _, err := egoscale.ParseUUID(value); err != nil {
			es = append(es, fmt.Errorf("string %q is not a valid UUID", value))
			return
		}

		return
	}
}

// ValidateIPv4String validates that the given field is a string representing an IPv4 address
func ValidateIPv4String(i interface{}, k string) (s []string, es []error) {
	value, ok := i.(string)
	if !ok {
		es = append(es, fmt.Errorf("expected type of %s to be string", k))
		return
	}

	address := net.ParseIP(value)
	if address == nil {
		es = append(es, fmt.Errorf("expected %s to be an IP address", k))
		return
	}

	if strings.Contains(value, ":") {
		es = append(es, fmt.Errorf("expected %s to be an IPv4 address", k))
	}

	return
}

// ValidateIPv6String validates that the given field is a string representing an IPv6 address
func ValidateIPv6String(i interface{}, k string) (s []string, es []error) {
	value, ok := i.(string)
	if !ok {
		es = append(es, fmt.Errorf("expected type of %s to be string", k))
		return
	}

	address := net.ParseIP(value)
	if address == nil {
		es = append(es, fmt.Errorf("expected %s to be an IP address", k))
		return
	}

	if strings.Contains(value, ".") {
		es = append(es, fmt.Errorf("expected %s to be an IPv6 address", k))
	}

	return
}

// ValidatePortRange validates that the given field contains a port range
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
