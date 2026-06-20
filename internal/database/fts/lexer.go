package fts

import (
	"errors"
)

func newLexer(str string) ftsLexer {
	return ftsLexer{NewLexer(str)}
}

type ftsLexer struct {
	Lexer
}

func (l *ftsLexer) readQuotedString(quoteChar rune) (string, error) {
	if !l.ReadChar(quoteChar) {
		return "", errors.New("missing opening quote")
	}

	var str string

	for {
		ch, ok := l.Read()
		if !ok {
			return str, errors.New("unexpected EOF")
		}

		if ch == quoteChar && !l.ReadChar(quoteChar) {
			break
		}

		str += string(ch)
	}

	return str, nil
}
