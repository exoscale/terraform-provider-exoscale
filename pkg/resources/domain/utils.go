package domain

import (
	"golang.org/x/net/idna"
)

// domainNameToUnicode converts an ACE/punycode domain name to its Unicode
// representation. If the name is already Unicode, or conversion fails, the
// original value is returned unchanged. This is used to match a
// punycode-configured domain name against the Unicode names the API returns.
func domainNameToUnicode(name string) string {
	unicode, err := idna.ToUnicode(name)
	if err != nil {
		return name
	}
	return unicode
}
