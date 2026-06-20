package fts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLexer(t *testing.T) {
	t.Parallel()

	l := NewLexer("")
	assert.Equal(t, 0, l.Pos())
}

func TestLexerRead_Empty(t *testing.T) {
	t.Parallel()

	l := NewLexer("")
	_, ok := l.Read()
	assert.False(t, ok)
}

func TestLexerRead_Basic(t *testing.T) {
	t.Parallel()

	l := NewLexer("ab")
	r, ok := l.Read()
	assert.True(t, ok)
	assert.Equal(t, 'a', r)
	assert.Equal(t, 1, l.Pos())

	r, ok = l.Read()
	assert.True(t, ok)
	assert.Equal(t, 'b', r)
	assert.Equal(t, 2, l.Pos())

	_, ok = l.Read()
	assert.False(t, ok)
	assert.Equal(t, 2, l.Pos())
}

func TestLexerRead_Unicode(t *testing.T) {
	t.Parallel()

	l := NewLexer("\u65e5\u672c\u8a9e")
	r, ok := l.Read()
	assert.True(t, ok)
	assert.Equal(t, '\u65e5', r)
	assert.Equal(t, 1, l.Pos())
}

func TestLexerBackup(t *testing.T) {
	t.Parallel()

	l := NewLexer("abc")
	l.Read()
	l.Backup()
	assert.Equal(t, 0, l.Pos())
	r, ok := l.Read()
	assert.True(t, ok)
	assert.Equal(t, 'a', r)
	assert.Equal(t, 1, l.Pos())

	l.Read()
	l.Backup()
	assert.Equal(t, 1, l.Pos())
	r, ok = l.Read()
	assert.True(t, ok)
	assert.Equal(t, 'b', r)
}

func TestLexerBackup_BeforeEOF(t *testing.T) {
	t.Parallel()

	l := NewLexer("ab")
	r, ok := l.Read()
	assert.True(t, ok)
	assert.Equal(t, 'a', r)
	l.Backup()
	r, ok = l.Read()
	assert.True(t, ok)
	assert.Equal(t, 'a', r)
	l.Read()
	l.Backup()
	r, ok = l.Read()
	assert.True(t, ok)
	assert.Equal(t, 'b', r)
}

func TestLexerBackupN_One(t *testing.T) {
	t.Parallel()

	l := NewLexer("abc")
	l.Read()
	l.Backup()
	assert.Equal(t, 0, l.Pos())
	r, ok := l.Read()
	assert.True(t, ok)
	assert.Equal(t, 'a', r)
}

func TestLexerPos(t *testing.T) {
	t.Parallel()

	l := NewLexer("hello")
	assert.Equal(t, 0, l.Pos())
	l.Read()
	assert.Equal(t, 1, l.Pos())
	l.Read()
	assert.Equal(t, 2, l.Pos())
	l.Backup()
	assert.Equal(t, 1, l.Pos())
}

func TestLexerIsEOF_Empty(t *testing.T) {
	t.Parallel()

	l := NewLexer("")
	assert.True(t, l.IsEOF())
}

func TestLexerIsEOF_NonEmpty(t *testing.T) {
	t.Parallel()

	l := NewLexer("a")
	assert.False(t, l.IsEOF())
	l.Read()
	assert.True(t, l.IsEOF())
}

func TestLexerReadIf(t *testing.T) {
	t.Parallel()

	l := NewLexer("abc")
	r, ok := l.ReadIf(IsChar('a'))
	assert.True(t, ok)
	assert.Equal(t, 'a', r)

	r, ok = l.ReadIf(IsChar('x'))
	assert.False(t, ok)
	assert.Equal(t, rune(0), r)

	r, ok = l.ReadIf(IsWordChar)
	assert.True(t, ok)
	assert.Equal(t, 'b', r)
}

func TestLexerReadWhile(t *testing.T) {
	t.Parallel()

	l := NewLexer("123abc")
	s := l.ReadWhile(IsInt)
	assert.Equal(t, "123", s)
	assert.Equal(t, 3, l.Pos())

	s = l.ReadWhile(IsWordChar)
	assert.Equal(t, "abc", s)
}

func TestLexerReadWhile_NoMatch(t *testing.T) {
	t.Parallel()

	l := NewLexer("abc")
	s := l.ReadWhile(IsInt)
	assert.Empty(t, s)
	assert.Equal(t, 0, l.Pos())
}

func TestLexerReadInt(t *testing.T) {
	t.Parallel()

	l := NewLexer("42abc")
	n, ok := l.ReadInt()
	assert.True(t, ok)
	assert.Equal(t, 42, n)

	l = NewLexer("abc")
	_, ok = l.ReadInt()
	assert.False(t, ok)

	l = NewLexer("0")
	n, ok = l.ReadInt()
	assert.True(t, ok)
	assert.Equal(t, 0, n)
}

func TestLexerReadChar(t *testing.T) {
	t.Parallel()

	l := NewLexer("abc")
	assert.True(t, l.ReadChar('a'))
	assert.False(t, l.ReadChar('a'))
	assert.True(t, l.ReadChar('b'))
	assert.True(t, l.ReadChar('c'))
	assert.False(t, l.ReadChar('d'))
}

func TestFtsLexerReadQuotedString(t *testing.T) {
	t.Parallel()

	l := newLexer("'hello'")
	s, err := l.readQuotedString('\'')
	require.NoError(t, err)
	assert.Equal(t, "hello", s)

	l = newLexer("'hello''world'")
	s, err = l.readQuotedString('\'')
	require.NoError(t, err)
	assert.Equal(t, "hello'world", s)
}

func TestFtsLexerReadQuotedString_NoOpenQuote(t *testing.T) {
	t.Parallel()

	l := newLexer("hello'")
	_, err := l.readQuotedString('\'')
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing opening quote")
}

func TestFtsLexerReadQuotedString_Unclosed(t *testing.T) {
	t.Parallel()

	l := newLexer("'hello")
	_, err := l.readQuotedString('\'')
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected EOF")
}

func TestFtsLexerReadQuotedString_EmptyString(t *testing.T) {
	t.Parallel()

	l := newLexer("''")
	s, err := l.readQuotedString('\'')
	require.NoError(t, err)
	assert.Empty(t, s)
}

func TestIsInt(t *testing.T) {
	t.Parallel()

	assert.True(t, IsInt('0'))
	assert.True(t, IsInt('5'))
	assert.True(t, IsInt('9'))
	assert.False(t, IsInt('a'))
	assert.False(t, IsInt('/'))
	assert.False(t, IsInt(':'))
}

func TestIsChar(t *testing.T) {
	t.Parallel()

	fn := IsChar('a')
	assert.True(t, fn('a'))
	assert.False(t, fn('b'))
	assert.False(t, fn('A'))

	fn = IsChar('1')
	assert.True(t, fn('1'))
	assert.False(t, fn('2'))
}

func TestIsWordChar(t *testing.T) {
	t.Parallel()

	assert.True(t, IsWordChar('a'))
	assert.True(t, IsWordChar('Z'))
	assert.True(t, IsWordChar('0'))
	assert.False(t, IsWordChar(' '))
	assert.False(t, IsWordChar('-'))
	assert.False(t, IsWordChar('.'))
	assert.False(t, IsWordChar('!'))
}

func TestIsNonWordChar(t *testing.T) {
	t.Parallel()

	assert.True(t, IsNonWordChar(' '))
	assert.True(t, IsNonWordChar('-'))
	assert.True(t, IsNonWordChar('.'))
	assert.False(t, IsNonWordChar('a'))
	assert.False(t, IsNonWordChar('Z'))
	assert.False(t, IsNonWordChar('0'))
}
