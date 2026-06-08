package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

const DefaultConfigPath = "config.local.json"

type Config struct {
	GRPCAddr                        string  `json:"grpc_addr"`
	MatcherMode                     string  `json:"matcher_mode"`
	DatabaseURL                     string  `json:"database_url"`
	RoadTable                       string  `json:"road_table"`
	LogLevel                        string  `json:"log_level"`
	LogFile                         string  `json:"log_file"`
	SameRoadHysteresisMargin        float64 `json:"same_road_hysteresis_margin"`
	UnconnectedJumpHysteresisMargin float64 `json:"unconnected_jump_hysteresis_margin"`
	HistoryBearingMinDistanceMeters float64 `json:"history_bearing_min_distance_meters"`
	HistoryBearingMaxAgeSeconds     float64 `json:"history_bearing_max_age_seconds"`
	HistoryBearingMinSpeedMPS       float64 `json:"history_bearing_min_speed_mps"`
	DistanceScoreWeight             float64 `json:"distance_score_weight"`
	HeadingScoreWeight              float64 `json:"heading_score_weight"`
	ConnectivityScoreWeight         float64 `json:"connectivity_score_weight"`
	RoadClassScoreWeight            float64 `json:"road_class_score_weight"`
	SpeedScoreWeight                float64 `json:"speed_score_weight"`
	MediumSpeedMPS                  float64 `json:"medium_speed_mps"`
	HighSpeedMPS                    float64 `json:"high_speed_mps"`
	TripLogEnabled                  bool    `json:"trip_log_enabled"`
	TripLogDir                      string  `json:"trip_log_dir"`
	TripLogMaxBytes                 int64   `json:"trip_log_max_bytes"`
}

type fileConfig struct {
	GRPCAddr                        *string  `json:"grpc_addr"`
	MatcherMode                     *string  `json:"matcher_mode"`
	DatabaseURL                     *string  `json:"database_url"`
	RoadTable                       *string  `json:"road_table"`
	LogLevel                        *string  `json:"log_level"`
	LogFile                         *string  `json:"log_file"`
	SameRoadHysteresisMargin        *float64 `json:"same_road_hysteresis_margin"`
	UnconnectedJumpHysteresisMargin *float64 `json:"unconnected_jump_hysteresis_margin"`
	HistoryBearingMinDistanceMeters *float64 `json:"history_bearing_min_distance_meters"`
	HistoryBearingMaxAgeSeconds     *float64 `json:"history_bearing_max_age_seconds"`
	HistoryBearingMinSpeedMPS       *float64 `json:"history_bearing_min_speed_mps"`
	DistanceScoreWeight             *float64 `json:"distance_score_weight"`
	HeadingScoreWeight              *float64 `json:"heading_score_weight"`
	ConnectivityScoreWeight         *float64 `json:"connectivity_score_weight"`
	RoadClassScoreWeight            *float64 `json:"road_class_score_weight"`
	SpeedScoreWeight                *float64 `json:"speed_score_weight"`
	MediumSpeedMPS                  *float64 `json:"medium_speed_mps"`
	HighSpeedMPS                    *float64 `json:"high_speed_mps"`
	TripLogEnabled                  *bool    `json:"trip_log_enabled"`
	TripLogDir                      *string  `json:"trip_log_dir"`
	TripLogMaxBytes                 *int64   `json:"trip_log_max_bytes"`
}

func Defaults() Config {
	return Config{
		GRPCAddr:                        ":50051",
		MatcherMode:                     "mock",
		RoadTable:                       "planet_osm_line",
		LogLevel:                        "info",
		LogFile:                         "logs/road-matcher.log",
		SameRoadHysteresisMargin:        0.12,
		UnconnectedJumpHysteresisMargin: 0.18,
		HistoryBearingMinDistanceMeters: 8,
		HistoryBearingMaxAgeSeconds:     15,
		HistoryBearingMinSpeedMPS:       2,
		DistanceScoreWeight:             0.32,
		HeadingScoreWeight:              0.23,
		ConnectivityScoreWeight:         0.25,
		RoadClassScoreWeight:            0.10,
		SpeedScoreWeight:                0.10,
		MediumSpeedMPS:                  8,
		HighSpeedMPS:                    15,
		TripLogEnabled:                  true,
		TripLogDir:                      "logs/trips",
		TripLogMaxBytes:                 10 * 1024 * 1024,
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
	if fileConfig.SameRoadHysteresisMargin != nil && *fileConfig.SameRoadHysteresisMargin >= 0 {
		c.SameRoadHysteresisMargin = *fileConfig.SameRoadHysteresisMargin
	}
	if fileConfig.UnconnectedJumpHysteresisMargin != nil && *fileConfig.UnconnectedJumpHysteresisMargin >= 0 {
		c.UnconnectedJumpHysteresisMargin = *fileConfig.UnconnectedJumpHysteresisMargin
	}
	if fileConfig.HistoryBearingMinDistanceMeters != nil && *fileConfig.HistoryBearingMinDistanceMeters > 0 {
		c.HistoryBearingMinDistanceMeters = *fileConfig.HistoryBearingMinDistanceMeters
	}
	if fileConfig.HistoryBearingMaxAgeSeconds != nil && *fileConfig.HistoryBearingMaxAgeSeconds > 0 {
		c.HistoryBearingMaxAgeSeconds = *fileConfig.HistoryBearingMaxAgeSeconds
	}
	if fileConfig.HistoryBearingMinSpeedMPS != nil && *fileConfig.HistoryBearingMinSpeedMPS >= 0 {
		c.HistoryBearingMinSpeedMPS = *fileConfig.HistoryBearingMinSpeedMPS
	}
	if fileConfig.DistanceScoreWeight != nil && *fileConfig.DistanceScoreWeight >= 0 {
		c.DistanceScoreWeight = *fileConfig.DistanceScoreWeight
	}
	if fileConfig.HeadingScoreWeight != nil && *fileConfig.HeadingScoreWeight >= 0 {
		c.HeadingScoreWeight = *fileConfig.HeadingScoreWeight
	}
	if fileConfig.ConnectivityScoreWeight != nil && *fileConfig.ConnectivityScoreWeight >= 0 {
		c.ConnectivityScoreWeight = *fileConfig.ConnectivityScoreWeight
	}
	if fileConfig.RoadClassScoreWeight != nil && *fileConfig.RoadClassScoreWeight >= 0 {
		c.RoadClassScoreWeight = *fileConfig.RoadClassScoreWeight
	}
	if fileConfig.SpeedScoreWeight != nil && *fileConfig.SpeedScoreWeight >= 0 {
		c.SpeedScoreWeight = *fileConfig.SpeedScoreWeight
	}
	if fileConfig.MediumSpeedMPS != nil && *fileConfig.MediumSpeedMPS >= 0 {
		c.MediumSpeedMPS = *fileConfig.MediumSpeedMPS
	}
	if fileConfig.HighSpeedMPS != nil && *fileConfig.HighSpeedMPS > 0 {
		c.HighSpeedMPS = *fileConfig.HighSpeedMPS
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
	if value := os.Getenv("SAME_ROAD_HYSTERESIS_MARGIN"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil && parsed >= 0 {
			c.SameRoadHysteresisMargin = parsed
		}
	}
	if value := os.Getenv("UNCONNECTED_JUMP_HYSTERESIS_MARGIN"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil && parsed >= 0 {
			c.UnconnectedJumpHysteresisMargin = parsed
		}
	}
	if value := os.Getenv("HISTORY_BEARING_MIN_DISTANCE_METERS"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil && parsed > 0 {
			c.HistoryBearingMinDistanceMeters = parsed
		}
	}
	if value := os.Getenv("HISTORY_BEARING_MAX_AGE_SECONDS"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil && parsed > 0 {
			c.HistoryBearingMaxAgeSeconds = parsed
		}
	}
	if value := os.Getenv("HISTORY_BEARING_MIN_SPEED_MPS"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil && parsed >= 0 {
			c.HistoryBearingMinSpeedMPS = parsed
		}
	}
	if value := os.Getenv("DISTANCE_SCORE_WEIGHT"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil && parsed >= 0 {
			c.DistanceScoreWeight = parsed
		}
	}
	if value := os.Getenv("HEADING_SCORE_WEIGHT"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil && parsed >= 0 {
			c.HeadingScoreWeight = parsed
		}
	}
	if value := os.Getenv("CONNECTIVITY_SCORE_WEIGHT"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil && parsed >= 0 {
			c.ConnectivityScoreWeight = parsed
		}
	}
	if value := os.Getenv("ROAD_CLASS_SCORE_WEIGHT"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil && parsed >= 0 {
			c.RoadClassScoreWeight = parsed
		}
	}
	if value := os.Getenv("SPEED_SCORE_WEIGHT"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil && parsed >= 0 {
			c.SpeedScoreWeight = parsed
		}
	}
	if value := os.Getenv("MEDIUM_SPEED_MPS"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil && parsed >= 0 {
			c.MediumSpeedMPS = parsed
		}
	}
	if value := os.Getenv("HIGH_SPEED_MPS"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil && parsed > 0 {
			c.HighSpeedMPS = parsed
		}
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
