package tmdb

type Config struct {
	Enabled     bool   `yaml:"enabled"`
	AccessToken string `yaml:"access_token"`
	RateLimit   int    `validate:"gte=1" yaml:"rate_limit"`
}

func NewDefaultConfig() Config {
	return Config{
		Enabled:     true,
		AccessToken: "",
		RateLimit:   20,
	}
}
