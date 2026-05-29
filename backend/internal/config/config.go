package config

import "os"

type Config struct {
	GRPCAddr    string
	MatcherMode string
	DatabaseURL string
	RoadTable   string
	LogLevel    string
}

func FromEnv() Config {
	return Config{
		GRPCAddr:    envOrDefault("GRPC_ADDR", ":50051"),
		MatcherMode: envOrDefault("MATCHER_MODE", "mock"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RoadTable:   envOrDefault("ROAD_TABLE", "planet_osm_line"),
		LogLevel:    envOrDefault("LOG_LEVEL", "info"),
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
