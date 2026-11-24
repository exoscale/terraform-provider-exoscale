package provider

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type TflogHandler struct {
	prefix  string      // log key prefix built from group names when WithGroup is invoked
	commons []slog.Attr // default log attributes, added with WithAttrs
}

func (*TflogHandler) Enabled(context.Context, slog.Level) bool {
	return true // we will let tflog decide when to print logs
}

func (h *TflogHandler) Handle(ctx context.Context, record slog.Record) error {
	fields := map[string]any{}
	record.Attrs(func(attr slog.Attr) bool {
		attr.Value = attr.Value.Resolve()
		if attr.Equal(slog.Attr{}) || attr.Key == "" { // silently ignore junk
			return true
		}
		if h.prefix != "" {
			attr.Key = fmt.Sprintf("%s.%s", h.prefix, attr.Key)
		}
		fields = attrResolver(attr)
		return true
	})

	for _, attr := range h.commons {
		if attr.Equal(slog.Attr{}) || attr.Key == "" { // silently ignore junk
			continue
		}
		if h.prefix != "" {
			attr.Key = fmt.Sprintf("%s.%s", h.prefix, attr.Key)
		}
		maps.Copy(fields, attrResolver(attr))
	}

	switch record.Level {
	case slog.LevelDebug:
		tflog.Debug(ctx, record.Message, fields)
	case slog.LevelInfo:
		tflog.Info(ctx, record.Message, fields)
	case slog.LevelWarn:
		tflog.Warn(ctx, record.Message, fields)
	case slog.LevelError:
		tflog.Error(ctx, record.Message, fields)
	}

	return nil
}

func (h *TflogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) < 1 {
		return h
	}

	return &TflogHandler{
		prefix:  h.prefix,
		commons: attrs,
	}
}

func (h *TflogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	if h.prefix != "" {
		name = fmt.Sprintf("%s.%s", h.prefix, name)
	}

	return &TflogHandler{
		prefix:  name,
		commons: slices.Clone(h.commons),
	}
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
