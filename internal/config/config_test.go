package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvDoesNotOverrideEnvironment(t *testing.T) {
	key := "AI_OPERATIONS_COPILOT_TEST_KEY"
	t.Setenv(key, "from-environment")
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(key+"=from-file\nSECOND_KEY=from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	loadDotEnv(path)
	if got := os.Getenv(key); got != "from-environment" {
		t.Fatalf("environment value was overwritten: %q", got)
	}
	if got := os.Getenv("SECOND_KEY"); got != "from-file" {
		t.Fatalf("file value was not loaded: %q", got)
	}
	t.Cleanup(func() { _ = os.Unsetenv("SECOND_KEY") })
}
