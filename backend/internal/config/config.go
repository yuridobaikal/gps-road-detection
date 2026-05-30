package config

import (
	"os"
	"strconv"
)

type Config struct {
	GRPCAddr        string
	MatcherMode     string
	DatabaseURL     string
	RoadTable       string
	LogLevel        string
	LogFile         string
	TripLogEnabled  bool
	TripLogDir      string
	TripLogMaxBytes int64
}

func FromEnv() Config {
	return Config{
		GRPCAddr:        envOrDefault("GRPC_ADDR", ":50051"),
		MatcherMode:     envOrDefault("MATCHER_MODE", "mock"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		RoadTable:       envOrDefault("ROAD_TABLE", "planet_osm_line"),
		LogLevel:        envOrDefault("LOG_LEVEL", "info"),
		LogFile:         envOrDefault("LOG_FILE", "logs/road-matcher.log"),
		TripLogEnabled:  boolEnvOrDefault("TRIP_LOG_ENABLED", true),
		TripLogDir:      envOrDefault("TRIP_LOG_DIR", "logs/trips"),
		TripLogMaxBytes: int64EnvOrDefault("TRIP_LOG_MAX_BYTES", 10*1024*1024),
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func boolEnvOrDefault(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func int64EnvOrDefault(key string, fallback int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
