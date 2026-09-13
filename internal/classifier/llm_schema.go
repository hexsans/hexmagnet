package classifier

import "encoding/json"

const llmResultSchemaName = "torrent_classification"

var llmResultSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "type": {
      "type": "string",
      "enum": [
        "movie",
        "tv_show",
        "music",
        "ebook",
        "comic",
        "audiobook",
        "game",
        "software",
        "adult",
        "other",
        "unknown"
      ]
    },
    "base_title": {"type": "string"},
    "date": {"type": ["string", "null"]},
    "languages": {"type": ["array", "null"], "items": {"type": "string"}}
  },
  "required": ["type", "base_title", "date", "languages"],
  "additionalProperties": false
}`)

func newJSONSchemaResponseFormat() *responseFormat {
	return &responseFormat{
		Type: "json_schema",
		JSONSchema: &jsonSchemaSpec{
			Name:   llmResultSchemaName,
			Strict: true,
			Schema: llmResultSchema,
		},
	}
}
