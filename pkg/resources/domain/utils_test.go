package domain

import (
	"testing"
)

func TestDomainNameToUnicode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"example.com", "example.com"},
		{"xn--n3h.ws", "☃.ws"},
		{"xn--domain-with--rcb.ch", "domain-with-ä.ch"},
		{"already-unicodeä.com", "already-unicodeä.com"},
		{"", ""},
	}

	for _, tt := range tests {
		got := domainNameToUnicode(tt.input)
		if got != tt.want {
			t.Errorf("domainNameToUnicode(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
