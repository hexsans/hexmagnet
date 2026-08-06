package postgres

import (
	"fmt"
	"strings"
)

type Config struct {
	Host              string `validate:"required"                                                 yaml:"host"`
	Username          string `validate:"required"                                                 yaml:"username"`
	Port              uint   `                                                                    yaml:"port"`
	Database          string `validate:"required"                                                 yaml:"database"`
	ConnectionTimeout uint   `                                                                    yaml:"connection_timeout"`
	Password          string `                                                                    yaml:"password"`
	SSLMode           string `validate:"oneof=disable allow prefer require verify-ca verify-full" yaml:"ssl_mode"`
	SSLCertPath       string `                                                                    yaml:"ssl_cert_path"`
	SSLKeyPath        string `                                                                    yaml:"ssl_key_path"`
	SSLRootCertPath   string `                                                                    yaml:"ssl_root_cert_path"`
	MaxConnections    int    `validate:"gte=1"                                                    yaml:"max_connections"`
}

func NewDefaultConfig() Config {
	return Config{
		Host:           "localhost",
		Username:       "postgres",
		Port:           5432,
		Database:       "hexmagnet",
		SSLMode:        "disable",
		MaxConnections: 25,
	}
}

func (c *Config) CreateDSN() string {
	vals := dbValues(c)
	p := make([]string, 0, len(vals))

	for k, v := range vals {
		p = append(p, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(p, " ")
}

func setIfNotEmpty(m map[string]string, key string, val any) {
	strVal := fmt.Sprintf("%v", val)
	if strVal != "" {
		m[key] = strVal
	}
}

func setIfPositive(m map[string]string, key string, val uint) {
	if val > 0 {
		m[key] = fmt.Sprintf("%d", val)
	}
}

func dbValues(cfg *Config) map[string]string {
	p := map[string]string{}
	setIfNotEmpty(p, "dbname", cfg.Database)
	setIfNotEmpty(p, "user", cfg.Username)
	setIfNotEmpty(p, "host", cfg.Host)
	setIfNotEmpty(p, "port", fmt.Sprintf("%d", cfg.Port))
	setIfNotEmpty(p, "sslmode", cfg.SSLMode)
	setIfPositive(p, "connect_timeout", cfg.ConnectionTimeout)
	setIfNotEmpty(p, "password", cfg.Password)
	setIfNotEmpty(p, "sslcert", cfg.SSLCertPath)
	setIfNotEmpty(p, "sslkey", cfg.SSLKeyPath)
	setIfNotEmpty(p, "sslrootcert", cfg.SSLRootCertPath)

	return p
}
