package optional

import (
	"database/sql/driver"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestOptionalJSONMarshal(t *testing.T) {
	present := Of(false)
	missing := Optional[bool]{}
	explicitNull := FromPtr[bool](nil)
	for _, tc := range []struct {
		name  string
		input any
		want  string
	}{
		{"present value", present, `false`},
		{"present pointer", &present, `false`},
		{"missing value", missing, `null`},
		{"missing pointer", &missing, `null`},
		{"explicit null value", explicitNull, `null`},
		{"explicit null pointer", &explicitNull, `null`},
		{"nil pointer", (*Optional[bool])(nil), `null`},
		{"map value", map[string]Optional[bool]{"active": present}, `{"active":false}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.input)
			if err != nil || string(got) != tc.want {
				t.Fatalf("json.Marshal() = %s, %v; want %s, nil", got, err, tc.want)
			}
		})
	}
}

func TestOptionalValueConvertsSQLParameters(t *testing.T) {
	id := uuid.MustParse("76bde1d8-d1a9-4fc5-8359-ae1f601846d3")
	for _, tc := range []struct {
		name  string
		input driver.Valuer
		want  driver.Value
	}{
		{"int", Of(42), int64(42)},
		{"UUID", Of(id), id.String()},
		{"null", FromPtr[int](nil), nil},
		{"missing", Optional[bool]{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := driver.DefaultParameterConverter.ConvertValue(tc.input)
			if err != nil || got != tc.want {
				t.Fatalf("SQL parameter = (%v, %v), want (%v, nil)", got, err, tc.want)
			}
		})
	}
}

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
