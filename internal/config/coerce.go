package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var durationType = reflect.TypeFor[time.Duration]()

func coerceStringValue(stringValue string, valueType reflect.Type) (any, error) {
	if valueType == durationType {
		return time.ParseDuration(stringValue)
	}

	switch valueType.Kind() {
	case reflect.String:
		return stringValue, nil

	case reflect.Bool:
		switch strings.ToLower(stringValue) {
		case "true", "1":
			return true, nil
		case "false", "0":
			return false, nil
		}

		return nil, fmt.Errorf("cannot coerce %q to bool", stringValue)

	case reflect.Int:
		return strconv.Atoi(stringValue)
	case reflect.Int8:
		v, err := strconv.ParseInt(stringValue, 10, 8)
		return int8(v), err
	case reflect.Int16:
		v, err := strconv.ParseInt(stringValue, 10, 16)
		return int16(v), err
	case reflect.Int32:
		v, err := strconv.ParseInt(stringValue, 10, 32)
		return int32(v), err
	case reflect.Int64:
		return strconv.ParseInt(stringValue, 10, 64)

	case reflect.Uint:
		v, err := strconv.ParseUint(stringValue, 10, 64)
		return uint(v), err
	case reflect.Uint8:
		v, err := strconv.ParseUint(stringValue, 10, 8)
		return uint8(v), err
	case reflect.Uint16:
		v, err := strconv.ParseUint(stringValue, 10, 16)
		return uint16(v), err
	case reflect.Uint32:
		v, err := strconv.ParseUint(stringValue, 10, 32)
		return uint32(v), err
	case reflect.Uint64:
		return strconv.ParseUint(stringValue, 10, 64)

	case reflect.Float32:
		v, err := strconv.ParseFloat(stringValue, 32)
		return float32(v), err
	case reflect.Float64:
		return strconv.ParseFloat(stringValue, 64)

	case reflect.Slice:
		strValues := strings.Split(stringValue, ",")
		values := make([]any, len(strValues))

		for i, strValue := range strValues {
			coercedValue, err := coerceStringValue(strValue, valueType.Elem())
			if err != nil {
				return nil, fmt.Errorf("slice element %d: %w", i, err)
			}

			values[i] = coercedValue
		}

		return values, nil

	default:
		return nil, fmt.Errorf("cannot coerce string %q to unsupported type %v", stringValue, valueType)
	}
}
