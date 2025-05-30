package http

import (
	"context"
	"strconv"
)

type Response struct {
	Data any `json:"data"`
}

func ReqID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(reqIDKey).(ctxKey)
	return string(v), ok
}

func Locale(ctx context.Context, def ...string) string {
	if v, ok := ctx.Value(reqLocale).(ctxKey); ok {
		return string(v)
	}

	if len(def) > 0 {
		return def[0]
	}

	return "en"
}

func getDefaultNum[T any](value string, def T) T {
	switch any(def).(type) {
	case int, int8, int16, int32, int64:
		i, err := strconv.Atoi(value)
		if err != nil {
			return def
		}
		return any(i).(T)
	case float32, float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return def
		}
		return any(f).(T)
	default:
		return def
	}
}
