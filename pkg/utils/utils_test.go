package utils

import (
"testing"
)

func TestSuppressUserDataDiff(t *testing.T) {
	t.Parallel()

	const plainText = "#cloud-config\npackages:\n  - nginx\n"
	// base64(plainText)
	const b64Plain = "I2Nsb3VkLWNvbmZpZwpwYWNrYWdlczoKICAtIG5naW54Cg=="
	// base64(gzip(plainText))
	const b64Gzip = "H4sIAAAAAAAC/1NOzskvTdFNzs9Ly0znKkhMzk5MTy224lJQ0FXIS8/Mq+ACAL8fHCoiAAAA"

	cases := []struct {
		name     string
		old      string
		new      string
		suppress bool
	}{
		{"identical plain text", plainText, plainText, true},
		{"old plain, new base64", plainText, b64Plain, true},
		{"old base64, new plain", b64Plain, plainText, true},
		{"old plain, new base64+gzip", plainText, b64Gzip, true},
		{"old base64+gzip, new plain", b64Gzip, plainText, true},
		{"both base64", b64Plain, b64Plain, true},
		{"different content", plainText, "#cloud-config\npackages:\n  - curl\n", false},
		{"old empty, new non-empty", "", plainText, false},
		{"both empty", "", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
t.Parallel()
			got := SuppressUserDataDiff("", tc.old, tc.new, nil)
			if got != tc.suppress {
				t.Errorf("SuppressUserDataDiff(%q, %q) = %v, want %v", tc.old, tc.new, got, tc.suppress)
			}
		})
	}
}
