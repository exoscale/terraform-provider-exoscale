package sks

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	egoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/egoscale/v3/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendAddon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		initial  []string
		addon    string
		expected []string
	}{
		{"add to empty", []string{}, "new-addon", []string{"new-addon"}},
		{"add to existing", []string{"a", "b"}, "c", []string{"a", "b", "c"}},
		{"add duplicate (allowed in slice)", []string{"a", "b"}, "a", []string{"a", "b", "a"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, appendAddon(tt.initial, tt.addon))
		})
	}
}

func TestRemoveAddon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		initial  []string
		addon    string
		expected []string
	}{
		{"remove from empty", []string{}, "a", []string{}},
		{"remove existing", []string{"a", "b", "c"}, "b", []string{"a", "c"}},
		{"remove non-existent", []string{"a", "b"}, "c", []string{"a", "b"}},
		{"remove last", []string{"a"}, "a", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, removeAddon(tt.initial, tt.addon))
		})
	}
}

type fakeSKSVersionsTransport struct {
	versions []string
}

func (t *fakeSKSVersionsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := json.Marshal(map[string]any{
		"sks-cluster-versions": t.versions,
	})
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     make(http.Header),
	}, nil
}

func TestResolveSKSClusterVersion(t *testing.T) {
	t.Parallel()

	fakeClusterVersions := []string{"1.36.1", "1.35.2", "1.34.5"}

	client, err := egoscale.NewClient(
		credentials.NewStaticCredentials("foo", "bar"),
		egoscale.ClientOptWithHTTPClient(&http.Client{
			Transport: &fakeSKSVersionsTransport{fakeClusterVersions},
		}),
	)
	if err != nil {
		t.Fatalf("failed to create egoscale dummy client: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name            string
		inputVersion    string
		expectedVersion string
		errorMsg        string
	}{
		{
			name:            "major.minor resolves to corresponding major.minor.patch",
			inputVersion:    strings.Join(strings.Split(fakeClusterVersions[1], ".")[:2], "."),
			expectedVersion: fakeClusterVersions[1],
		},
		{
			name:            "empty version resolves to latest version",
			inputVersion:    "",
			expectedVersion: fakeClusterVersions[0],
		},
		{
			name:            "major.minor.patch is returned as-is",
			inputVersion:    fakeClusterVersions[1],
			expectedVersion: fakeClusterVersions[1],
		},
		{
			name:         "unsupported major.minor throws error",
			inputVersion: "1.2",
			errorMsg:     "the SKS cluster version 1.2 is not supported",
		},
		{
			name:         "version fits no format",
			inputVersion: "foo",
			errorMsg:     "error resolving the provided SKS cluster version: foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolveSKSClusterVersion(ctx, client, tt.inputVersion)
			assert.Equal(t, tt.expectedVersion, result)
			if tt.errorMsg != "" {
				assert.ErrorContains(t, err, tt.errorMsg)
			}
		})
	}
}

func TestSKSNodepoolMIGProfiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		family    string
		profile   string
		expected  *egoscale.NvidiaMigProfiles
		expectErr bool
	}{
		{
			name:     "a30 family",
			family:   string(egoscale.InstanceTypeFamilyGpua30),
			profile:  "2g.12gb",
			expected: &egoscale.NvidiaMigProfiles{A3024gb: egoscale.NvidiaMigProfileA3024gb("2g.12gb")},
		},
		{
			name:     "rtxpro6000 family",
			family:   string(egoscale.InstanceTypeFamilyGpurtx6000pro),
			profile:  "1g.24gb+me.all",
			expected: &egoscale.NvidiaMigProfiles{Rtxpro600096gb: egoscale.NvidiaMigProfileRtxpro600096gb("1g.24gb+me.all")},
		},
		{
			name:      "non-GPU family",
			family:    string(egoscale.InstanceTypeFamilyStandard),
			profile:   "2g.12gb",
			expectErr: true,
		},
		{
			name:      "non-MIG GPU family",
			family:    string(egoscale.InstanceTypeFamilyGpua5000),
			profile:   "2g.12gb",
			expectErr: true,
		},
		{
			name:      "invalid profile for family",
			family:    string(egoscale.InstanceTypeFamilyGpua30),
			profile:   "1g.24gb", // an rtxpro6000 value
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sksNodepoolMIGProfiles(tt.family, tt.profile)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestSKSNodepoolMIGProfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *egoscale.NvidiaMigProfiles
		expected string
	}{
		{
			name:     "nil",
			input:    nil,
			expected: "",
		},
		{
			name:     "empty struct",
			input:    &egoscale.NvidiaMigProfiles{},
			expected: "",
		},
		{
			name:     "a30 set",
			input:    &egoscale.NvidiaMigProfiles{A3024gb: egoscale.NvidiaMigProfileA3024gb("4g.24gb")},
			expected: "4g.24gb",
		},
		{
			name:     "rtxpro6000 set",
			input:    &egoscale.NvidiaMigProfiles{Rtxpro600096gb: egoscale.NvidiaMigProfileRtxpro600096gb("2g.48gb+gfx")},
			expected: "2g.48gb+gfx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, sksNodepoolMIGProfile(tt.input))
		})
	}
}

func TestSKSNodepoolMIGProfilesRoundTrip(t *testing.T) {
	t.Parallel()

	profiles, err := sksNodepoolMIGProfiles(string(egoscale.InstanceTypeFamilyGpurtx6000pro), "2g.48gb-me")
	require.NoError(t, err)
	require.Equal(t, "2g.48gb-me", sksNodepoolMIGProfile(profiles))
}

func TestParseSKSNodepoolTaintV3(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  *egoscale.SKSNodepoolTaint
		expectErr bool
	}{
		{
			name:     "valid",
			input:    "test:NoSchedule",
			expected: &egoscale.SKSNodepoolTaint{Value: "test", Effect: "NoSchedule"},
		},
		{
			name:      "missing effect",
			input:     "test",
			expectErr: true,
		},
		{
			name:      "empty value",
			input:     ":NoSchedule",
			expectErr: true,
		},
		{
			name:      "empty effect",
			input:     "test:",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSKSNodepoolTaintV3(tt.input)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestSKSNodepoolInstanceTypeFamily(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "standard", sksNodepoolInstanceTypeFamily("standard.medium"))
	assert.Equal(t, "gpua30", sksNodepoolInstanceTypeFamily("GPUA30.huge"))
}
