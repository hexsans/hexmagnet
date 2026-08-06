package classifier

import (
	"encoding/json"
)

type JSONSchema map[string]any

func (s JSONSchema) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(map[string]any(s), "", "  ")
}

const schemaID = "https://hexmagnet.local/schemas/classifier-0.1.json"

func (f features) JSONSchema() JSONSchema {
	return map[string]any{
		"$schema":  "http://json-schema.org/draft-07/schema#",
		"$id":      schemaID,
		schemaType: schemaTypeObject,
		schemaProperties: map[string]any{
			"$schema": map[string]any{
				schemaConst: schemaID,
			},
			"workflows": map[string]any{
				schemaType: schemaTypeObject,
				schemaAdditionalProperties: map[string]any{
					schemaRef: refAction,
				},
			},
			"flag_definitions": map[string]any{
				schemaType: schemaTypeObject,
				schemaAdditionalProperties: map[string]any{
					schemaType: celTypeString,
					schemaEnum: FlagTypeValues(),
				},
			},
			"flags": map[string]any{
				schemaType:                 schemaTypeObject,
				schemaAdditionalProperties: true,
			},
			"keywords": map[string]any{
				schemaType: schemaTypeObject,
				schemaAdditionalProperties: map[string]any{
					schemaType: schemaTypeArray,
					schemaItems: map[string]any{
						schemaType: celTypeString,
					},
				},
			},
			"extensions": map[string]any{
				schemaType: schemaTypeObject,
				schemaAdditionalProperties: map[string]any{
					schemaType: schemaTypeArray,
					schemaItems: map[string]any{
						schemaType: celTypeString,
					},
				},
			},
		},
		schemaAdditionalProperties: false,
		"definitions": func() map[string]any {
			defs := map[string]any{
				"action": map[string]any{
					schemaOneOf: []map[string]any{
						{
							schemaRef: refActionSingle,
						},
						{
							schemaRef: "#/definitions/action_multi",
						},
					},
				},
				"action_multi": map[string]any{
					schemaType: schemaTypeArray,
					schemaItems: map[string]any{
						schemaRef: refActionSingle,
					},
				},
				"action_single": map[string]any{
					schemaOneOf: func() []map[string]any {
						result := make([]map[string]any, 0, len(f.actions))
						for _, def := range f.actions {
							result = append(result, map[string]any{
								schemaRef: "#/definitions/action__" + def.name(),
							})
						}

						return result
					}(),
				},
				schemaKeyCondition: map[string]any{
					schemaOneOf: func() []map[string]any {
						result := make([]map[string]any, 0, len(f.conditions))
						for _, def := range f.conditions {
							result = append(result, map[string]any{
								schemaRef: "#/definitions/condition__" + def.name(),
							})
						}

						return result
					}(),
				},
			}
			for _, def := range f.actions {
				defs["action__"+def.name()] = def.JSONSchema()
			}

			for _, def := range f.conditions {
				defs["condition__"+def.name()] = def.JSONSchema()
			}

			return defs
		}(),
	}
}
