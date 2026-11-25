package provider_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/exoscale/terraform-provider-exoscale/pkg/provider"
)

func TestLogHandlerLevels(t *testing.T) {
	cases := []struct {
		EnvLevel  string
		WantLevel slog.Level
		Enabled   bool
	}{
		{
			EnvLevel:  "",
			WantLevel: slog.LevelInfo,
			Enabled:   false,
		},
		{
			EnvLevel:  "INFO",
			WantLevel: slog.LevelDebug,
			Enabled:   false,
		},
		{
			EnvLevel:  "info",
			WantLevel: slog.LevelInfo,
			Enabled:   true,
		},
		{
			EnvLevel:  "DEBUG",
			WantLevel: slog.LevelError,
			Enabled:   true,
		},
	}

	h := provider.TflogHandler{}
	for _, cs := range cases {
		os.Setenv("TF_LOG", cs.EnvLevel)
		if h.Enabled(context.Background(), cs.WantLevel) != cs.Enabled {
			t.Errorf(
				"for TF_LOG=%s and slog.Level=%s expected handler.Enabled() to be %v",
				cs.EnvLevel,
				cs.WantLevel.String(),
				cs.Enabled,
			)
		}
	}
}
