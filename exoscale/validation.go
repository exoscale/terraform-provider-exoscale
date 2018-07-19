package exoscale

import (
	"fmt"
	"net"
	"strings"
)

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
