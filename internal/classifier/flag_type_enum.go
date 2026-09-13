package classifier

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
)

const (
	FlagTypeBool            FlagType = "bool"
	FlagTypeString          FlagType = "string"
	FlagTypeInt             FlagType = "int"
	FlagTypeStringList      FlagType = "string_list"
	FlagTypeContentTypeList FlagType = "content_type_list"
)

var ErrInvalidFlagType = fmt.Errorf("not a valid FlagType, try [%s]", strings.Join(FlagTypeNames(), ", "))

var _flagTypeValue = map[string]FlagType{
	"bool":              FlagTypeBool,
	"string":            FlagTypeString,
	"int":               FlagTypeInt,
	"string_list":       FlagTypeStringList,
	"content_type_list": FlagTypeContentTypeList,
}

var _flagTypeValues = []FlagType{
	FlagTypeBool,
	FlagTypeString,
	FlagTypeInt,
	FlagTypeStringList,
	FlagTypeContentTypeList,
}

// FlagTypeNames returns a list of possible string values of FlagType.
func FlagTypeNames() []string {
	names := make([]string, len(_flagTypeValues))
	for i, v := range _flagTypeValues {
		names[i] = string(v)
	}

	return names
}

// FlagTypeValues returns a list of the values for FlagType
func FlagTypeValues() []FlagType {
	return append([]FlagType(nil), _flagTypeValues...)
}

// String implements the Stringer interface.
func (x FlagType) String() string {
	return string(x)
}

// IsValid provides a quick way to determine if the typed value is
// part of the allowed enumerated values
func (x FlagType) IsValid() bool {
	_, err := ParseFlagType(string(x))
	return err == nil
}

// ParseFlagType attempts to convert a string to a FlagType.
func ParseFlagType(name string) (FlagType, error) {
	if x, ok := _flagTypeValue[name]; ok {
		return x, nil
	}
	// Case insensitive parse, do a separate lookup to prevent unnecessary cost of lowercasing a string if we don't need to.
	if x, ok := _flagTypeValue[strings.ToLower(name)]; ok {
		return x, nil
	}

	return FlagType(""), fmt.Errorf("%s is %w", name, ErrInvalidFlagType)
}

// MarshalText implements the text marshaller method.
func (x FlagType) MarshalText() ([]byte, error) {
	return []byte(string(x)), nil
}

// UnmarshalText implements the text unmarshaller method.
func (x *FlagType) UnmarshalText(text []byte) error {
	tmp, err := ParseFlagType(string(text))
	if err != nil {
		return err
	}

	*x = tmp

	return nil
}

// AppendText appends the textual representation of itself to the end of b
// (allocating a larger slice if necessary) and returns the updated slice.
//
// Implementations must not retain b, nor mutate any bytes within b[:len(b)].
func (x *FlagType) AppendText(b []byte) ([]byte, error) {
	return append(b, x.String()...), nil
}

var errFlagTypeNilPtr = errors.New("value pointer is nil") // one per type for package clashes

// Scan implements the Scanner interface.
func (x *FlagType) Scan(value interface{}) (err error) {
	if value == nil {
		*x = FlagType("")
		return
	}

	// A wider range of scannable types.
	// driver.Value values at the top of the list for expediency
	switch v := value.(type) {
	case string:
		*x, err = ParseFlagType(v)
	case []byte:
		*x, err = ParseFlagType(string(v))
	case FlagType:
		*x = v
	case *FlagType:
		if v == nil {
			return errFlagTypeNilPtr
		}

		*x = *v
	case *string:
		if v == nil {
			return errFlagTypeNilPtr
		}

		*x, err = ParseFlagType(*v)
	default:
		return errors.New("invalid type for FlagType")
	}

	return
}

// Value implements the driver Valuer interface.
func (x FlagType) Value() (driver.Value, error) {
	return x.String(), nil
}
