package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"YANDEX_CLIENT_ID", "YANDEX_CLIENT_SECRET", "PORT", "DATA_DIR", "HOST", "ALLOWED_HOSTS", "POLL_SECONDS", "RULES_ENABLED"} {
		t.Setenv(k, "")
	}
}

func TestLoadReadsDotEnvAndDefaults(t *testing.T) {
	clearEnv(t)
	path := filepath.Join(t.TempDir(), ".env")
	content := "# комментарий\nYANDEX_CLIENT_ID=id1\nYANDEX_CLIENT_SECRET=\"sec1\"\n\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ClientID != "id1" || cfg.ClientSecret != "sec1" {
		t.Errorf("creds = %q/%q, want id1/sec1", cfg.ClientID, cfg.ClientSecret)
	}
	if cfg.Port != "8080" || cfg.DataDir != "data" {
		t.Errorf("defaults = port %q, dataDir %q", cfg.Port, cfg.DataDir)
	}
}

func TestLoadEnvOverridesFile(t *testing.T) {
	clearEnv(t)
	path := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(path, []byte("YANDEX_CLIENT_ID=file\nYANDEX_CLIENT_SECRET=s\nPORT=1111\n"), 0o600)
	t.Setenv("YANDEX_CLIENT_ID", "env")
	t.Setenv("PORT", "9090")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ClientID != "env" || cfg.Port != "9090" {
		t.Errorf("got clientID %q port %q, want env/9090", cfg.ClientID, cfg.Port)
	}
}

func TestLoadMissingFileIsFine(t *testing.T) {
	clearEnv(t)
	t.Setenv("YANDEX_CLIENT_ID", "id")
	t.Setenv("YANDEX_CLIENT_SECRET", "sec")

	cfg, err := Load(filepath.Join(t.TempDir(), "nope.env"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ClientID != "id" {
		t.Errorf("ClientID = %q", cfg.ClientID)
	}
}

func TestLoadFailsWithoutCredentials(t *testing.T) {
	clearEnv(t)
	if _, err := Load(filepath.Join(t.TempDir(), "nope.env")); err == nil {
		t.Fatal("ожидалась ошибка без YANDEX_CLIENT_ID/SECRET")
	}
}

func TestLoadRejectsMalformedLine(t *testing.T) {
	clearEnv(t)
	path := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(path, []byte("YANDEX_CLIENT_ID id\n"), 0o600)
	if _, err := Load(path); err == nil {
		t.Fatal("ожидалась ошибка на строке без '='")
	}
}

func TestLoadHostAndAllowedHostsDefaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("YANDEX_CLIENT_ID", "id")
	t.Setenv("YANDEX_CLIENT_SECRET", "sec")

	cfg, err := Load(filepath.Join(t.TempDir(), "nope.env"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Host != "127.0.0.1" {
		t.Errorf("Host = %q, want 127.0.0.1", cfg.Host)
	}
	if len(cfg.AllowedHosts) != 0 {
		t.Errorf("AllowedHosts = %v, want empty", cfg.AllowedHosts)
	}
}

func TestLoadParsesAllowedHosts(t *testing.T) {
	clearEnv(t)
	t.Setenv("YANDEX_CLIENT_ID", "id")
	t.Setenv("YANDEX_CLIENT_SECRET", "sec")
	t.Setenv("ALLOWED_HOSTS", "192.168.1.95, smart-house.local")

	cfg, err := Load(filepath.Join(t.TempDir(), "nope.env"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []string{"192.168.1.95", "smart-house.local"}
	if !reflect.DeepEqual(cfg.AllowedHosts, want) {
		t.Errorf("AllowedHosts = %v, want %v", cfg.AllowedHosts, want)
	}
}

func TestLoadPollAndRules(t *testing.T) {
	clearEnv(t)
	t.Setenv("YANDEX_CLIENT_ID", "id")
	t.Setenv("YANDEX_CLIENT_SECRET", "sec")
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.env"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PollSeconds != 10 || !cfg.RulesEnabled {
		t.Errorf("defaults: poll=%d rules=%v", cfg.PollSeconds, cfg.RulesEnabled)
	}
	t.Setenv("POLL_SECONDS", "1")
	t.Setenv("RULES_ENABLED", "false")
	cfg, _ = Load(filepath.Join(t.TempDir(), "nope.env"))
	if cfg.PollSeconds != 3 || cfg.RulesEnabled {
		t.Errorf("overrides: poll=%d (min 3) rules=%v", cfg.PollSeconds, cfg.RulesEnabled)
	}
}
