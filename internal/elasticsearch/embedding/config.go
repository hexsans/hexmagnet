package embedding

type Config struct {
	Endpoint           string `validate:"required" yaml:"endpoint"`
	APIKey             string `                    yaml:"api_key"`
	Model              string `validate:"required" yaml:"model"`
	Dimensions         int    `validate:"min=1"    yaml:"dimensions"`
	InstructionEnabled bool   `                    yaml:"instruction_enabled"`
}

func DefaultConfig() Config {
	return Config{
		Endpoint:   "http://localhost:11434/v1",
		Model:      "bge-m3",
		Dimensions: 1024,
	}
}
