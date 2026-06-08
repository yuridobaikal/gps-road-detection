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
  "same_road_hysteresis_margin": 0.15,
  "unconnected_jump_hysteresis_margin": 0.22,
  "history_bearing_min_distance_meters": 12,
  "history_bearing_max_age_seconds": 20,
  "history_bearing_min_speed_mps": 3,
  "distance_score_weight": 0.31,
  "heading_score_weight": 0.21,
  "connectivity_score_weight": 0.24,
  "road_class_score_weight": 0.09,
  "speed_score_weight": 0.17,
  "medium_speed_mps": 9,
  "high_speed_mps": 17,
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
	if cfg.SameRoadHysteresisMargin != 0.15 {
		t.Fatalf("SameRoadHysteresisMargin = %f", cfg.SameRoadHysteresisMargin)
	}
	if cfg.UnconnectedJumpHysteresisMargin != 0.22 {
		t.Fatalf("UnconnectedJumpHysteresisMargin = %f", cfg.UnconnectedJumpHysteresisMargin)
	}
	if cfg.HistoryBearingMinDistanceMeters != 12 {
		t.Fatalf("HistoryBearingMinDistanceMeters = %f", cfg.HistoryBearingMinDistanceMeters)
	}
	if cfg.HistoryBearingMaxAgeSeconds != 20 {
		t.Fatalf("HistoryBearingMaxAgeSeconds = %f", cfg.HistoryBearingMaxAgeSeconds)
	}
	if cfg.HistoryBearingMinSpeedMPS != 3 {
		t.Fatalf("HistoryBearingMinSpeedMPS = %f", cfg.HistoryBearingMinSpeedMPS)
	}
	if cfg.DistanceScoreWeight != 0.31 {
		t.Fatalf("DistanceScoreWeight = %f", cfg.DistanceScoreWeight)
	}
	if cfg.HeadingScoreWeight != 0.21 {
		t.Fatalf("HeadingScoreWeight = %f", cfg.HeadingScoreWeight)
	}
	if cfg.ConnectivityScoreWeight != 0.24 {
		t.Fatalf("ConnectivityScoreWeight = %f", cfg.ConnectivityScoreWeight)
	}
	if cfg.RoadClassScoreWeight != 0.09 {
		t.Fatalf("RoadClassScoreWeight = %f", cfg.RoadClassScoreWeight)
	}
	if cfg.SpeedScoreWeight != 0.17 {
		t.Fatalf("SpeedScoreWeight = %f", cfg.SpeedScoreWeight)
	}
	if cfg.MediumSpeedMPS != 9 {
		t.Fatalf("MediumSpeedMPS = %f", cfg.MediumSpeedMPS)
	}
	if cfg.HighSpeedMPS != 17 {
		t.Fatalf("HighSpeedMPS = %f", cfg.HighSpeedMPS)
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
	if cfg.SameRoadHysteresisMargin != 0.12 {
		t.Fatalf("SameRoadHysteresisMargin = %f", cfg.SameRoadHysteresisMargin)
	}
	if cfg.UnconnectedJumpHysteresisMargin != 0.18 {
		t.Fatalf("UnconnectedJumpHysteresisMargin = %f", cfg.UnconnectedJumpHysteresisMargin)
	}
	if cfg.HistoryBearingMinDistanceMeters != 8 {
		t.Fatalf("HistoryBearingMinDistanceMeters = %f", cfg.HistoryBearingMinDistanceMeters)
	}
	if cfg.HistoryBearingMaxAgeSeconds != 15 {
		t.Fatalf("HistoryBearingMaxAgeSeconds = %f", cfg.HistoryBearingMaxAgeSeconds)
	}
	if cfg.HistoryBearingMinSpeedMPS != 2 {
		t.Fatalf("HistoryBearingMinSpeedMPS = %f", cfg.HistoryBearingMinSpeedMPS)
	}
	if cfg.DistanceScoreWeight != 0.32 {
		t.Fatalf("DistanceScoreWeight = %f", cfg.DistanceScoreWeight)
	}
	if cfg.HeadingScoreWeight != 0.23 {
		t.Fatalf("HeadingScoreWeight = %f", cfg.HeadingScoreWeight)
	}
	if cfg.ConnectivityScoreWeight != 0.25 {
		t.Fatalf("ConnectivityScoreWeight = %f", cfg.ConnectivityScoreWeight)
	}
	if cfg.RoadClassScoreWeight != 0.10 {
		t.Fatalf("RoadClassScoreWeight = %f", cfg.RoadClassScoreWeight)
	}
	if cfg.SpeedScoreWeight != 0.10 {
		t.Fatalf("SpeedScoreWeight = %f", cfg.SpeedScoreWeight)
	}
	if cfg.MediumSpeedMPS != 8 {
		t.Fatalf("MediumSpeedMPS = %f", cfg.MediumSpeedMPS)
	}
	if cfg.HighSpeedMPS != 15 {
		t.Fatalf("HighSpeedMPS = %f", cfg.HighSpeedMPS)
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
