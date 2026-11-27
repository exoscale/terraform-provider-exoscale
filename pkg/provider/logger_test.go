package provider_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"log/slog"
	"os"
	"testing"

	"github.com/exoscale/terraform-provider-exoscale/pkg/provider"

	"github.com/hashicorp/terraform-plugin-log/tflogtest"
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
	os.Unsetenv("TF_LOG")
}

func TestLogHandlerFormating(t *testing.T) {
	os.Setenv("TF_LOG", "TRACE")
	var output bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &output)
	logger := slog.New(&provider.TflogHandler{})

	cases := []struct {
		InputGroup      string
		InputShareAttrs []any
		InputMsg        string
		InputAttrs      []any
		Output          map[string]string
	}{
		{
			InputMsg:   "msg1",
			InputAttrs: []any{"key1", "value1"},
			Output: map[string]string{
				"@message": "msg1",
				"key1":     "value1",
			},
		},
		{
			InputGroup: "group2",
			InputMsg:   "msg2",
			InputAttrs: []any{"key2", "value2"},
			Output: map[string]string{
				"@message":    "msg2",
				"group2.key2": "value2",
			},
		},
		{
			InputGroup:      "group3",
			InputMsg:        "msg3",
			InputShareAttrs: []any{"shared_key", "shared_value"},
			InputAttrs:      []any{"key3", "value3"},
			Output: map[string]string{
				"@message":          "msg3",
				"group3.shared_key": "shared_value",
				"group3.key3":       "value3",
			},
		},
	}
	for _, cs := range cases {
		logger := logger
		if cs.InputGroup != "" {
			logger = logger.WithGroup(cs.InputGroup)
		}
		if len(cs.InputShareAttrs) > 0 {
			logger = logger.With(cs.InputShareAttrs...)
		}
		logger.DebugContext(ctx, cs.InputMsg, cs.InputAttrs...)
		line := map[string]string{}
		if err := json.Unmarshal(output.Bytes(), &line); err != nil {
			t.Fatal(err)
		}
		for key, val := range cs.Output {
			if v, ok := line[key]; !ok || val != v {
				t.Errorf("%s=%s not in log output: %v", key, val, line)
			}
		}
		output.Reset()
	}
	os.Unsetenv("TF_LOG")
}
