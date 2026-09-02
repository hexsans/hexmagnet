package classifier

import (
	"fmt"

	"github.com/hexsans/hexmagnet/internal/model"
)

type flagDefinitions map[string]FlagType

func (d flagDefinitions) merge(other flagDefinitions) (flagDefinitions, error) {
	result := make(flagDefinitions)

	for k, v := range d {
		tp, ok := other[k]
		if ok && tp != v {
			return nil, fmt.Errorf("conflicting flag definition %s", k)
		}

		result[k] = v
	}

	for k, v := range other {
		if _, ok := result[k]; !ok {
			result[k] = v
		}
	}

	return result, nil
}

type Flags map[string]any

func (f Flags) merge(other Flags) Flags {
	result := make(Flags)

	for k, v := range f {
		if _, ok := other[k]; ok {
			result[k] = other[k]
		} else {
			result[k] = v
		}
	}

	for k, v := range other {
		if _, ok := result[k]; !ok {
			result[k] = v
		}
	}

	return result
}

// validate checks a raw flag value against its declared type and returns a
// normalized value suitable for expression evaluation.
func (t FlagType) validate(rawVal any) (any, error) {
	switch t {
	case FlagTypeBool:
		if _, ok := rawVal.(bool); ok {
			return rawVal, nil
		}
	case FlagTypeString:
		if _, ok := rawVal.(string); ok {
			return rawVal, nil
		}
	case FlagTypeInt:
		if v, ok := rawVal.(int); ok {
			return v, nil
		}
	case FlagTypeStringList:
		if sliceVal, ok := rawVal.([]any); ok {
			nativeVal := make([]string, len(sliceVal))

			for i, v := range sliceVal {
				strVal, ok := v.(string)
				if !ok {
					return nil, fmt.Errorf("could not convert type %T to string", v)
				}

				nativeVal[i] = strVal
			}

			return nativeVal, nil
		}
	case FlagTypeContentTypeList:
		if sliceVal, ok := rawVal.([]any); ok {
			nativeVal := make([]int32, len(sliceVal))

			for i, v := range sliceVal {
				strVal, ok := v.(string)
				if !ok {
					return nil, fmt.Errorf("could not convert type %T to content type", v)
				}

				var ct model.NullContentType

				if strVal != contentTypeUnknown {
					parsed, parseErr := model.ParseContentType(strVal)
					if parseErr != nil {
						return nil, fmt.Errorf(
							"could not parse content type %s: %w",
							strVal,
							parseErr,
						)
					}

					ct = model.NewNullContentType(parsed)
				}

				nativeVal[i] = ContentTypeToInt(ct)
			}

			return nativeVal, nil
		}
	default:
		return nil, ErrInvalidFlagType
	}

	return nil, fmt.Errorf("could not convert type %T to %s", rawVal, t)
}
