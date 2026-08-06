package classifier

const (
	// JSON schema keywords.
	schemaAdditionalProperties = "additionalProperties"
	schemaConst                = "const"
	schemaEnum                 = "enum"
	schemaItems                = "items"
	schemaKeyCondition         = "condition"
	schemaOneOf                = "oneOf"
	schemaProperties           = "properties"
	schemaRef                  = "$ref"
	schemaType                 = "type"

	schemaTypeArray  = "array"
	schemaTypeObject = "object"
	schemaTypeString = "string"

	refAction       = "#/definitions/action"
	refActionSingle = "#/definitions/action_single"
	refCondition    = "#/definitions/condition"
)

// CEL field names of the exposed struct types.
const (
	celFieldBaseName  = "baseName"
	celFieldBasePath  = "basePath"
	celFieldExtension = "extension"
	celFieldFileType  = "fileType"
	celFieldPath      = "path"
	celFieldSize      = "size"
)

// CEL type names.
const (
	celTypeBool      = "bool"
	celTypeBytes     = "bytes"
	celTypeDouble    = "double"
	celTypeDuration  = "duration"
	celTypeInt       = "int"
	celTypeString    = "string"
	celTypeTimestamp = "timestamp"
	celTypeUint      = "uint"
)

const (
	contentTypeUnknown  = "unknown"
	reasoningEffortNone = "none"
)
