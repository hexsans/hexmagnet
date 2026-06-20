package utils

import (
	"regexp"
	"testing"

	"github.com/hedhyw/rex/pkg/rex"
	"github.com/stretchr/testify/assert"
)

func TestAnyNonWordChar(t *testing.T) {
	t.Parallel()

	re := rex.New(rex.Group.Define(AnyNonWordChar()).Repeat().OneOrMore()).MustCompile()

	assert.True(t, re.MatchString("   "))
	assert.True(t, re.MatchString("!!!"))
	assert.True(t, re.MatchString("@#$%^"))
	assert.True(t, re.MatchString(" \t\n"))
	assert.False(t, re.MatchString("hello"))
	assert.False(t, re.MatchString("123"))
	assert.False(t, re.MatchString("hëllo"))
	assert.False(t, re.MatchString("abc123"))
}

func TestAnyNonWordChar_Empty(t *testing.T) {
	t.Parallel()

	re := rex.New(rex.Group.Define(AnyNonWordChar()).Repeat().OneOrMore()).MustCompile()
	assert.False(t, re.MatchString(""))
}

func TestWordTokenRegex(t *testing.T) {
	t.Parallel()

	re := WordTokenRegex()
	assert.IsType(t, &regexp.Regexp{}, re)

	assert.True(t, re.MatchString("hello"))
	assert.True(t, re.MatchString("it's"))
	assert.True(t, re.MatchString("don't"))
	assert.False(t, re.MatchString("   "))
	assert.False(t, re.MatchString("!!!"))

	matches := re.FindAllString("hello world", -1)
	assert.Equal(t, []string{"hello", "world"}, matches)
}

func TestWordTokenRegex_OpeningClosingPunctuation(t *testing.T) {
	t.Parallel()

	re := WordTokenRegex()

	assert.True(t, re.MatchString("\"hello\""))
	assert.True(t, re.MatchString("'hello'"))
	assert.True(t, re.MatchString("(hello)"))
	assert.False(t, re.MatchString(","))

	matches := re.FindAllString("\"hello, world!\"", -1)
	assert.Equal(t, []string{"\"hello,", "world!\""}, matches)
}

func TestNormalizeString_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World!", "hello world!"},
		{"HELLO", "hello"},
		{"  spaces  ", "spaces"},
		{"it's fine", "it's fine"},
		{"U.S.A. rocks", "u s a rocks"},
		{"!!!noise!!!", "noise!!!"},
		{"NoN-word", "non-word"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			result := NormalizeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeString_Empty(t *testing.T) {
	t.Parallel()

	assert.Empty(t, NormalizeString(""))
	assert.Empty(t, NormalizeString("   "))
}

func TestNormalizeString_SingleToken(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "hello", NormalizeString("hello"))
	assert.Equal(t, "hello", NormalizeString("  hello  "))
	assert.Equal(t, "(hello)", NormalizeString("(hello)"))
}
