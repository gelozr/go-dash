package http

import (
	"math"
	"strings"
	"testing"
)

func TestGetDefaultInt(t *testing.T) {
	large := strings.Repeat("9", 30)
	for _, tc := range []struct {
		name  string
		value string
		want  int
	}{
		{"valid", "23", 23},
		{"empty", "", 10},
		{"too large", large, math.MaxInt},
		{"positive sign", "+" + large, math.MaxInt},
		{"too small", "-" + large, math.MinInt},
		{"invalid suffix after overflow", large + "x", 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := getDefaultInt(tc.value, 10); got != tc.want {
				t.Errorf("getDefaultInt(%q, 10) = %d, want %d", tc.value, got, tc.want)
			}
		})
	}
}
