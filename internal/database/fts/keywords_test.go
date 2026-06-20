package fts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRexTokensFromKeywords_Empty(t *testing.T) {
	t.Parallel()

	_, err := NewRexTokensFromKeywords()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no keywords provided")
}

func TestNewRexTokensFromKeywords_Deduplicates(t *testing.T) {
	t.Parallel()

	tokens, err := NewRexTokensFromKeywords("foo", "foo")
	require.NoError(t, err)
	assert.Len(t, tokens, 1)
}

func TestNewRexTokensFromKeywords_Multiple(t *testing.T) {
	t.Parallel()

	tokens, err := NewRexTokensFromKeywords("foo", "bar")
	require.NoError(t, err)
	assert.Len(t, tokens, 2)
}

func TestNewRexTokensFromKeywords_WithModifiers(t *testing.T) {
	t.Parallel()

	tokens, err := NewRexTokensFromKeywords("foo?")
	require.NoError(t, err)
	assert.Len(t, tokens, 1)

	tokens, err = NewRexTokensFromKeywords("foo+")
	require.NoError(t, err)
	assert.Len(t, tokens, 1)

	tokens, err = NewRexTokensFromKeywords("*")
	require.NoError(t, err)
	assert.Len(t, tokens, 1)
}

func TestNewRegexFromKeywords(t *testing.T) {
	t.Parallel()

	re, err := NewRegexFromKeywords("foo")
	require.NoError(t, err)
	assert.NotNil(t, re)
	assert.True(t, re.MatchString("foo"))
	assert.True(t, re.MatchString(" foo "))
	assert.True(t, re.MatchString("foo bar"))
	assert.False(t, re.MatchString("foobar"))
	assert.False(t, re.MatchString("bar"))
}

func TestNewRegexFromKeywords_Digits(t *testing.T) {
	t.Parallel()

	re, err := NewRegexFromKeywords("version#")
	require.NoError(t, err)
	assert.NotNil(t, re)
	assert.True(t, re.MatchString("version1"))
	assert.True(t, re.MatchString("version2"))
	assert.False(t, re.MatchString("version"))
	assert.False(t, re.MatchString("version 1 "))
}

func TestNewRegexFromKeywords_Pipe(t *testing.T) {
	t.Parallel()

	re, err := NewRegexFromKeywords("foo|bar")
	require.NoError(t, err)
	assert.NotNil(t, re)
	assert.True(t, re.MatchString("foo"))
	assert.True(t, re.MatchString("bar"))
	assert.False(t, re.MatchString("baz"))
}

func TestMustNewRegexFromKeywords(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		re := MustNewRegexFromKeywords("hello")
		assert.NotNil(t, re)
	})
}

func TestMustNewRegexFromKeywords_Panics(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		MustNewRegexFromKeywords()
	})
}
