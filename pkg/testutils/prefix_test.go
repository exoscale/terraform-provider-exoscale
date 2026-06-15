package testutils

import "testing"

func TestShortSHA(t *testing.T) {
	s := ShortSHA()
	if s == "" {
		t.Fatal("ShortSHA returned empty string")
	}
	if len(s) > 8 {
		t.Fatalf("ShortSHA returned %q, want length <= 8", s)
	}
	if s == "local" {
		t.Skip("running outside a git checkout; skipping hex-shape check")
	}
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			t.Fatalf("ShortSHA returned %q, want lowercase hex", s)
		}
	}
}

func TestTestResourceName(t *testing.T) {
	n := TestResourceName()
	if want := ResourcePrefix + "-"; len(n) < len(want) || n[:len(want)] != want {
		t.Fatalf("TestResourceName() = %q, want prefix %q", n, want)
	}
	if n == TestResourceName() {
		t.Fatal("TestResourceName() returned the same value twice; expected a random suffix")
	}
}
