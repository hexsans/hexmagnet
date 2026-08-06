package config

import "github.com/go-playground/validator/v10"

type ValidatorOption func(*validator.Validate)

func WithStructValidator(fn func(validator.StructLevel), structType any) ValidatorOption {
	return func(v *validator.Validate) {
		v.RegisterStructValidation(fn, structType)
	}
}

func newValidator() *validator.Validate {
	return validator.New()
}
