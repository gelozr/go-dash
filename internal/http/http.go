package http

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type Response struct {
	Data any `json:"data"`
}

func ReqID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(reqIDCtxKey).(string)
	return v, ok
}

func Locale(ctx context.Context, def ...string) string {
	if v, ok := ctx.Value(reqLocaleCtxKey).(string); ok {
		return v
	}

	if len(def) > 0 {
		return def[0]
	}

	return "en"
}

func UserIDFromCtx(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(userIDCtxKey).(uuid.UUID)
	return v, ok
}

func getDefaultInt(value string, def int) int {
	n, err := strconv.Atoi(value)
	if err == nil {
		return n
	}

	// Atoi can report overflow before checking the remaining digits.
	if errors.Is(err, strconv.ErrRange) {
		digits := strings.TrimLeft(value, "+-")
		if strings.TrimLeft(digits, "0123456789") == "" {
			return n
		}
	}
	return def
}
