package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const maxDefaultConfigSize = 64 * 1024 * 1024

// FilePath is the resolved path of the runtime config file. It is exposed so
// components that persist config changes write back to the same file the
// loader read from.
type FilePath string

type LoadResult struct {
	Path     FilePath
	Resolved *ResolvedConfig
}

func Load(specs ...SpecEntry) (*LoadResult, error) {
	hexmagnetPath := resolveConfigPath()

	if err := ensureParentDir(hexmagnetPath); err != nil {
		return nil, err
	}

	if err := ensureDefaultConfigFile(hexmagnetPath, specs); err != nil {
		return nil, err
	}

	val := newValidator()

	var resolvers []Resolver

	hexResolver, err := NewFromYamlFile(hexmagnetPath, false, val, WithPriority(-100))
	if err != nil {
		return nil, fmt.Errorf("primary config %s: %w", hexmagnetPath, err)
	}

	resolvers = append(resolvers, hexResolver)

	cwdResolver, err := NewFromYamlFile("./config.yml", true, val, WithPriority(10))
	if err != nil {
		return nil, fmt.Errorf("local config: %w", err)
	}

	if cwdResolver != nil {
		resolvers = append(resolvers, cwdResolver)
	}

	resolvers = append(resolvers, NewEnv(getEnvMap(), WithPriority(30)))

	resolved, err := Resolve(resolvers, val, specs)
	if err != nil {
		return nil, err
	}

	return &LoadResult{
		Path:     FilePath(hexmagnetPath),
		Resolved: resolved,
	}, nil
}

func resolveConfigPath() string {
	if v := os.Getenv("HEXMAGNET_CONFIG_FILE"); v != "" {
		return v
	}

	return "./hexmagnet.yaml"
}

func ensureParentDir(path string) error {
	parentDir := filepath.Dir(path)

	fi, err := os.Stat(parentDir)
	if err != nil {
		return fmt.Errorf("config directory %s does not exist: %w", parentDir, err)
	}

	if !fi.IsDir() {
		return fmt.Errorf("%s is not a directory", parentDir)
	}

	return nil
}

func ensureDefaultConfigFile(path string, specs []SpecEntry) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		data, err := buildDefaultConfigMap(specs)
		if err != nil {
			return fmt.Errorf("cannot build default config: %w", err)
		}

		yamlBytes, marshalErr := yaml.Marshal(data)
		if marshalErr != nil {
			return fmt.Errorf("cannot marshal default config: %w", marshalErr)
		}

		if len(yamlBytes) > maxDefaultConfigSize {
			return fmt.Errorf("default config too large")
		}

		out := make([]byte, 0, len(yamlHeader)+len(yamlBytes))
		out = append(out, yamlHeader...)

		out = append(out, yamlBytes...)
		if wErr := os.WriteFile(path, out, 0o644); wErr != nil {
			return fmt.Errorf("cannot create default config file %s: %w", path, wErr)
		}
	}

	return nil
}

func getEnvMap() map[string]string {
	env := os.Environ()

	m := make(map[string]string, len(env))
	for _, e := range env {
		for i := range len(e) {
			if e[i] == '=' {
				m[e[:i]] = e[i+1:]
				break
			}
		}
	}

	return m
}
