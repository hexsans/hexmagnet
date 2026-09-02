package classifier

import (
	"gopkg.in/yaml.v3"
)

func newSourceProvider(tmdbEnabled bool) sourceProvider {
	return mergeSourceProvider{
		providers: []sourceProvider{
			yamlSourceProvider{rawSourceProvider: coreSourceProvider{}},
			configSourceProvider{tmdbEnabled: tmdbEnabled},
		},
	}
}

type sourceProvider interface {
	source() (Source, error)
}

type mergeSourceProvider struct {
	providers []sourceProvider
}

func (m mergeSourceProvider) source() (Source, error) {
	source := Source{}
	for _, p := range m.providers {
		s, err := p.source()
		if err != nil {
			return source, err
		}

		merged, err := source.merge(s)
		if err != nil {
			return source, err
		}

		source = merged
	}

	return source, nil
}

type rawSourceProvider interface {
	source() ([]byte, error)
}

type yamlSourceProvider struct {
	rawSourceProvider
}

func (y yamlSourceProvider) source() (Source, error) {
	raw, err := y.rawSourceProvider.source()
	if err != nil {
		return Source{}, err
	}

	rawWorkflow := make(map[string]any)

	parseErr := yaml.Unmarshal(raw, &rawWorkflow)
	if parseErr != nil {
		return Source{}, parseErr
	}

	src := Source{}

	decoder, decoderErr := newDecoder(&src)
	if decoderErr != nil {
		return Source{}, decoderErr
	}

	if decodeErr := decoder.Decode(rawWorkflow); decodeErr != nil {
		return Source{}, decodeErr
	}

	return src, nil
}

type coreSourceProvider struct{}

func (coreSourceProvider) source() ([]byte, error) {
	return classifierCoreYaml, nil
}

type configSourceProvider struct {
	tmdbEnabled bool
}

func (c configSourceProvider) source() (Source, error) {
	fs := make(Flags)
	if !c.tmdbEnabled {
		fs["tmdb_enabled"] = false
	}

	if len(fs) == 0 {
		return Source{}, nil
	}

	return Source{Flags: fs}, nil
}
