package classifier

import (
	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/common/types/ref"
)

type Provider struct{}

func (Provider) EnumValue(enumName string) ref.Val {
	return types.NewErr("unknown enum: %s", enumName)
}

func (Provider) FindIdent(_ string) (ref.Val, bool) {
	return nil, false
}

func (Provider) FindStructType(structType string) (*types.Type, bool) {
	switch structType {
	case "Torrent", "File", "Classification":
		return types.NewObjectType(structType), true
	}

	return nil, false
}

func (Provider) FindStructFieldNames(structType string) ([]string, bool) {
	names, ok := structFields[structType]
	return names, ok
}

func (Provider) FindStructFieldType(structType, fieldName string) (*types.FieldType, bool) {
	fields, ok := structFields[structType]
	if !ok {
		return nil, false
	}

	for _, f := range fields {
		if f == fieldName {
			ft, ok := fieldTypes[structType+"."+fieldName]
			if !ok {
				return nil, false
			}

			return &types.FieldType{Type: ft}, true
		}
	}

	return nil, false
}

func (Provider) NewValue(_ string, _ map[string]ref.Val) ref.Val {
	return types.NewErr("unsupported")
}

var structFields = map[string][]string{
	"Torrent": {
		"infoHash",
		"name",
		celFieldBaseName,
		celFieldSize,
		celFieldExtension,
		"files",
		"filesCount",
		"filesSize",
		"fileExtensions",
	},
	"File": {
		"index",
		celFieldPath,
		celFieldBasePath,
		celFieldBaseName,
		celFieldSize,
		celFieldExtension,
		celFieldFileType,
	},
	"Classification": {
		"contentType",
		"hasAttachedContent",
		"hasBaseTitle",
		"date",
		"languages",
		"contentId",
		"contentSource",
	},
}

var fileType = types.NewObjectType("File")

var fieldTypes = map[string]*types.Type{
	"Torrent.infoHash":                  types.StringType,
	"Torrent.name":                      types.StringType,
	"Torrent.baseName":                  types.StringType,
	"Torrent.size":                      types.IntType,
	"Torrent.extension":                 types.StringType,
	"Torrent.files":                     types.NewListType(fileType),
	"Torrent.filesCount":                types.IntType,
	"Torrent.filesSize":                 types.IntType,
	"Torrent.fileExtensions":            types.NewListType(types.StringType),
	"File.index":                        types.IntType,
	"File.path":                         types.StringType,
	"File.basePath":                     types.StringType,
	"File.baseName":                     types.StringType,
	"File.size":                         types.IntType,
	"File.extension":                    types.StringType,
	"File.fileType":                     types.IntType,
	"Classification.contentType":        types.IntType,
	"Classification.hasAttachedContent": types.BoolType,
	"Classification.hasBaseTitle":       types.BoolType,
	"Classification.date":               types.IntType,
	"Classification.languages":          types.NewListType(types.StringType),
	"Classification.contentId":          types.StringType,
	"Classification.contentSource":      types.StringType,
}
