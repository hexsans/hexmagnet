package torznab

import (
	"testing"

	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCategoryForContentType(t *testing.T) {
	t.Parallel()
	assert.Equal(t, catMovies, categoryForContentType(model.ContentTypeMovie))
	assert.Equal(t, catTV, categoryForContentType(model.ContentTypeTvShow))
	assert.Equal(t, catAudio, categoryForContentType(model.ContentTypeMusic))
	assert.Equal(t, catBooks, categoryForContentType(model.ContentTypeEbook))
	assert.Equal(t, catBooksComics, categoryForContentType(model.ContentTypeComic))
	assert.Equal(t, catBooksAudiobook, categoryForContentType(model.ContentTypeAudiobook))
	assert.Equal(t, catConsole, categoryForContentType(model.ContentTypeGame))
	assert.Equal(t, catSoftware, categoryForContentType(model.ContentTypeSoftware))
	assert.Equal(t, catXXX, categoryForContentType(model.ContentTypeAdult))
	assert.Equal(t, catOther, categoryForContentType(model.ContentTypeOther))
}

func TestContentTypesForCategory(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []model.ContentType{model.ContentTypeMovie}, contentTypesForCategory(2000))
	assert.Equal(t, []model.ContentType{model.ContentTypeMovie}, contentTypesForCategory(2070))
	assert.Equal(t, []model.ContentType{model.ContentTypeTvShow}, contentTypesForCategory(5000))
	assert.Equal(t, []model.ContentType{model.ContentTypeMusic}, contentTypesForCategory(3020))
	assert.Equal(t, []model.ContentType{model.ContentTypeEbook}, contentTypesForCategory(7000))
	assert.Equal(t, []model.ContentType{model.ContentTypeComic}, contentTypesForCategory(7020))
	assert.Equal(t, []model.ContentType{model.ContentTypeAudiobook}, contentTypesForCategory(7030))
	assert.Equal(t, []model.ContentType{model.ContentTypeOther, model.ContentTypeUnknown}, contentTypesForCategory(8000))
}

func TestParseCategories(t *testing.T) {
	t.Parallel()

	types, err := parseCategories("2000,5000")
	require.NoError(t, err)

	assert.Equal(t, []model.ContentType{model.ContentTypeMovie, model.ContentTypeTvShow}, types)

	types, err = parseCategories("2070")
	require.NoError(t, err)
	assert.Equal(t, []model.ContentType{model.ContentTypeMovie}, types)

	_, err = parseCategories("abc")
	require.Error(t, err)

	types, err = parseCategories("")
	require.NoError(t, err)
	assert.Nil(t, types)
}

func TestAllowedCategories(t *testing.T) {
	t.Parallel()
	assert.True(t, allowedCategories([]string{"*"}, 2000))
	assert.True(t, allowedCategories([]string{"2000"}, 2000))
	assert.True(t, allowedCategories([]string{"2000"}, 2070))
	assert.False(t, allowedCategories([]string{"2000"}, 5000))
	assert.False(t, allowedCategories([]string{"2000", "3000"}, 5000))
	assert.True(t, allowedCategories([]string{"5000"}, 5030))
}

func TestSplitCSV(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"2000", "5000"}, splitCSV("2000,5000"))
	assert.Equal(t, []string{"2000"}, splitCSV("2000"))
	assert.Nil(t, splitCSV(""))
	assert.Equal(t, []string{"2000", "5000"}, splitCSV("2000,5000,"))
}

func TestCapabilitiesHonorsCategoryFilter(t *testing.T) {
	t.Parallel()

	cfg := NewDefaultConfig()
	cfg.Categories = []string{"2000", "5000"}

	body := capabilitiesFor(cfg)

	for _, allowed := range []string{`id="2000"`, `id="5000"`} {
		assert.Contains(t, string(body), allowed)
	}

	for _, blocked := range []string{`id="3000"`, `id="7000"`} {
		assert.NotContains(t, string(body), blocked)
	}
}
