package model

import (
	"errors"
	"fmt"
	"io"
)

type Year uint16

func ParseYear(s string) (Year, error) {
	var y Year

	_, err := fmt.Sscanf(s, "%d", &y)
	if err != nil {
		return 0, err
	}

	return y, nil
}

func (y Year) String() string {
	return fmt.Sprintf("%d", y)
}

func (y Year) IsNil() bool {
	return y == 0
}

func (y *Year) Scan(src any) error {
	switch src := src.(type) {
	case nil:
		*y = 0
	case int:
		*y = Year(src)
	case int32:
		*y = Year(src)
	case int64:
		*y = Year(src)
	case uint:
		*y = Year(src)
	case uint32:
		*y = Year(src)
	case uint64:
		*y = Year(src)
	case float32:
		*y = Year(src)
	case float64:
		*y = Year(src)
	case string:
		_, err := fmt.Sscanf(src, "%d", y)
		if err != nil {
			return err
		}
	default:
		return errors.New("wrong type")
	}

	return nil
}

func (y Year) Value() (any, error) {
	if y.IsNil() {
		//nolint:nilnil
		return nil, nil
	}

	return int(y), nil
}

func (y Year) MarshalGQL(w io.Writer) {
	if y.IsNil() {
		_, _ = fmt.Fprintf(w, "null")
		return
	}

	_, _ = fmt.Fprintf(w, "%d", y)
}

func (y *Year) UnmarshalGQL(v any) error {
	switch v := v.(type) {
	case int:
		*y = Year(v)
	case int32:
		*y = Year(v)
	case int64:
		*y = Year(v)
	case uint:
		*y = Year(v)
	case uint32:
		*y = Year(v)
	case uint64:
		*y = Year(v)
	case float32:
		*y = Year(v)
	case float64:
		*y = Year(v)
	case string:
		_, err := fmt.Sscanf(v, "%d", y)
		if err != nil {
			return err
		}
	default:
		return errors.New("wrong type")
	}

	return nil
}
