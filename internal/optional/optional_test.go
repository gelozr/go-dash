package optional

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestStringToUUIDWithDefinedString(t *testing.T) {
	type uuidString string

	want := uuid.MustParse("76bde1d8-d1a9-4fc5-8359-ae1f601846d3")
	got, err := StringToUUID(Of(uuidString(want.String())))
	if err != nil || got != Of(want) {
		t.Fatalf("StringToUUID() = %+v, %v; want %+v, nil", got, err, Of(want))
	}
}

func TestStringToTimeWithDefinedPointer(t *testing.T) {
	type timeStringPointer *string

	want := time.Date(2026, time.November, 27, 10, 0, 0, 0, time.UTC)
	raw := want.Format(time.RFC3339)
	got, err := StringToTime(Of(timeStringPointer(&raw)), time.RFC3339)
	if err != nil || !got.IsPresent || got.IsNull || !got.Val.Equal(want) {
		t.Fatalf("StringToTime() = %+v, %v; want %+v, nil", got, err, Of(want))
	}
}

func TestOptionalStringParsersRejectPresentNilPointer(t *testing.T) {
	t.Run("UUID", func(t *testing.T) {
		if _, err := StringToUUID(Of((*string)(nil))); err == nil {
			t.Fatal("StringToUUID() accepted a present nil pointer")
		}
	})
	t.Run("time", func(t *testing.T) {
		if _, err := StringToTime(Of((*string)(nil)), ""); err == nil {
			t.Fatal("StringToTime() accepted a present nil pointer")
		}
	})
}
