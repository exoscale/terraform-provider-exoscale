package provider

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var envLogLevelMap = map[string]slog.Level{
	"TRACE": slog.LevelDebug,
	"DEBUG": slog.LevelDebug,
	"INFO":  slog.LevelInfo,
	"WARN":  slog.LevelWarn,
	"ERROR": slog.LevelError,
}

// func envSlogLevel converts terraform log level to slog.Level
// Terraform log level is defined using environment variable which is what this function does as well.
func envSlogLevel() slog.Level {
	envLevel := strings.ToUpper(os.Getenv("TF_LOG"))
	level, ok := envLogLevelMap[envLevel]
	if !ok {
		return slog.LevelError + 1 // makes h.Enabled() always return false
	}

	return level
}

// groupOrAttrs holds either a group name or a list of slog.Attrs.
type groupOrAttrs struct {
	group string      // group name if non-empty
	attrs []slog.Attr // attrs if non-empty
}

// TflogHandler implements logs/slog.Handler interace and outputs logs using tflog library.
// Implementation is based on official slog [example].
//
// [example]: https://github.com/golang/example/blob/master/slog-handler-guide/README.md
type TflogHandler struct {
	goas []groupOrAttrs
}

func (*TflogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= envSlogLevel()
}

func (h *TflogHandler) Handle(ctx context.Context, record slog.Record) error {
	fields := map[string]any{}
	var prefix string

	for _, goa := range h.goas {
		if goa.group != "" {
			if prefix != "" {
				prefix = fmt.Sprintf("%s.%s", prefix, goa.group)
			} else {
				prefix = goa.group
			}
		} else {
			for _, attr := range goa.attrs {
				if prefix != "" {
					attr.Key = fmt.Sprintf("%s.%s", prefix, goa.group)
				}

				maps.Copy(fields, attrResolver(attr))
			}
		}
	}

	record.Attrs(func(attr slog.Attr) bool {
		attr.Value = attr.Value.Resolve()
		if prefix != "" {
			attr.Key = fmt.Sprintf("%s.%s", prefix, attr.Key)
		}
		maps.Copy(fields, attrResolver(attr))
		return true
	})

	switch {
	case record.Level < slog.LevelDebug:
		tflog.Trace(ctx, record.Message, fields)
	case record.Level >= slog.LevelDebug && record.Level < slog.LevelInfo:
		tflog.Debug(ctx, record.Message, fields)
	case record.Level >= slog.LevelInfo && record.Level < slog.LevelWarn:
		tflog.Info(ctx, record.Message, fields)
	case record.Level >= slog.LevelWarn && record.Level < slog.LevelError:
		tflog.Warn(ctx, record.Message, fields)
	case record.Level >= slog.LevelError:
		tflog.Error(ctx, record.Message, fields)
	}

	return nil
}

func (h *TflogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) < 1 {
		return h
	}

	return h.withGroupOrAttrs(groupOrAttrs{attrs: attrs})
}

func (h *TflogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	return h.withGroupOrAttrs(groupOrAttrs{group: name})
}

func attrResolver(attr slog.Attr) map[string]any {
	fields := map[string]any{}

	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) || attr.Key == "" { // silently ignore junk
		return fields
	}

	switch attr.Value.Kind() {
	case slog.KindGroup:
		group := attr.Value.Group()
		for _, a := range group {
			if a.Key != "" {
				a.Key = fmt.Sprintf("%s.%s", attr.Key, a.Key)
				maps.Copy(fields, attrResolver(a))
			}
		}
	default:
		fields[attr.Key] = attr.Value.String()
	}

	return fields
}

func (h *TflogHandler) withGroupOrAttrs(goa groupOrAttrs) *TflogHandler {
	h2 := *h
	h2.goas = make([]groupOrAttrs, len(h.goas)+1)
	copy(h2.goas, h.goas)
	h2.goas[len(h2.goas)-1] = goa
	return &h2
}
