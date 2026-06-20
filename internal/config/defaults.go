package config

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
)

func buildDefaultConfigMap(specs []SpecEntry) (map[string]any, error) {
	result := make(map[string]any)

	for _, spec := range specs {
		keys := strings.Split(spec.Key, ".")

		structMap, err := structToYamlMap(reflect.ValueOf(spec.DefaultValue))
		if err != nil {
			return nil, fmt.Errorf("default for %q: %w", spec.Key, err)
		}

		placeAtPath(result, keys, structMap)
	}

	return result, nil
}

func structToYamlMap(v reflect.Value) (map[string]any, error) {
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %v", v.Kind())
	}

	result := make(map[string]any)

	t := v.Type()
	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		key := strcase.ToSnake(field.Name)
		fv := v.Field(i)

		if fv.Type() == durationType {
			d, ok := fv.Interface().(time.Duration)
			if !ok {
				return nil, fmt.Errorf("field %s: expected time.Duration, got %T", field.Name, fv.Interface())
			}

			result[key] = d.String()

			continue
		}

		switch fv.Kind() {
		case reflect.Pointer:
			if fv.IsNil() {
				result[key] = nil
			} else {
				nested, err := structToYamlMap(fv.Elem())
				if err != nil {
					return nil, err
				}

				result[key] = nested
			}
		case reflect.Struct:
			nested, err := structToYamlMap(fv)
			if err != nil {
				return nil, err
			}

			result[key] = nested
		case reflect.Slice:
			if fv.Type().Elem().Kind() == reflect.Struct {
				slice := make([]any, fv.Len())
				for j := range fv.Len() {
					elem, err := structToYamlMap(fv.Index(j))
					if err != nil {
						return nil, err
					}

					slice[j] = elem
				}

				result[key] = slice
			} else {
				result[key] = fv.Interface()
			}
		default:
			result[key] = fv.Interface()
		}
	}

	return result, nil
}

func placeAtPath(m map[string]any, keys []string, value any) {
	current := m

	for i, key := range keys {
		if i == len(keys)-1 {
			current[key] = value
		} else {
			if _, ok := current[key]; !ok {
				current[key] = make(map[string]any)
			}

			next, ok := current[key].(map[string]any)
			if !ok {
				next = make(map[string]any)
				current[key] = next
			}

			current = next
		}
	}
}

var yamlHeader = []byte("# HexMagnet runtime configuration\n" +
	"# Path: ./hexmagnet.yaml (or $HEXMAGNET_CONFIG_FILE)\n\n")
