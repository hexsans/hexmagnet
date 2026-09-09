package torznab

import (
	"fmt"
	"strconv"

	"github.com/hexsans/hexmagnet/internal/model"
)

// Newznab category IDs (base categories per the Newznab spec).
const (
	catConsole  = 1000
	catMovies   = 2000
	catAudio    = 3000
	catSoftware = 4000
	catTV       = 5000
	catXXX      = 6000
	catBooks    = 7000
	catOther    = 8000

	catBooksComics    = 7020
	catBooksAudiobook = 7030
)

var baseCategories = []int{catConsole, catMovies, catAudio, catSoftware, catTV, catXXX, catBooks, catOther}

// categoryForContentType returns the primary Newznab category for a content type.
func categoryForContentType(ct model.ContentType) int {
	switch ct {
	case model.ContentTypeMovie:
		return catMovies
	case model.ContentTypeTvShow:
		return catTV
	case model.ContentTypeMusic:
		return catAudio
	case model.ContentTypeEbook:
		return catBooks
	case model.ContentTypeComic:
		return catBooksComics
	case model.ContentTypeAudiobook:
		return catBooksAudiobook
	case model.ContentTypeGame:
		return catConsole
	case model.ContentTypeSoftware:
		return catSoftware
	case model.ContentTypeAdult:
		return catXXX
	default:
		return catOther
	}
}

// contentTypesForCategory maps a Newznab category (or sub-category) to the
// HexMagnet content types it represents.
func contentTypesForCategory(cat int) []model.ContentType {
	switch cat / 1000 {
	case 1:
		return []model.ContentType{model.ContentTypeGame}
	case 2:
		return []model.ContentType{model.ContentTypeMovie}
	case 3:
		return []model.ContentType{model.ContentTypeMusic}
	case 4:
		return []model.ContentType{model.ContentTypeSoftware}
	case 5:
		return []model.ContentType{model.ContentTypeTvShow}
	case 6:
		return []model.ContentType{model.ContentTypeAdult}
	case 7:
		switch cat {
		case catBooksComics:
			return []model.ContentType{model.ContentTypeComic}
		case catBooksAudiobook:
			return []model.ContentType{model.ContentTypeAudiobook}
		default:
			return []model.ContentType{model.ContentTypeEbook}
		}
	default:
		return []model.ContentType{model.ContentTypeOther, model.ContentTypeUnknown}
	}
}

// parseCategories parses the ?cat= parameter (comma-separated Newznab IDs) into
// a set of content types. An empty input returns nil (no filtering).
func parseCategories(catParam string) ([]model.ContentType, error) {
	if catParam == "" {
		return nil, nil
	}

	var types []model.ContentType

	seen := make(map[model.ContentType]struct{})

	for _, part := range splitCSV(catParam) {
		cat, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid category %q", part)
		}

		for _, ct := range contentTypesForCategory(cat) {
			if _, ok := seen[ct]; ok {
				continue
			}

			seen[ct] = struct{}{}
			types = append(types, ct)
		}
	}

	return types, nil
}

// allowedCategories reports whether a Newznab category is permitted by the
// configured category allow-list. The "*" entry allows everything.
func allowedCategories(configCats []string, category int) bool {
	for _, c := range configCats {
		if c == "*" {
			return true
		}

		parsed, err := strconv.Atoi(c)
		if err != nil {
			continue
		}

		if parsed == category {
			return true
		}

		if parsed%1000 == 0 && parsed/1000 == category/1000 {
			return true
		}
	}

	return false
}

func splitCSV(s string) []string {
	var out []string

	start := 0

	for i := range len(s) {
		if s[i] == ',' {
			if i > start {
				out = append(out, s[start:i])
			}

			start = i + 1
		}
	}

	if start < len(s) {
		out = append(out, s[start:])
	}

	return out
}
