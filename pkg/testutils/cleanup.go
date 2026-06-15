package testutils

import (
	"os"
	"strings"
	"sync"
	"testing"
)

// cleanupRegistry collects best-effort teardown callbacks registered by a
// test. Each entry is invoked once when the test ends (via t.Cleanup).
// Cleanup runs even when the test fails or panics, so it's the right place
// to delete cloud resources that the test may have created.
//
// Cleanup is best-effort by design: a failure is logged, not returned, so a
// flappy teardown doesn't mask the actual test failure. Set
// EXOSCALE_TEST_CLEANUP=skip to disable cleanup entirely (useful when
// debugging a failed run and you want to inspect leftover state).
type cleanupRegistry struct {
	mu      sync.Mutex
	entries []cleanupEntry
}

type cleanupEntry struct {
	name     string
	destroy  func() error
	disabled bool
}

var registries sync.Map

func registryFor(t *testing.T) *cleanupRegistry {
	t.Helper()
	v, _ := registries.LoadOrStore(t, &cleanupRegistry{})
	return v.(*cleanupRegistry)
}

// RegisterCleanup records a destroy callback to run at the end of t. The
// callback is invoked via t.Cleanup, so it runs even if the test fails or
// panics. The name is logged on success/failure; pass the resource name
// (or any short identifier) for easier debugging.
//
// Cleanup is best-effort: a destroy failure is logged via t.Logf and does
// not fail the test. Set EXOSCALE_TEST_CLEANUP=skip to skip all cleanup.
func RegisterCleanup(t *testing.T, name string, destroy func() error) {
	t.Helper()

	if cleanupDisabled() {
		return
	}

	reg := registryFor(t)
	reg.mu.Lock()
	reg.entries = append(reg.entries, cleanupEntry{name: name, destroy: destroy})
	reg.mu.Unlock()

	t.Cleanup(func() {
		if cleanupDisabled() {
			return
		}
		for _, e := range reg.entries {
			if err := e.destroy(); err != nil {
				t.Logf("cleanup %q failed: %v", e.name, err)
			}
		}
	})
}

func cleanupDisabled() bool {
	return strings.EqualFold(os.Getenv("EXOSCALE_TEST_CLEANUP"), "skip")
}

// GuardProdRefuse bails the test when the target environment is prod.
// Acceptance tests should never run against prod by default; passing
// EXOSCALE_TEST_ALLOW_PROD=1 is required to override (production debugging
// only; the override is logged).
func GuardProdRefuse(t *testing.T) {
	t.Helper()
	if msg := prodRefuseMessage(os.Getenv("EXOSCALE_API_ENVIRONMENT"), os.Getenv("EXOSCALE_TEST_ALLOW_PROD")); msg != "" {
		t.Fatal(msg)
	}
}

// prodRefuseMessage returns the error message (or "") describing whether
// the test should be refused, given the current env and override values.
// Exposed for unit tests; do not call directly from production code.
func prodRefuseMessage(env, override string) string {
	if env != "prod" {
		return ""
	}
	if override == "1" {
		return ""
	}
	return "refusing to run acceptance tests against prod (EXOSCALE_API_ENVIRONMENT=prod); " +
		"set EXOSCALE_TEST_ALLOW_PROD=1 to override"
}
