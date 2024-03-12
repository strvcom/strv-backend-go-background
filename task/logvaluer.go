package task

import "log/slog"

// LogValue implements slog.LogValuer.
func (t Type) LogValue() slog.Value {
	switch t {
	case TypeOneOff:
		return slog.StringValue("oneoff")
	case TypeLoop:
		return slog.StringValue("loop")
	default:
		return slog.StringValue("invalid")
	}
}

// LogValue implements slog.LogValuer.
func (definition Task) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("type", definition.Type),
		slog.Any("meta", definition.Meta),
	)
}

// LogValue implements slog.LogValuer.
func (meta Metadata) LogValue() slog.Value {
	values := make([]slog.Attr, 0, len(meta))
	for key, value := range meta {
		values = append(values, slog.String(key, value))
	}

	return slog.GroupValue(values...)
}
