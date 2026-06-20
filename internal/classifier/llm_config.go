package classifier

type LLMConfig struct {
	Endpoint        string  `json:"endpoint"         yaml:"endpoint"`
	APIKey          string  `json:"api_key"          yaml:"api_key"`
	Model           string  `json:"model"            yaml:"model"`
	Timeout         int     `json:"timeout"          yaml:"timeout"`
	MaxRetries      int     `json:"max_retries"      yaml:"max_retries"`
	Temperature     float64 `json:"temperature"      yaml:"temperature"`
	ReasoningEffort string  `json:"reasoning_effort" yaml:"reasoning_effort"`
	MaxFiles        int     `json:"max_files"        yaml:"max_files"`
	Enabled         bool    `json:"enabled"          yaml:"enabled"`
}

func NewDefaultLLMConfig() LLMConfig {
	return LLMConfig{MaxFiles: 30}
}

func (c LLMConfig) IsActive() bool {
	return c.Enabled && c.APIKey != ""
}

func (c LLMConfig) Merge(other LLMConfig) LLMConfig {
	merged := c
	if other.Endpoint != "" {
		merged.Endpoint = other.Endpoint
	}

	if other.APIKey != "" {
		merged.APIKey = other.APIKey
	}

	if other.Model != "" {
		merged.Model = other.Model
	}

	if other.Timeout != 0 {
		merged.Timeout = other.Timeout
	}

	if other.MaxRetries != 0 {
		merged.MaxRetries = other.MaxRetries
	}

	if other.Temperature != 0 {
		merged.Temperature = other.Temperature
	}

	if other.ReasoningEffort != "" {
		merged.ReasoningEffort = other.ReasoningEffort
	}

	if other.MaxFiles != 0 {
		merged.MaxFiles = other.MaxFiles
	}

	if other.Enabled {
		merged.Enabled = true
	}

	return merged
}
