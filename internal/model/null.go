package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// NullString - nullable string
type NullString struct {
	String string
	Valid  bool // Valid is true if String is not NULL
}

func NewNullString(s string) NullString {
	return NullString{
		String: s,
		Valid:  true,
	}
}

func NewNullStringFromPtr(s *string) NullString {
	if s == nil {
		return NullString{}
	}

	return NewNullString(*s)
}

func (n NullString) Ptr() *string {
	if !n.Valid {
		return nil
	}

	return &n.String
}

func (n *NullString) Scan(value any) error {
	v, ok := value.(string)
	if !ok {
		n.Valid = false
	} else {
		n.String = v
		n.Valid = true
	}

	return nil
}

func (n NullString) Value() (driver.Value, error) {
	if !n.Valid {
		//nolint:nilnil
		return nil, nil
	}

	return n.String, nil
}

func (n NullString) MarshalJSON() ([]byte, error) {
	const nullStr = "null"

	if n.Valid {
		return json.Marshal(n.String)
	}

	return []byte(nullStr), nil
}

func (n *NullString) UnmarshalJSON(b []byte) error {
	var x any

	err := json.Unmarshal(b, &x)
	if err != nil {
		return err
	}

	err = n.Scan(x)

	return err
}

func (n *NullString) UnmarshalGQL(v any) error {
	if v == nil {
		n.Valid = false
		return nil
	}

	switch v := v.(type) {
	case string:
		n.String = v
	case []byte:
		n.String = string(v)
	default:
		return errors.New("wrong type")
	}

	n.Valid = true

	return nil
}

func (n NullString) MarshalGQL(w io.Writer) {
	if !n.Valid {
		_, _ = w.Write([]byte("null"))
		return
	}

	_, _ = fmt.Fprintf(w, "%q", n.String)
}

// NullBool - nullable bool
type NullBool struct {
	Bool  bool
	Valid bool // Valid is true if Bool is not NULL
}

func NewNullBool(b bool) NullBool {
	return NullBool{
		Bool:  b,
		Valid: true,
	}
}

func (n NullBool) Ptr() *bool {
	if !n.Valid {
		return nil
	}

	return &n.Bool
}

func (n *NullBool) Scan(value any) error {
	v, ok := value.(bool)
	if !ok {
		n.Valid = false
	} else {
		n.Bool = v
		n.Valid = true
	}

	return nil
}

func (n NullBool) Value() (driver.Value, error) {
	if !n.Valid {
		//nolint:nilnil
		return nil, nil
	}

	return n.Bool, nil
}

func (n *NullBool) UnmarshalGQL(v any) error {
	if v == nil {
		n.Valid = false
		return nil
	}

	switch v := v.(type) {
	case bool:
		n.Bool = v
	case string:
		_, err := fmt.Sscanf(v, "%t", &n.Bool)
		if err != nil {
			return err
		}
	default:
		return errors.New("wrong type")
	}

	n.Valid = true

	return nil
}

func (n NullBool) MarshalGQL(w io.Writer) {
	if !n.Valid {
		_, _ = w.Write([]byte("null"))
		return
	}

	_, _ = fmt.Fprintf(w, "%t", n.Bool)
}

// NullFloat32 - nullable float32
type NullFloat32 struct {
	Float32 float32
	Valid   bool // Valid is true if Float32 is not NULL
}

func NewNullFloat32(f float32) NullFloat32 {
	return NullFloat32{
		Float32: f,
		Valid:   true,
	}
}

func (n NullFloat32) Ptr() *float32 {
	if !n.Valid {
		return nil
	}

	return &n.Float32
}

func (n *NullFloat32) Scan(value any) error {
	v, ok := value.(float64)
	if !ok {
		n.Valid = false
	} else {
		n.Float32 = float32(v)
		n.Valid = true
	}

	return nil
}

func (n NullFloat32) Value() (driver.Value, error) {
	if !n.Valid {
		//nolint:nilnil
		return nil, nil
	}

	return n.Float32, nil
}

func (n *NullFloat32) UnmarshalGQL(v any) error {
	if v == nil {
		n.Valid = false
		return nil
	}

	switch v := v.(type) {
	case int:
		n.Float32 = float32(v)
	case int32:
		n.Float32 = float32(v)
	case int64:
		n.Float32 = float32(v)
	case uint:
		n.Float32 = float32(v)
	case uint32:
		n.Float32 = float32(v)
	case uint64:
		n.Float32 = float32(v)
	case float32:
		n.Float32 = v
	case float64:
		n.Float32 = float32(v)
	case string:
		_, err := fmt.Sscanf(v, "%f", &n.Float32)
		if err != nil {
			return err
		}
	default:
		return errors.New("wrong type")
	}

	n.Valid = true

	return nil
}

func (n NullFloat32) MarshalGQL(w io.Writer) {
	if !n.Valid {
		_, _ = w.Write([]byte("null"))
		return
	}

	_, _ = fmt.Fprintf(w, "%f", n.Float32)
}

// NullFloat64 - nullable float64
type NullFloat64 struct {
	Float64 float64
	Valid   bool // Valid is true if Float64 is not NULL
}

func (n *NullFloat64) Scan(value any) error {
	v, ok := value.(float64)
	if !ok {
		n.Valid = false
	} else {
		n.Float64 = v
		n.Valid = true
	}

	return nil
}

func (n NullFloat64) Value() (driver.Value, error) {
	if !n.Valid {
		//nolint:nilnil
		return nil, nil
	}

	return n.Float64, nil
}

func (n *NullFloat64) UnmarshalGQL(v any) error {
	if v == nil {
		n.Valid = false
		return nil
	}

	switch v := v.(type) {
	case int:
		n.Float64 = float64(v)
	case int32:
		n.Float64 = float64(v)
	case int64:
		n.Float64 = float64(v)
	case uint:
		n.Float64 = float64(v)
	case uint32:
		n.Float64 = float64(v)
	case uint64:
		n.Float64 = float64(v)
	case float32:
		n.Float64 = float64(v)
	case float64:
		n.Float64 = v
	case string:
		_, err := fmt.Sscanf(v, "%f", &n.Float64)
		if err != nil {
			return err
		}
	default:
		return errors.New("wrong type")
	}

	n.Valid = true

	return nil
}

func (n NullFloat64) MarshalGQL(w io.Writer) {
	if !n.Valid {
		_, _ = w.Write([]byte("null"))
		return
	}

	_, _ = fmt.Fprintf(w, "%f", n.Float64)
}

// NullUint - nullable uint
type NullUint struct {
	Uint  uint
	Valid bool // Valid is true if Uint is not NULL
}

func NewNullUint(n uint) NullUint {
	return NullUint{
		Uint:  n,
		Valid: true,
	}
}

func (n *NullUint) Scan(value any) error {
	v, ok := value.(int64)
	if !ok {
		n.Valid = false
	} else {
		n.Uint = uint(v)
		n.Valid = true
	}

	return nil
}

func (n NullUint) Value() (driver.Value, error) {
	if !n.Valid {
		//nolint:nilnil
		return nil, nil
	}

	return n.Uint, nil
}

func (n *NullUint) UnmarshalGQL(v any) error {
	if v == nil {
		n.Valid = false
		return nil
	}

	switch v := v.(type) {
	case int:
		n.Uint = uint(v)
	case int32:
		n.Uint = uint(v)
	case int64:
		n.Uint = uint(v)
	case uint:
		n.Uint = v
	case uint32:
		n.Uint = uint(v)
	case uint64:
		n.Uint = uint(v)
	case float32:
		n.Uint = uint(v)
	case float64:
		n.Uint = uint(v)
	case string:
		_, err := fmt.Sscanf(v, "%d", &n.Uint)
		if err != nil {
			return err
		}
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return err
		}

		n.Uint = uint(i)
	default:
		return errors.New("wrong type")
	}

	n.Valid = true

	return nil
}

func (n NullUint) MarshalGQL(w io.Writer) {
	if !n.Valid {
		_, _ = w.Write([]byte("null"))
		return
	}

	_, _ = fmt.Fprintf(w, "%d", n.Uint)
}
