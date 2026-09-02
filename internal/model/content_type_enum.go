package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const nullStr = "null"

const (
	ContentTypeMovie     ContentType = "movie"
	ContentTypeTvShow    ContentType = "tv_show"
	ContentTypeMusic     ContentType = "music"
	ContentTypeEbook     ContentType = "ebook"
	ContentTypeComic     ContentType = "comic"
	ContentTypeAudiobook ContentType = "audiobook"
	ContentTypeGame      ContentType = "game"
	ContentTypeSoftware  ContentType = "software"
	ContentTypeOther     ContentType = "other"
	ContentTypeUnknown   ContentType = "unknown"
	ContentTypeAdult     ContentType = "adult"
)

var ErrInvalidContentType = fmt.Errorf("not a valid ContentType, try [%s]", strings.Join(ContentTypeNames(), ", "))

var _contentTypeValue = map[string]ContentType{
	"movie":     ContentTypeMovie,
	"tv_show":   ContentTypeTvShow,
	"music":     ContentTypeMusic,
	"ebook":     ContentTypeEbook,
	"comic":     ContentTypeComic,
	"audiobook": ContentTypeAudiobook,
	"game":      ContentTypeGame,
	"software":  ContentTypeSoftware,
	"other":     ContentTypeOther,
	"unknown":   ContentTypeUnknown,
	"adult":     ContentTypeAdult,
}

var _contentTypeValues = []ContentType{
	ContentTypeMovie,
	ContentTypeTvShow,
	ContentTypeMusic,
	ContentTypeEbook,
	ContentTypeComic,
	ContentTypeAudiobook,
	ContentTypeGame,
	ContentTypeSoftware,
	ContentTypeOther,
	ContentTypeUnknown,
	ContentTypeAdult,
}

// ContentTypeNames returns a list of possible string values of ContentType.
func ContentTypeNames() []string {
	names := make([]string, len(_contentTypeValues))
	for i, v := range _contentTypeValues {
		names[i] = string(v)
	}

	return names
}

// ContentTypeValues returns a list of the values for ContentType
func ContentTypeValues() []ContentType {
	return append([]ContentType(nil), _contentTypeValues...)
}

// String implements the Stringer interface.
func (x ContentType) String() string {
	return string(x)
}

// IsValid provides a quick way to determine if the typed value is
// part of the allowed enumerated values
func (x ContentType) IsValid() bool {
	_, err := ParseContentType(string(x))
	return err == nil
}

// ParseContentType attempts to convert a string to a ContentType.
func ParseContentType(name string) (ContentType, error) {
	if x, ok := _contentTypeValue[name]; ok {
		return x, nil
	}
	// Case insensitive parse, do a separate lookup to prevent unnecessary cost of lowercasing a string if we don't need to.
	if x, ok := _contentTypeValue[strings.ToLower(name)]; ok {
		return x, nil
	}

	return ContentType(""), fmt.Errorf("%s is %w", name, ErrInvalidContentType)
}

// MarshalText implements the text marshaller method.
func (x ContentType) MarshalText() ([]byte, error) {
	return []byte(string(x)), nil
}

// UnmarshalText implements the text unmarshaller method.
func (x *ContentType) UnmarshalText(text []byte) error {
	tmp, err := ParseContentType(string(text))
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
func (x *ContentType) AppendText(b []byte) ([]byte, error) {
	return append(b, x.String()...), nil
}

var errContentTypeNilPtr = errors.New("value pointer is nil") // one per type for package clashes

// Scan implements the Scanner interface.
func (x *ContentType) Scan(value interface{}) (err error) {
	if value == nil {
		*x = ContentType("")
		return
	}

	// A wider range of scannable types.
	// driver.Value values at the top of the list for expediency
	switch v := value.(type) {
	case string:
		*x, err = ParseContentType(v)
	case []byte:
		*x, err = ParseContentType(string(v))
	case ContentType:
		*x = v
	case *ContentType:
		if v == nil {
			return errContentTypeNilPtr
		}

		*x = *v
	case *string:
		if v == nil {
			return errContentTypeNilPtr
		}

		*x, err = ParseContentType(*v)
	default:
		return errors.New("invalid type for ContentType")
	}

	return
}

// Value implements the driver Valuer interface.
func (x ContentType) Value() (driver.Value, error) {
	return x.String(), nil
}

type NullContentType struct {
	ContentType ContentType
	Valid       bool
}

func NewNullContentType(val interface{}) (x NullContentType) {
	err := x.Scan(val) // yes, we ignore this error, it will just be an invalid value.
	_ = err            // make any errcheck linters happy

	return
}

// Scan implements the Scanner interface.
func (x *NullContentType) Scan(value interface{}) (err error) {
	if value == nil {
		x.ContentType, x.Valid = ContentType(""), false
		return
	}

	err = x.ContentType.Scan(value)
	x.Valid = (err == nil)

	return
}

// Value implements the driver Valuer interface.
func (x NullContentType) Value() (driver.Value, error) {
	if !x.Valid {
		//nolint:nilnil
		return nil, nil
	}

	return x.ContentType.String(), nil
}

// MarshalJSON correctly serializes a NullContentType to JSON.
func (x NullContentType) MarshalJSON() ([]byte, error) {
	if x.Valid {
		return json.Marshal(x.ContentType)
	}

	return []byte(nullStr), nil
}

// UnmarshalJSON correctly deserializes a NullContentType from JSON.
func (x *NullContentType) UnmarshalJSON(b []byte) error {
	var v interface{}

	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	return x.Scan(v)
}

// MarshalGQL correctly serializes a NullContentType to GraphQL.
func (x NullContentType) MarshalGQL(w io.Writer) {
	bytes, err := json.Marshal(x)
	if err == nil {
		_, _ = w.Write(bytes)
	}
}

// UnmarshalGQL correctly deserializes a NullContentType from GraphQL.
func (x *NullContentType) UnmarshalGQL(v any) error {
	if v == nil {
		return nil
	}

	str, ok := v.(string)
	if !ok {
		return errors.New("value is not a string")
	}

	if str == nullStr {
		return nil
	}

	return x.UnmarshalJSON([]byte("\"" + str + "\""))
}
