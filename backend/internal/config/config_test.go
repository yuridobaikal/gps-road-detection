package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMergesDefaultsFileAndEnv(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{
  "grpc_addr": "127.0.0.1:50055",
  "matcher_mode": "postgis",
  "database_url": "postgres://user:pass@localhost:5432/db?sslmode=disable",
  "road_table": "roads",
  "log_file": "",
  "trip_log_enabled": false,
  "trip_log_max_bytes": 12345
}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path, true)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.GRPCAddr != "127.0.0.1:50055" {
		t.Fatalf("GRPCAddr = %q", cfg.GRPCAddr)
	}
	if cfg.MatcherMode != "postgis" {
		t.Fatalf("MatcherMode = %q", cfg.MatcherMode)
	}
	if cfg.DatabaseURL == "" {
		t.Fatal("DatabaseURL is empty")
	}
	if cfg.RoadTable != "roads" {
		t.Fatalf("RoadTable = %q", cfg.RoadTable)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("LogLevel = %q, want env override debug", cfg.LogLevel)
	}
	if cfg.LogFile != "" {
		t.Fatalf("LogFile = %q, want empty", cfg.LogFile)
	}
	if cfg.TripLogEnabled {
		t.Fatal("TripLogEnabled = true, want false")
	}
	if cfg.TripLogMaxBytes != 12345 {
		t.Fatalf("TripLogMaxBytes = %d", cfg.TripLogMaxBytes)
	}
}

func TestLoadIgnoresMissingDefaultConfig(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.json"), false)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.GRPCAddr == "" {
		t.Fatal("expected defaults")
	}
}

func TestLoadFailsMissingExplicitConfig(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing.json"), true); err == nil {
		t.Fatal("expected missing explicit config error")
	}
}

func TestPathFromArgs(t *testing.T) {
	path, explicit := PathFromArgs([]string{"-config", "custom.json"})
	if path != "custom.json" || !explicit {
		t.Fatalf("PathFromArgs -config = %q, %t", path, explicit)
	}

	path, explicit = PathFromArgs([]string{"-config=other.json"})
	if path != "other.json" || !explicit {
		t.Fatalf("PathFromArgs -config= = %q, %t", path, explicit)
	}

	path, explicit = PathFromArgs(nil)
	if path != DefaultConfigPath || explicit {
		t.Fatalf("PathFromArgs default = %q, %t", path, explicit)
	}
}
