package classifier

import (
	"github.com/hexsans/hexmagnet/internal/database/fts"
	"github.com/hexsans/hexmagnet/internal/model"
)

// ExprEnv is the evaluation environment for expression conditions.
// Field names are exposed to expressions via the `expr` tags.
type ExprEnv struct {
	Torrent     Torrent             `expr:"torrent"`
	Result      Classification      `expr:"result"`
	Flags       map[string]any      `expr:"flags"`
	Keywords    map[string]string   `expr:"keywords"`
	Extensions  map[string][]string `expr:"extensions"`
	ContentType map[string]int32    `expr:"contentType"`
	FileType    map[string]int32    `expr:"fileType"`
	KB          int64               `expr:"kb"`
	MB          int64               `expr:"mb"`
	GB          int64               `expr:"gb"`
}

func exprEnvOption(src Source, ctx *compilerContext) error {
	keywords := make(map[string]string, len(src.Keywords))
	for group, kws := range src.Keywords {
		r, err := fts.NewRegexFromKeywords(kws...)
		if err != nil {
			return err
		}

		keywords[group] = r.String()
	}

	contentType := map[string]int32{contentTypeUnknown: ContentTypeUnknown}
	for _, ct := range model.ContentTypeValues() {
		contentType[ct.String()] = ContentTypeToInt(model.NullContentType{Valid: true, ContentType: ct})
	}

	fileType := map[string]int32{"unknown": FileTypeUnknown}
	for _, ft := range model.FileTypeValues() {
		fileType[ft.String()] = FileTypeToInt(model.NullFileType{Valid: true, FileType: ft})
	}

	ctx.exprEnv = ExprEnv{
		Keywords:    keywords,
		Extensions:  src.Extensions,
		ContentType: contentType,
		FileType:    fileType,
		KB:          1_000,
		MB:          1_000_000,
		GB:          1_000_000_000,
	}

	return nil
}
