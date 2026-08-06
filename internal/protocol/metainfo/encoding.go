package metainfo

import (
	"bytes"
	"io"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

var detectionOrder = []encoding.Encoding{
	japanese.ShiftJIS,
	japanese.EUCJP,
	simplifiedchinese.GBK,
	traditionalchinese.Big5,
	korean.EUCKR,
}

func detectAndDecode(raw []byte) (string, bool) {
	if utf8.Valid(raw) {
		return string(raw), true
	}

	for _, enc := range detectionOrder {
		decoded, err := decodeBytes(raw, enc)
		if err == nil && utf8.Valid(decoded) {
			return string(decoded), true
		}
	}

	return string(raw), false
}

func decodeBytes(raw []byte, enc encoding.Encoding) ([]byte, error) {
	decoder := enc.NewDecoder()
	reader := transform.NewReader(bytes.NewReader(raw), decoder)

	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func NormalizeInfo(info *Info) {
	if info.NameUtf8 == "" && !utf8.ValidString(info.Name) {
		if decoded, ok := detectAndDecode([]byte(info.Name)); ok {
			info.NameUtf8 = decoded
		}
	}

	for i := range info.Files {
		if len(info.Files[i].PathUtf8) == 0 {
			converted := make([]string, len(info.Files[i].Path))
			allConverted := true

			for j, p := range info.Files[i].Path {
				if utf8.ValidString(p) {
					converted[j] = p
				} else if decoded, ok := detectAndDecode([]byte(p)); ok {
					converted[j] = decoded
				} else {
					allConverted = false
					break
				}
			}

			if allConverted {
				info.Files[i].PathUtf8 = converted
			}
		}
	}
}
