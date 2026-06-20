package enums

import "github.com/hexsans/hexmagnet/internal/model"

const orderBySizeValue = "size"

type enum struct {
	Name   string
	Values []string
}

func newEnum(name string, values []string) enum {
	return enum{
		Name:   name,
		Values: values,
	}
}

var Enums = []enum{
	newEnum("ContentType", model.ContentTypeNames()),
	newEnum("FacetLogic", model.FacetLogicNames()),
	newEnum("FileType", model.FileTypeNames()),
	newEnum("Language", model.LanguageValueStrings()),
	newEnum("TorrentContentOrderByField", []string{
		"relevance", "created_at", "updated_at", orderBySizeValue,
		"files_count", "seeders", "leechers", "name", "info_hash",
	}),
	newEnum("TorrentSearchOrderByField", []string{
		"relevance", "created_at", "updated_at", orderBySizeValue,
		"files_count", "seeders", "leechers", "name", "info_hash",
	}),
	newEnum("SortDirection", []string{"asc", "desc"}),
	newEnum("TorrentFilesOrderByField", []string{"index", "extension", orderBySizeValue}),
}
