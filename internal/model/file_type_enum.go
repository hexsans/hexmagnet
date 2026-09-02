package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	FileTypeArchive   FileType = "archive"
	FileTypeAudio     FileType = "audio"
	FileTypeData      FileType = "data"
	FileTypeDocument  FileType = "document"
	FileTypeImage     FileType = "image"
	FileTypeSoftware  FileType = "software"
	FileTypeSubtitles FileType = "subtitles"
	FileTypeVideo     FileType = "video"
)

var ErrInvalidFileType = fmt.Errorf("not a valid FileType, try [%s]", strings.Join(FileTypeNames(), ", "))

var _fileTypeValue = map[string]FileType{
	"archive":   FileTypeArchive,
	"audio":     FileTypeAudio,
	"data":      FileTypeData,
	"document":  FileTypeDocument,
	"image":     FileTypeImage,
	"software":  FileTypeSoftware,
	"subtitles": FileTypeSubtitles,
	"video":     FileTypeVideo,
}

var _fileTypeValues = []FileType{
	FileTypeArchive,
	FileTypeAudio,
	FileTypeData,
	FileTypeDocument,
	FileTypeImage,
	FileTypeSoftware,
	FileTypeSubtitles,
	FileTypeVideo,
}

// FileTypeNames returns a list of possible string values of FileType.
func FileTypeNames() []string {
	names := make([]string, len(_fileTypeValues))
	for i, v := range _fileTypeValues {
		names[i] = string(v)
	}

	return names
}

// FileTypeValues returns a list of the values for FileType
func FileTypeValues() []FileType {
	return append([]FileType(nil), _fileTypeValues...)
}

// String implements the Stringer interface.
func (x FileType) String() string {
	return string(x)
}

// IsValid provides a quick way to determine if the typed value is
// part of the allowed enumerated values
func (x FileType) IsValid() bool {
	_, err := ParseFileType(string(x))
	return err == nil
}

// ParseFileType attempts to convert a string to a FileType.
func ParseFileType(name string) (FileType, error) {
	if x, ok := _fileTypeValue[name]; ok {
		return x, nil
	}
	// Case insensitive parse, do a separate lookup to prevent unnecessary cost of lowercasing a string if we don't need to.
	if x, ok := _fileTypeValue[strings.ToLower(name)]; ok {
		return x, nil
	}

	return FileType(""), fmt.Errorf("%s is %w", name, ErrInvalidFileType)
}

// MarshalText implements the text marshaller method.
func (x FileType) MarshalText() ([]byte, error) {
	return []byte(string(x)), nil
}

// UnmarshalText implements the text unmarshaller method.
func (x *FileType) UnmarshalText(text []byte) error {
	tmp, err := ParseFileType(string(text))
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
func (x *FileType) AppendText(b []byte) ([]byte, error) {
	return append(b, x.String()...), nil
}

var errFileTypeNilPtr = errors.New("value pointer is nil") // one per type for package clashes

// Scan implements the Scanner interface.
func (x *FileType) Scan(value interface{}) (err error) {
	if value == nil {
		*x = FileType("")
		return
	}

	// A wider range of scannable types.
	// driver.Value values at the top of the list for expediency
	switch v := value.(type) {
	case string:
		*x, err = ParseFileType(v)
	case []byte:
		*x, err = ParseFileType(string(v))
	case FileType:
		*x = v
	case *FileType:
		if v == nil {
			return errFileTypeNilPtr
		}

		*x = *v
	case *string:
		if v == nil {
			return errFileTypeNilPtr
		}

		*x, err = ParseFileType(*v)
	default:
		return errors.New("invalid type for FileType")
	}

	return
}

// Value implements the driver Valuer interface.
func (x FileType) Value() (driver.Value, error) {
	return x.String(), nil
}

type NullFileType struct {
	FileType FileType
	Valid    bool
}

func NewNullFileType(val interface{}) (x NullFileType) {
	err := x.Scan(val) // yes, we ignore this error, it will just be an invalid value.
	_ = err            // make any errcheck linters happy

	return
}

// Scan implements the Scanner interface.
func (x *NullFileType) Scan(value interface{}) (err error) {
	if value == nil {
		x.FileType, x.Valid = FileType(""), false
		return
	}

	err = x.FileType.Scan(value)
	x.Valid = (err == nil)

	return
}

// Value implements the driver Valuer interface.
func (x NullFileType) Value() (driver.Value, error) {
	if !x.Valid {
		//nolint:nilnil
		return nil, nil
	}

	return x.FileType.String(), nil
}

// MarshalJSON correctly serializes a NullFileType to JSON.
func (x NullFileType) MarshalJSON() ([]byte, error) {
	if x.Valid {
		return json.Marshal(x.FileType)
	}

	return []byte(nullStr), nil
}

// UnmarshalJSON correctly deserializes a NullFileType from JSON.
func (x *NullFileType) UnmarshalJSON(b []byte) error {
	var v interface{}

	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	return x.Scan(v)
}

// MarshalGQL correctly serializes a NullFileType to GraphQL.
func (x NullFileType) MarshalGQL(w io.Writer) {
	bytes, err := json.Marshal(x)
	if err == nil {
		_, _ = w.Write(bytes)
	}
}

// UnmarshalGQL correctly deserializes a NullFileType from GraphQL.
func (x *NullFileType) UnmarshalGQL(v any) error {
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
