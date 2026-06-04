package testutils

import (
	"os"
	"strings"
	"testing"
)

// DefaultLocalAccount is the substring matched against the account name in
// ~/.config/exoscale/exoscale.toml when -account is not provided to `go test`.
const DefaultLocalAccount = "owner-production"

// LoadLocalCreds reads an Exoscale account block from
// $HOME/.config/exoscale/exoscale.toml whose name contains `account` and
// exports its credentials to the test process via t.Setenv. Mirrors the helper
// in exoscale/cli (PR #837) so the same exoscale.toml file works for both the
// CLI and the provider's local acceptance runs.
//
// Also sets TF_ACC=1 if unset, so `resource.Test()` (which gates on the env
// var) runs without the caller having to remember to export it. A pre-set
// TF_ACC is respected.
//
// Use only from `_test.go` files gated by `//go:build local_integration` —
// never link this into CI.
func LoadLocalCreds(t *testing.T, account string) {
	t.Helper()

	if account == "" {
		account = DefaultLocalAccount
	}

	if os.Getenv("TF_ACC") == "" {
		t.Setenv("TF_ACC", "1")
	}

	path := os.ExpandEnv("$HOME/.config/exoscale/exoscale.toml")
	toml, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("LoadLocalCreds: read %s: %v", path, err)
	}

	for _, block := range strings.Split(string(toml), "[[accounts]]")[1:] {
		if !strings.Contains(block, account) {
			continue
		}
		key := tomlStringValue(block, "key")
		secret := tomlStringValue(block, "secret")
		if key == "" || secret == "" {
			continue
		}
		t.Setenv("EXOSCALE_API_KEY", key)
		t.Setenv("EXOSCALE_API_SECRET", secret)
		if zone := tomlStringValue(block, "defaultZone"); zone != "" {
			t.Setenv("EXOSCALE_ZONE", zone)
		}
		return
	}

	t.Fatalf("LoadLocalCreds: no account matching %q in %s", account, path)
}

// tomlStringValue extracts a single-quoted string value for `key` from a raw
// `[[accounts]]` block. Avoids pulling in a toml dependency for a one-off
// read of three fields.
func tomlStringValue(block, key string) string {
	prefix := key + " = '"
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, prefix); ok {
			return strings.TrimSuffix(after, "'")
		}
	}
	return ""
}
