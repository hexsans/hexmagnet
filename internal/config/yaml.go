package config

import (
	"os"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

func NewFromYamlFile(path string, ignoreMissing bool, val *validator.Validate, options ...ResolverOption) (Resolver, error) {
	m := make(map[string]any)

	data, readErr := os.ReadFile(path)
	if readErr != nil {
		if !ignoreMissing || !os.IsNotExist(readErr) {
			return nil, readErr
		}
	} else {
		parseErr := yaml.Unmarshal(data, &m)
		if parseErr != nil {
			return nil, parseErr
		}
	}

	return NewMap(m, val, append([]ResolverOption{WithKey(path)}, options...)...), nil
}
