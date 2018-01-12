package exoscale

import (
	"fmt"
	"net"

	"github.com/hashicorp/terraform/helper/schema"
)

// StringIPAddress validates that the given field is a string representing an IP address
func StringIPAddress() schema.SchemaValidateFunc {
	return func(i interface{}, k string) (s []string, es []error) {
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

		return
	}
}
