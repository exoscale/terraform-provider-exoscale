package testutils

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

// shortSHA holds the resolved commit short SHA used as a name prefix in
// acceptance tests. Resolved once on first call to ShortSHA().
var (
	shortSHAOnce sync.Once
	shortSHAVal  string
)

// injectedSHA lets the build wire a value via -ldflags when running from a
// release tarball where .git is missing.
//
//	-X github.com/exoscale/terraform-provider-exoscale/pkg/testutils.injectedSHA=...
var injectedSHA string

// ShortSHA returns the first 8 characters of HEAD, used to namespace test
// resources so a leaked run is easy to identify in the cloud console.
// Falls back to injectedSHA (set via -ldflags for release tarballs) and to
// "local" when neither is available.
func ShortSHA() string {
	shortSHAOnce.Do(func() {
		shortSHAVal = resolveShortSHA()
	})
	return shortSHAVal
}

func resolveShortSHA() string {
	if injectedSHA != "" {
		return sanitizeSHA(injectedSHA)
	}

	cmd := exec.Command("git", "rev-parse", "--short=8", "HEAD")
	if dir, ok := gitRoot(); ok {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	if err == nil {
		return sanitizeSHA(strings.TrimSpace(string(out)))
	}
	return "local"
}

func gitRoot() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for i := 0; i < 12; i++ {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

func sanitizeSHA(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 8 {
		s = s[:8]
	}
	if !isHex(s) {
		return "local"
	}
	return s
}

func isHex(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.ParseUint(s, 16, 64)
	return err == nil
}

// ResourcePrefix is the namespace used for resources created by acceptance
// tests. Resources are named "<prefix>-<short-sha>-<rand>" so leftovers are
// attributable to the commit that produced them.
const ResourcePrefix = "test-terraform-exoscale"

// TestResourceName returns a unique resource name. Shape:
//
//	test-terraform-exoscale-<sha>-<rand>
//
// The sha lets a leaked resource be tied back to the commit that created it;
// the rand suffix avoids collisions when the same commit is run twice in
// parallel (e.g. CI matrix, two local invocations). Safe to call from
// package-level var initializers (no *testing.T required).
func TestResourceName() string {
	return testResourceNameWithSuffix("")
}

// TestResourceNameWithSuffix is like TestResourceName but inserts the
// supplied suffix before the random part, producing:
//
//	test-terraform-exoscale-<sha>-<suffix>-<rand>
//
// Useful for tests that filter the cloud console by resource kind (e.g.
// "role" or "api-key").
func TestResourceNameWithSuffix(suffix string) string {
	return testResourceNameWithSuffix(suffix)
}

func testResourceNameWithSuffix(suffix string) string {
	base := ResourcePrefix + "-" + ShortSHA()
	if suffix != "" {
		base += "-" + suffix
	}
	return base + "-" + strconv.Itoa(acctest.RandInt())
}
