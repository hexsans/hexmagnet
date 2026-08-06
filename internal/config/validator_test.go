package config

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithStructValidator_Triggers(t *testing.T) {
	t.Parallel()

	type LogCfg struct {
		Level string
		Path  string
	}

	val := validator.New()
	opt := WithStructValidator(func(sl validator.StructLevel) {
		lc := sl.Current().Interface().(LogCfg)
		if lc.Level != "off" && lc.Path == "" {
			sl.ReportError(lc.Path, "Path", "Path", "required", "")
		}
	}, LogCfg{})
	opt(val)

	err := val.Struct(LogCfg{Level: "info", Path: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestWithStructValidator_NoMatch(t *testing.T) {
	t.Parallel()

	type LogCfg struct {
		Level string
		Path  string
	}

	val := validator.New()
	opt := WithStructValidator(func(sl validator.StructLevel) {
		lc := sl.Current().Interface().(LogCfg)
		if lc.Level != "off" && lc.Path == "" {
			sl.ReportError(lc.Path, "Path", "Path", "required", "")
		}
	}, LogCfg{})
	opt(val)

	err := val.Struct(LogCfg{Level: "off", Path: ""})
	require.NoError(t, err)
}

func TestNewValidator(t *testing.T) {
	t.Parallel()

	val := newValidator()
	require.NotNil(t, val)
}
