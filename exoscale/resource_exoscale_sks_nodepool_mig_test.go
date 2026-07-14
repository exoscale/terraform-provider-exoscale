package exoscale

import (
	"testing"

	"github.com/stretchr/testify/require"

	v3 "github.com/exoscale/egoscale/v3"
)

func TestSKSNodepoolMIGProfiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		family    string
		profile   string
		expected  *v3.NvidiaMigProfiles
		expectErr bool
	}{
		{
			name:     "a30 family",
			family:   string(v3.InstanceTypeFamilyGpua30),
			profile:  "2g.12gb",
			expected: &v3.NvidiaMigProfiles{A3024gb: v3.NvidiaMigProfileA3024gb("2g.12gb")},
		},
		{
			name:     "rtxpro6000 family",
			family:   string(v3.InstanceTypeFamilyGpurtx6000pro),
			profile:  "1g.24gb+me.all",
			expected: &v3.NvidiaMigProfiles{Rtxpro600096gb: v3.NvidiaMigProfileRtxpro600096gb("1g.24gb+me.all")},
		},
		{
			name:      "non-GPU family",
			family:    string(v3.InstanceTypeFamilyStandard),
			profile:   "2g.12gb",
			expectErr: true,
		},
		{
			name:      "non-MIG GPU family",
			family:    string(v3.InstanceTypeFamilyGpua5000),
			profile:   "2g.12gb",
			expectErr: true,
		},
		{
			name:      "invalid profile for family",
			family:    string(v3.InstanceTypeFamilyGpua30),
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
		input    *v3.NvidiaMigProfiles
		expected string
	}{
		{
			name:     "nil",
			input:    nil,
			expected: "",
		},
		{
			name:     "empty struct",
			input:    &v3.NvidiaMigProfiles{},
			expected: "",
		},
		{
			name:     "a30 set",
			input:    &v3.NvidiaMigProfiles{A3024gb: v3.NvidiaMigProfileA3024gb("4g.24gb")},
			expected: "4g.24gb",
		},
		{
			name:     "rtxpro6000 set",
			input:    &v3.NvidiaMigProfiles{Rtxpro600096gb: v3.NvidiaMigProfileRtxpro600096gb("2g.48gb+gfx")},
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

	profiles, err := sksNodepoolMIGProfiles(string(v3.InstanceTypeFamilyGpurtx6000pro), "2g.48gb-me")
	require.NoError(t, err)
	require.Equal(t, "2g.48gb-me", sksNodepoolMIGProfile(profiles))
}
