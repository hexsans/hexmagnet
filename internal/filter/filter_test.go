package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompile_CaseInsensitive(t *testing.T) {
	t.Parallel()

	p, err := Compile([]string{"xxx"}, []string{`\.exe$`})
	require.NoError(t, err)

	assert.True(t, AnyMatch(p.Titles(), "XXX video"))
	assert.True(t, AnyMatch(p.Filenames(), "setup.EXE"))
	assert.False(t, AnyMatch(p.Titles(), "clean video"))
	assert.False(t, AnyMatch(p.Filenames(), "readme.txt"))
}

func TestCompile_InvalidTitlePattern(t *testing.T) {
	t.Parallel()

	_, err := Compile([]string{"["}, nil)
	assert.ErrorContains(t, err, "invalid title pattern")
}

func TestCompile_InvalidFilenamePattern(t *testing.T) {
	t.Parallel()

	_, err := Compile(nil, []string{"("})
	assert.ErrorContains(t, err, "invalid filename pattern")
}

func TestCompile_Empty(t *testing.T) {
	t.Parallel()

	p, err := Compile(nil, nil)
	require.NoError(t, err)
	assert.Empty(t, p.Titles())
	assert.Empty(t, p.Filenames())
	assert.False(t, AnyMatch(p.Titles(), "anything"))
}

func TestAnyMatch_Or(t *testing.T) {
	t.Parallel()

	p, err := Compile([]string{"first", "second"}, nil)
	require.NoError(t, err)

	assert.True(t, AnyMatch(p.Titles(), "the second one"))
	assert.False(t, AnyMatch(p.Titles(), "the third one"))
}
