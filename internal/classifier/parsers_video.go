package classifier

import (
	"strconv"

	"github.com/hedhyw/rex/pkg/dialect"
	"github.com/hedhyw/rex/pkg/rex"
	"github.com/hexsans/hexmagnet/internal/database/fts"
	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/hexsans/hexmagnet/internal/utils"
)

var titleTokens = []dialect.Token{
	rex.Group.Define(
		rex.Group.Composite(
			rex.Group.NonCaptured(
				utils.AnyWordChar().Repeat().OneOrMore(),
				rex.Group.NonCaptured(
					rex.Chars.Single('-'), utils.AnyWordChar().Repeat().OneOrMore(),
				).Repeat().ZeroOrMore(),
			),
			utils.AnyNonWordChar().Repeat().OneOrMore(),
		).NonCaptured().Repeat().OneOrMore(),
		rex.Group.Composite(
			utils.AnyNonWordChar().Repeat().OneOrMore(),
			rex.Chars.End(),
		).NonCaptured(),
	),
}

var titleRegex = rex.New(
	rex.Chars.Begin(),
	rex.Group.NonCaptured(titleTokens...),
).MustCompile()

var yearTokens = []dialect.Token{
	rex.Group.NonCaptured(rex.Common.NotClass(rex.Chars.WordCharacter()).Repeat().ZeroOrMore()),
	rex.Group.Define(
		rex.Group.Composite(
			rex.Common.Text("18"), rex.Common.Text("19"), rex.Common.Text("20"),
		).NonCaptured(),
		rex.Chars.Digits().Repeat().Exactly(2),
	),
	rex.Group.Composite(
		rex.Common.NotClass(rex.Chars.WordCharacter()),
		rex.Chars.End(),
	).NonCaptured(),
}

var titleYearRegex = rex.New(
	rex.Chars.Begin(),
	rex.Group.NonCaptured(rex.Group.NonCaptured(titleTokens...), rex.Group.NonCaptured(yearTokens...)),
).MustCompile()

var titleEpisodesRegex = rex.New(
	rex.Chars.Begin(),
	rex.Group.NonCaptured(
		rex.Group.NonCaptured(titleTokens...),
		model.EpisodesToken,
	),
).MustCompile()

var multiRegex = fts.MustNewRegexFromKeywords("multi", "dual")

var separatorToken = rex.Chars.Runes(" ._")

var titlePartRegex = rex.New(
	separatorToken.Repeat().ZeroOrOne(),
	rex.Group.Define(utils.WordToken()),
	separatorToken.Repeat().ZeroOrOne(),
).MustCompile()

var trimTitleRegex = rex.New(
	rex.Chars.Begin(),
	rex.Group.Composite(
		rex.Group.NonCaptured(
			rex.Chars.Single('['),
			rex.Common.NotClass(rex.Chars.Single(']')).Repeat().OneOrMore(),
			rex.Chars.Single(']'),
		),
		rex.Group.NonCaptured(
			rex.Chars.Single('【'),
			rex.Common.NotClass(rex.Chars.Single('】')).Repeat().OneOrMore(),
			rex.Chars.Single('】'),
		),
	).NonCaptured().Repeat().ZeroOrOne(),
	utils.AnyNonWordChar().Repeat().ZeroOrMore(),
	rex.Group.Define(
		utils.WordToken(),
		rex.Group.NonCaptured(
			rex.Chars.Any(),
			utils.WordToken(),
		).Repeat().ZeroOrMore(),
	),
	utils.AnyNonWordChar().Repeat().ZeroOrMore(),
	rex.Chars.End(),
).MustCompile()

func cleanTitle(title string) string {
	title = titlePartRegex.ReplaceAllStringFunc(title, func(s string) string {
		partMatch := titlePartRegex.FindStringSubmatch(s)
		if partMatch == nil {
			return ""
		}

		return partMatch[1] + " "
	})
	title = trimTitleRegex.ReplaceAllString(title, "$1")

	return title
}

func parseTitleYear(input string) (string, model.Year, string, error) {
	if match := titleYearRegex.FindStringSubmatch(input); match != nil {
		yearMatch, _ := strconv.ParseUint(match[2], 10, 16)
		title := cleanTitle(match[1])

		if title != "" {
			return title, model.Year(yearMatch), input[len(match[0]):], nil
		}
	}

	return "", 0, "", ErrUnmatched
}

func parseTitle(input string) (title string, rest string, err error) {
	if match := titleRegex.FindStringSubmatch(input); match != nil {
		title = cleanTitle(match[1])
		if title != "" {
			return title, input[len(match[0]):], nil
		}
	}

	return "", "", ErrUnmatched
}

func parseTitleYearEpisodes(input string) (string, model.Year, model.Episodes, string, error) {
	if match := titleEpisodesRegex.FindStringSubmatch(input); match != nil {
		title := match[1]
		year := model.Year(0)

		if t, y, _, err := parseTitleYear(title); err == nil {
			title = t
			year = y
		} else {
			title = cleanTitle(title)
		}

		episodes := model.EpisodesMatchToEpisodes(match[2:])

		return title, year, episodes, input[len(match[0]):], nil
	}

	return "", 0, nil, "", ErrUnmatched
}

func ParseTitleYearEpisodes(
	contentType model.NullContentType,
	input string,
) (string, model.Year, model.Episodes, string, error) {
	if !contentType.Valid ||
		contentType.ContentType == model.ContentTypeTvShow ||
		contentType.ContentType == model.ContentTypeUnknown {
		if title, year, episodes, rest, err := parseTitleYearEpisodes(input); err == nil {
			return title, year, episodes, rest, nil
		}
	}

	if title, year, rest, err := parseTitleYear(input); err == nil {
		return title, year, nil, rest, nil
	}

	if title, rest, err := parseTitle(input); err == nil {
		return title, 0, nil, rest, nil
	}

	return "", 0, nil, "", ErrUnmatched
}

func ParseVideoContent(torrent model.Torrent, result ClassificationResult) (ContentAttributes, error) {
	title, year, episodes, rest, err := ParseTitleYearEpisodes(result.ContentType, torrent.Name)
	if err != nil {
		if !result.ContentType.Valid {
			return ContentAttributes{}, err
		}

		rest = torrent.Name
	}

	ct := model.NullContentType{}

	switch {
	case len(episodes) > 0:
		ct = model.NullContentType{Valid: true, ContentType: model.ContentTypeTvShow}
	case result.ContentType.Valid:
		ct = model.NullContentType{Valid: true, ContentType: result.ContentType.ContentType}
	case !year.IsNil():
		ct = model.NullContentType{Valid: true, ContentType: model.ContentTypeMovie}
	}

	if year.IsNil() && len(episodes) == 0 {
		title = ""
		rest = torrent.Name
	}

	attrs := ContentAttributes{
		ContentType:   ct,
		BaseTitle:     model.NullString{Valid: title != "", String: title},
		Date:          model.Date{Year: year},
		Languages:     model.InferLanguages(rest),
		LanguageMulti: multiRegex.MatchString(rest),
	}

	return attrs, nil
}
