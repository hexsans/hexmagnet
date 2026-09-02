package classifier

import (
	"testing"

	"github.com/hexsans/hexmagnet/internal/database/fts"
	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestParseDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected model.Date
	}{
		{"2020-01-01", model.Date{Year: 2020, Month: 1, Day: 1}},
		{"01-01-2020", model.Date{Year: 2020, Month: 1, Day: 1}},
		{"01-Jan-2020", model.Date{Year: 2020, Month: 1, Day: 1}},
		{"Jan-01-2020", model.Date{Year: 2020, Month: 1, Day: 1}},
		{"MP3-daily-2019-July-16-Disco", model.Date{Year: 2019, Month: 7, Day: 16}},
		{"XXX Video (2022-09-21) 1080p.mp4", model.Date{Year: 2022, Month: 9, Day: 21}},
		{"Studio.23.02.01.Bla.Bla.Bla.XXX.1080p.HEVC.x265.GRP[GRP]", model.Date{Year: 2023, Month: 2, Day: 1}},
		{"The Movie (13.10.2017)_1080p.mp4", model.Date{Year: 2017, Month: 10, Day: 13}},
		{"Movie.23.05.15..The.Best.Of.XXX.1080p.MP4-GRP[GRP]", model.Date{Year: 2023, Month: 5, Day: 15}},
		{
			"2021.09.11_Serie_C_2021.22_R.03_Xxx_FC_vs_Xxx_FC_[football.net]_720p.50_RUS.mkv",
			model.Date{Year: 2021, Month: 9, Day: 11},
		},
		// {"Bla Bla June 27, 2015", model.Date{Year: 2015, Month: 6, Day: 27}},
		{input: "Software.Pro.X2.Suite.v19.0.2.23117-R2R"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()

			result := ParseDate(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestLexDatePart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		format   datePartFormat
		expected model.Date
	}{
		{"0", datePart1Digit, model.Date{}},
		{"5", datePart1Digit, model.Date{Day: 5, Month: 5}},
		{"9", datePart1Digit, model.Date{Day: 9, Month: 9}},
		{"0024", datePart4Digits, model.Date{}},
		{"9999", datePart4Digits, model.Date{Year: 9999}},
		{"2024", datePart4Digits, model.Date{Year: 2024}},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()

			part := (&dateLexer{Lexer: fts.NewLexer(test.input)}).lexDatePart()
			assert.Equal(t, test.format, part.format)
			assert.Equal(t, test.expected, part.Date)
		})
	}
}
