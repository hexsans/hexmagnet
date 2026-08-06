package banning

import (
	"errors"
	"unicode"

	"github.com/hexsans/hexmagnet/internal/protocol/metainfo"
)

type contentChecker struct{}

func (contentChecker) Check(info metainfo.Info) error {
	checkStrings := make([]string, 0, len(info.Files)+1)
	checkStrings = append(checkStrings, info.BestName())

	for _, file := range info.Files {
		checkStrings = append(checkStrings, file.DisplayPath(&info))
	}

	for _, s := range checkStrings {
		if containsGarbage(s) {
			return errors.New("meta info contains garbage characters")
		}
	}

	return nil
}

func containsGarbage(s string) bool {
	hasCJK := false
	hasHalfWidthKana := false
	hasASCIILetter := false
	hasReplacement := false
	hasControl := false
	hasPUA := false

	for _, r := range s {
		switch {
		case r == unicode.ReplacementChar:
			hasReplacement = true
		case r < 0x20 && r != '\t' && r != '\n' && r != '\r':
			hasControl = true
		case r >= 0xE000 && r <= 0xF8FF ||
			r >= 0xF0000 && r <= 0xFFFFF ||
			r >= 0x100000 && r <= 0x10FFFF:
			hasPUA = true
		case unicode.Is(unicode.Han, r):
			hasCJK = true
		case r >= 0xFF65 && r <= 0xFF9F:
			hasHalfWidthKana = true
		case (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z'):
			hasASCIILetter = true
		}
	}

	if hasReplacement || hasControl || hasPUA {
		return true
	}

	if hasCJK && hasHalfWidthKana && hasASCIILetter {
		return true
	}

	return false
}
