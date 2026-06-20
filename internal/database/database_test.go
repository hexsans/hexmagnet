package database

import (
	"testing"

	"github.com/hexsans/hexmagnet/internal/database/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := postgres.NewDefaultConfig()
	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, "postgres", cfg.Username)
	assert.Equal(t, uint(5432), cfg.Port)
	assert.Equal(t, "hexmagnet", cfg.Database)
	assert.Equal(t, "disable", cfg.SSLMode)
	assert.Equal(t, 25, cfg.MaxConnections)
	assert.Empty(t, cfg.Password)
	assert.Equal(t, uint(0), cfg.ConnectionTimeout)
	assert.Empty(t, cfg.SSLCertPath)
	assert.Empty(t, cfg.SSLKeyPath)
	assert.Empty(t, cfg.SSLRootCertPath)
}

func TestConfigCreateDSN_Default(t *testing.T) {
	t.Parallel()

	cfg := postgres.NewDefaultConfig()
	dsn := cfg.CreateDSN()
	assert.Contains(t, dsn, "dbname=hexmagnet")
	assert.Contains(t, dsn, "user=postgres")
	assert.Contains(t, dsn, "host=localhost")
	assert.Contains(t, dsn, "port=5432")
	assert.Contains(t, dsn, "sslmode=disable")
	assert.NotContains(t, dsn, "password")
	assert.NotContains(t, dsn, "connect_timeout")
}

func TestConfigCreateDSN_WithAllFields(t *testing.T) {
	t.Parallel()

	cfg := postgres.Config{
		Host:              "db.example.com",
		Username:          "admin",
		Port:              5433,
		Database:          "testdb",
		ConnectionTimeout: 10,
		Password:          "secret",
		SSLMode:           "require",
		SSLCertPath:       "/certs/client.crt",
		SSLKeyPath:        "/certs/client.key",
		SSLRootCertPath:   "/certs/ca.crt",
		MaxConnections:    50,
	}
	dsn := cfg.CreateDSN()
	assert.Contains(t, dsn, "dbname=testdb")
	assert.Contains(t, dsn, "user=admin")
	assert.Contains(t, dsn, "host=db.example.com")
	assert.Contains(t, dsn, "port=5433")
	assert.Contains(t, dsn, "sslmode=require")
	assert.Contains(t, dsn, "password=secret")
	assert.Contains(t, dsn, "connect_timeout=10")
	assert.Contains(t, dsn, "sslcert=/certs/client.crt")
	assert.Contains(t, dsn, "sslkey=/certs/client.key")
	assert.Contains(t, dsn, "sslrootcert=/certs/ca.crt")
}

func TestNewModule(t *testing.T) {
	t.Parallel()

	opt := NewModule()
	assert.NotNil(t, opt)

	fxOpt := opt

	_ = fxOpt
}

func TestNewHealthCheck(t *testing.T) {
	t.Parallel()

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	sugar := logger.Sugar()
	defer func() { _ = sugar.Sync() }()

	result := NewHealthCheck(HealthCheckParams{
		Pool: nil,
	})
	assert.NotNil(t, result.Option)
}
