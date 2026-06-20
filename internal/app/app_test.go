package app

import (
	"testing"
)

func TestNew_ResolvesDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := tmpDir + "/hexmagnet.yaml"
	t.Setenv("HEXMAGNET_CONFIG_FILE", configPath)

	ap := New()
	if err := ap.Err(); err != nil {
		t.Fatalf("app.New() construction failed — likely a DI wiring issue:\n%v", err)
	}
}
