package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

const DefaultConfigPath = "config.local.json"

type Config struct {
	GRPCAddr        string `json:"grpc_addr"`
	MatcherMode     string `json:"matcher_mode"`
	DatabaseURL     string `json:"database_url"`
	RoadTable       string `json:"road_table"`
	LogLevel        string `json:"log_level"`
	LogFile         string `json:"log_file"`
	TripLogEnabled  bool   `json:"trip_log_enabled"`
	TripLogDir      string `json:"trip_log_dir"`
	TripLogMaxBytes int64  `json:"trip_log_max_bytes"`
}

type fileConfig struct {
	GRPCAddr        *string `json:"grpc_addr"`
	MatcherMode     *string `json:"matcher_mode"`
	DatabaseURL     *string `json:"database_url"`
	RoadTable       *string `json:"road_table"`
	LogLevel        *string `json:"log_level"`
	LogFile         *string `json:"log_file"`
	TripLogEnabled  *bool   `json:"trip_log_enabled"`
	TripLogDir      *string `json:"trip_log_dir"`
	TripLogMaxBytes *int64  `json:"trip_log_max_bytes"`
}

func Defaults() Config {
	return Config{
		GRPCAddr:        ":50051",
		MatcherMode:     "mock",
		RoadTable:       "planet_osm_line",
		LogLevel:        "info",
		LogFile:         "logs/road-matcher.log",
		TripLogEnabled:  true,
		TripLogDir:      "logs/trips",
		TripLogMaxBytes: 10 * 1024 * 1024,
	}
}

func Load(path string, explicit bool) (Config, error) {
	cfg := Defaults()
	if path != "" {
		if err := cfg.ApplyFile(path, explicit); err != nil {
			return Config{}, err
		}
	}
	cfg.ApplyEnv()
	return cfg, nil
}

func (c *Config) ApplyFile(path string, explicit bool) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && !explicit {
			return nil
		}
		return fmt.Errorf("read config file %q: %w", path, err)
	}

	var fileConfig fileConfig
	if err := json.Unmarshal(bytes, &fileConfig); err != nil {
		return fmt.Errorf("parse config file %q: %w", path, err)
	}

	if fileConfig.GRPCAddr != nil {
		c.GRPCAddr = *fileConfig.GRPCAddr
	}
	if fileConfig.MatcherMode != nil {
		c.MatcherMode = *fileConfig.MatcherMode
	}
	if fileConfig.DatabaseURL != nil {
		c.DatabaseURL = *fileConfig.DatabaseURL
	}
	if fileConfig.RoadTable != nil {
		c.RoadTable = *fileConfig.RoadTable
	}
	if fileConfig.LogLevel != nil {
		c.LogLevel = *fileConfig.LogLevel
	}
	if fileConfig.LogFile != nil {
		c.LogFile = *fileConfig.LogFile
	}
	if fileConfig.TripLogEnabled != nil {
		c.TripLogEnabled = *fileConfig.TripLogEnabled
	}
	if fileConfig.TripLogDir != nil {
		c.TripLogDir = *fileConfig.TripLogDir
	}
	if fileConfig.TripLogMaxBytes != nil && *fileConfig.TripLogMaxBytes > 0 {
		c.TripLogMaxBytes = *fileConfig.TripLogMaxBytes
	}
	return nil
}

func (c *Config) ApplyEnv() {
	if value := os.Getenv("GRPC_ADDR"); value != "" {
		c.GRPCAddr = value
	}
	if value := os.Getenv("MATCHER_MODE"); value != "" {
		c.MatcherMode = value
	}
	if value := os.Getenv("DATABASE_URL"); value != "" {
		c.DatabaseURL = value
	}
	if value := os.Getenv("ROAD_TABLE"); value != "" {
		c.RoadTable = value
	}
	if value := os.Getenv("LOG_LEVEL"); value != "" {
		c.LogLevel = value
	}
	if value := os.Getenv("LOG_FILE"); value != "" {
		c.LogFile = value
	}
	if value := os.Getenv("TRIP_LOG_ENABLED"); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			c.TripLogEnabled = parsed
		}
	}
	if value := os.Getenv("TRIP_LOG_DIR"); value != "" {
		c.TripLogDir = value
	}
	if value := os.Getenv("TRIP_LOG_MAX_BYTES"); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil && parsed > 0 {
			c.TripLogMaxBytes = parsed
		}
	}
}

func PathFromArgs(args []string) (string, bool) {
	for i, arg := range args {
		if arg == "-config" && i+1 < len(args) {
			return args[i+1], true
		}
		const prefix = "-config="
		if len(arg) > len(prefix) && arg[:len(prefix)] == prefix {
			return arg[len(prefix):], true
		}
	}
	return DefaultConfigPath, false
}
