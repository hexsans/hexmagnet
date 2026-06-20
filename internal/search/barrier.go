package search

import (
	"encoding/base64"
	"fmt"
)

func EncodeBarrier(raw string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func DecodeBarrier(encoded string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode barrier: %w", err)
	}

	return string(b), nil
}
