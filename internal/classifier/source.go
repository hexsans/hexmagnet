package classifier

import "maps"

type Source struct {
	Schema          string          `json:"$schema,omitempty" yaml:"$schema,omitempty"`
	Workflows       workflowSources `json:"workflows"`
	FlagDefinitions flagDefinitions `json:"flag_definitions"`
	Flags           Flags           `json:"flags"`
	Keywords        keywordGroups   `json:"keywords"`
	Extensions      extensionGroups `json:"extensions"`
}

func (s Source) merge(other Source) (Source, error) {
	flagDefs, err := s.FlagDefinitions.merge(other.FlagDefinitions)
	if err != nil {
		return Source{}, err
	}

	return Source{
		FlagDefinitions: flagDefs,
		Flags:           s.Flags.merge(other.Flags),
		Keywords:        s.Keywords.merge(other.Keywords),
		Extensions:      s.Extensions.merge(other.Extensions),
		Workflows:       s.Workflows.merge(other.Workflows),
	}, nil
}

func (s Source) workflowNames() map[string]struct{} {
	result := make(map[string]struct{})
	for k := range s.Workflows {
		result[k] = struct{}{}
	}

	return result
}

type keywordGroups map[string][]string

func (g keywordGroups) merge(other keywordGroups) keywordGroups {
	return keywordGroups(mergeStringSliceMap(g, other))
}

type extensionGroups map[string][]string

func (g extensionGroups) merge(other extensionGroups) extensionGroups {
	return extensionGroups(mergeStringSliceMap(g, other))
}

func mergeStringSliceMap[K comparable](a, b map[K][]string) map[K][]string {
	result := make(map[K][]string)

	for k, v := range a {
		if _, ok := b[k]; ok {
			result[k] = append(v, b[k]...)
		} else {
			result[k] = v
		}
	}

	for k, v := range b {
		if _, ok := result[k]; !ok {
			result[k] = v
		}
	}

	return result
}

type workflowSources map[string]any

func (s workflowSources) merge(other workflowSources) workflowSources {
	result := make(workflowSources)
	maps.Copy(result, s)

	maps.Copy(result, other)

	return result
}
