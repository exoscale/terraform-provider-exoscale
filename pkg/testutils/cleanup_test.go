package testutils

import "testing"

func TestProdRefuseMessage(t *testing.T) {
	cases := []struct {
		name, env, override, wantSubstr string
	}{
		{"non-prod passes", "test", "", ""},
		{"prod with override passes", "prod", "1", ""},
		{"prod without override bails", "prod", "", "EXOSCALE_TEST_ALLOW_PROD=1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := prodRefuseMessage(tc.env, tc.override)
			if tc.wantSubstr == "" {
				if got != "" {
					t.Fatalf("got %q, want empty", got)
				}
				return
			}
			if !contains(got, tc.wantSubstr) {
				t.Fatalf("got %q, want substring %q", got, tc.wantSubstr)
			}
		})
	}
}

func contains(s, sub string) bool {
	if sub == "" {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
