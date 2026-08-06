package classifier

import "github.com/hexsans/hexmagnet/internal/model"

type ClassificationResult struct {
	ContentAttributes
	Content *model.Content
}

func (r *ClassificationResult) AttachContent(content *model.Content) {
	r.Content = content
	r.ContentType = model.NewNullContentType(content.Type)
}

type ContentAttributes struct {
	ContentType   model.NullContentType
	BaseTitle     model.NullString
	Date          model.Date
	Languages     model.Languages
	LanguageMulti bool
}

func (a *ContentAttributes) Merge(other ContentAttributes) {
	if !a.ContentType.Valid {
		a.ContentType = other.ContentType
	}

	if !a.BaseTitle.Valid {
		a.BaseTitle = other.BaseTitle
	}

	if a.Date.IsNil() {
		a.Date = other.Date
	}

	if len(a.Languages) == 0 {
		a.Languages = other.Languages
	}

	a.LanguageMulti = a.LanguageMulti || other.LanguageMulti
}
