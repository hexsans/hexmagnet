package classifier

import (
	"strconv"
	"strings"

	"github.com/hexsans/hexmagnet/internal/model"
)

const llmClassifyName = "llm_classify"

type llmClassifyAction struct{}

func (llmClassifyAction) name() string {
	return llmClassifyName
}

var llmClassifyPayloadSpec = payloadLiteral[string]{
	literal:     llmClassifyName,
	description: "Classify the torrent using an LLM, falling back to rules on failure",
}

func (llmClassifyAction) compileAction(ctx compilerContext) (action, error) {
	if _, err := llmClassifyPayloadSpec.Unmarshal(ctx); err != nil {
		return action{}, ctx.error(err)
	}

	return action{
		run: func(ctx executionContext) (ClassificationResult, error) {
			cl := ctx.result

			if !cl.ContentType.Valid {
				cl.ContentType = model.NewNullContentType(model.ContentTypeUnknown)
			}

			if !ctx.llmEnabled {
				return cl, ErrUnmatched
			}

			llmFiles := make([]TorrentFile, 0, len(ctx.torrent.Files))
			for _, f := range ctx.torrent.Files {
				path := strings.Join(f.PathParts, "/")

				ext := ""
				if f.Extension.Valid {
					ext = f.Extension.String
				}

				llmFiles = append(llmFiles, TorrentFile{
					Path:      path,
					Size:      f.Size,
					Extension: ext,
				})
			}

			result, err := ctx.llmClient.Classify(ctx, ctx.torrent.Name, llmFiles, ctx.torrent.InfoHash.String())
			if err != nil {
				ctx.logger.Debugw("llm classify failed, falling back to rules",
					"info_hash", ctx.torrent.InfoHash,
					"name", ctx.torrent.Name,
					"error", err,
				)

				return cl, ErrUnmatched
			}

			applyLLMResult(&cl, result)

			return cl, nil
		},
	}, nil
}

func (llmClassifyAction) JSONSchema() JSONSchema {
	return llmClassifyPayloadSpec.JSONSchema()
}

func applyLLMResult(cl *ClassificationResult, result *LLMResult) {
	if result.Type != "" && result.Type != "unknown" {
		ct, err := model.ParseContentType(result.Type)
		if err == nil {
			cl.ContentType = model.NullContentType{ContentType: ct, Valid: true}
		}
	}

	if result.BaseTitle != "" {
		cl.BaseTitle = model.NewNullString(result.BaseTitle)
	}

	if result.Date != "" {
		d, err := model.NewDateFromIsoString(result.Date)
		if err == nil {
			cl.Date = d
		} else if year, parseErr := strconv.Atoi(result.Date); parseErr == nil && year >= 1000 && year <= 9999 {
			cl.Date.Year = model.Year(year)
		}
	}

	for _, lang := range result.Languages {
		if lang == "" {
			continue
		}

		langParsed := model.ParseLanguage(lang)
		if langParsed.Valid {
			if cl.Languages == nil {
				cl.Languages = make(model.Languages)
			}

			cl.Languages[langParsed.Language] = struct{}{}
		}
	}
}
