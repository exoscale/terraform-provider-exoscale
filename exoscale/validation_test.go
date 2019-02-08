package exoscale

import (
	"testing"
)

func TestValidateIPv4StringNumber(t *testing.T) {
	_, errs := ValidateIPv4String(15, "test_property")
	if len(errs) == 0 {
		t.Error("an error was expected")
	}
}

func TestValidateIPv4StringNonIP(t *testing.T) {
	_, errs := ValidateIPv4String("hello", "test_property")
	if len(errs) == 0 {
		t.Error("an error was expected")
	}
}

func TestValidateIPv4StringOk(t *testing.T) {
	_, errs := ValidateIPv4String("10.0.0.1", "test_property")
	if len(errs) != 0 {
		t.Error("no errors were expected")
	}
}

func TestValidateIPv4StringKo(t *testing.T) {
	_, errs := ValidateIPv4String("64:ff9b::", "test_property")
	if len(errs) == 0 {
		t.Error("an error was expected")
	}
}

func TestValidateIPv6StringNumber(t *testing.T) {
	_, errs := ValidateIPv6String(15, "test_property")
	if len(errs) == 0 {
		t.Error("an error was expected")
	}
}

func TestValidateIPv6StringNonIP(t *testing.T) {
	_, errs := ValidateIPv6String("hello", "test_property")
	if len(errs) == 0 {
		t.Error("an error was expected")
	}
}

func TestValidateIPv6StringKo(t *testing.T) {
	_, errs := ValidateIPv6String("10.0.0.1", "test_property")
	if len(errs) == 0 {
		t.Error("an error was expected")
	}
}

func TestValidateIPv6StringOk(t *testing.T) {
	_, errs := ValidateIPv6String("64:ff9b::", "test_property")
	if len(errs) != 0 {
		t.Error("no errors were expected")
	}
}

func TestValidatePortRangeOk(t *testing.T) {
	tests := []struct {
		ports string
	}{
		{"0"},
		{"22"},
		{"8000-8080"},
		{"49150"},
	}

	for _, tt := range tests {
		_, errs := ValidatePortRange(tt.ports, "test_property")
		if len(errs) != 0 {
			t.Errorf("no errors were expected %q %v", tt.ports, errs)
		}
	}
}

func TestValidatePortRangeKo(t *testing.T) {
	tests := []struct {
		ports string
	}{
		{"-1"},
		{"22-22"},
		{"22-23-24"},
		{"8000-7000"},
		{"65536"},
	}

	for _, tt := range tests {
		_, errs := ValidatePortRange(tt.ports, "test_property")
		if len(errs) == 0 {
			t.Errorf("an error was expected, %q", tt.ports)
		}
	}
}
