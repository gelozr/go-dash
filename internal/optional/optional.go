package optional

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
)

// Optional carries three states: missing, explicitly null, or a value.
type Optional[T any] struct {
	Val       T
	IsPresent bool
	IsNull    bool
}

// UnmarshalJSON marks IsPresent=true if preset, otherwise false
// if explicitly set to null the marks IsNull=true
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.IsPresent = true
	if bytes.Equal(data, []byte("null")) {
		o.IsNull = true
		var zero T
		o.Val = zero
		return nil
	}
	o.IsNull = false
	return json.Unmarshal(data, &o.Val)
}

// MarshalJSON emits either "null" (missing or null) or the JSON of Val.
func (o *Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.IsPresent || o.IsNull {
		return []byte("null"), nil
	}
	return json.Marshal(o.Val)
}

// Value implements driver.Valuer for database writes.
func (o Optional[T]) Value() (driver.Value, error) {
	if o.IsNull {
		return nil, nil
	}

	v := any(o.Val)

	if u, ok := v.(driver.Valuer); ok {
		return u.Value()
	}

	if dv, ok := v.(driver.Value); ok {
		return dv, nil
	}

	switch tv := v.(type) {
	case int64, float64, bool, string, []byte:
		return tv, nil
	default:
		return nil, fmt.Errorf("optional: unsupported type %T for driver.Value", o.Val)
	}
}

// Scan implements sql.Scanner for database reads.
func (o *Optional[T]) Scan(src any) error {
	o.IsPresent = true
	if src == nil {
		o.IsNull = true
		var zero T
		o.Val = zero
		return nil
	}
	o.IsNull = false

	// use T if it implements scanner
	if scanner, ok := any(&o.Val).(sql.Scanner); ok {
		err := scanner.Scan(src)
		if err != nil {
			return fmt.Errorf("optional scan: %w", err)
		}
		return nil
	}

	// direct conversion
	if v, ok := src.(T); ok {
		o.Val = v
		return nil
	}

	// Handle boolean
	if _, ok := any(o.Val).(bool); ok {
		switch v := src.(type) {
		case int64:
			o.Val = any(v != 0).(T)
			return nil
			// case []byte:
			// 	o.Val = any(string(v) == "1").(T)
			// 	return nil
			// case string:
			// 	o.Val = any(v == "1").(T)
			// 	return nil
		}
	}

	// for drivers that return []byte for text columns
	if b, ok := src.([]byte); ok {
		switch any(o.Val).(type) {
		case string:
			o.Val = any(string(b)).(T)
			return nil
		case []byte:
			o.Val = any(b).(T)
			return nil
		}
	}

	return fmt.Errorf(
		"cannot scan %T into Optional[%s]",
		src, reflect.TypeOf(o.Val).Name(),
	)
}

func Of[T any](v T) Optional[T] {
	return Optional[T]{Val: v, IsPresent: true, IsNull: false}
}

func FromPtr[T any](v *T) Optional[T] {
	var (
		n      T
		isNull bool
	)

	if v != nil {
		n = *v
	} else {
		isNull = true
	}

	return Optional[T]{
		Val:       n,
		IsPresent: true,
		IsNull:    isNull,
	}
}

// Ptr returns nil when the value is missing or null, so a NULL column doesn't read back as the zero value.
func (o Optional[T]) Ptr() *T {
	if !o.IsPresent || o.IsNull {
		return nil
	}
	return &o.Val
}

func StringToUUID[T ~string | ~*string](o Optional[T]) (Optional[uuid.UUID], error) {
	if !o.IsPresent {
		return Optional[uuid.UUID]{}, nil
	}
	if o.IsNull {
		return Optional[uuid.UUID]{IsPresent: true, IsNull: true}, nil
	}

	var str string
	switch v := any(o.Val).(type) {
	case string:
		str = v
	case *string:
		str = *v
	}

	u, err := uuid.Parse(str)
	if err != nil {
		return Optional[uuid.UUID]{}, fmt.Errorf("cannot parse %q as UUID: %w", str, err)
	}
	return Optional[uuid.UUID]{Val: u, IsPresent: true}, nil
}

func StringToTime[T ~string | ~*string](o Optional[T], layout string) (Optional[time.Time], error) {
	if !o.IsPresent {
		return Optional[time.Time]{}, nil
	}
	if o.IsNull {
		return Optional[time.Time]{IsPresent: true, IsNull: true}, nil
	}

	var str string
	switch v := any(o.Val).(type) {
	case string:
		str = v
	case *string:
		str = *v
	}

	t, err := time.Parse(layout, str)
	if err != nil {
		return Optional[time.Time]{}, fmt.Errorf("cannot parse %q as Time: %w", str, err)
	}

	return Optional[time.Time]{Val: t, IsPresent: true}, nil
}
