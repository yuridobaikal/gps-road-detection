package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/yurido/gps-road-detection/backend/gen/roadmatcherpb"
	"github.com/yurido/gps-road-detection/backend/internal/config"
	"github.com/yurido/gps-road-detection/backend/internal/matching"
	"github.com/yurido/gps-road-detection/backend/internal/triplog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type roadMatcherServer struct {
	pb.UnimplementedRoadMatcherServer
	matcher    matching.Matcher
	logLevel   string
	tripLogger *triplog.Logger
}

func (s *roadMatcherServer) MatchRoad(ctx context.Context, req *pb.MatchRoadRequest) (*pb.MatchRoadResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.logRequest("unary", req)
	start := time.Now()
	resp, err := s.matcher.Match(ctx, req)
	duration := time.Since(start)
	setProcessingDuration(resp, duration)
	s.logTrip("unary", req, resp, duration, err)
	if err != nil {
		log.Printf("unary error sequence=%d device=%s: %v", req.GetSequenceId(), req.GetDeviceId(), err)
		return nil, err
	}
	s.logResponse("unary", resp)
	return resp, nil
}

func (s *roadMatcherServer) StreamGps(stream pb.RoadMatcher_StreamGpsServer) error {
	ctx := stream.Context()

	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		s.logRequest("stream", req)
		start := time.Now()
		resp, err := s.matcher.Match(ctx, req)
		duration := time.Since(start)
		setProcessingDuration(resp, duration)
		s.logTrip("stream", req, resp, duration, err)
		if err != nil {
			log.Printf("stream error sequence=%d device=%s: %v", req.GetSequenceId(), req.GetDeviceId(), err)
			return err
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
		s.logResponse("stream", resp)
	}
}

func (s *roadMatcherServer) debugEnabled() bool {
	return s.logLevel == "debug"
}

func setProcessingDuration(resp *pb.MatchRoadResponse, duration time.Duration) {
	if resp == nil {
		return
	}
	resp.ProcessingDurationMs = float64(duration.Microseconds()) / 1000.0
}

func (s *roadMatcherServer) logTrip(kind string, req *pb.MatchRoadRequest, resp *pb.MatchRoadResponse, duration time.Duration, matchErr error) {
	if s.tripLogger == nil {
		return
	}
	if err := s.tripLogger.LogMatch(kind, req, resp, duration, matchErr); err != nil {
		log.Printf("write trip log sequence=%d device=%s: %v", req.GetSequenceId(), req.GetDeviceId(), err)
	}
}

func (s *roadMatcherServer) logRequest(kind string, req *pb.MatchRoadRequest) {
	if !s.debugEnabled() {
		return
	}
	log.Printf(
		"%s request sequence=%d device=%s lat=%.6f lon=%.6f accuracy=%.1f speed=%.1f heading=%.1f last_road_id=%d history=%d",
		kind,
		req.GetSequenceId(),
		req.GetDeviceId(),
		req.GetLat(),
		req.GetLon(),
		req.GetAccuracy(),
		req.GetSpeed(),
		req.GetHeading(),
		req.GetLastRoadId(),
		len(req.GetHistory()),
	)
}

func (s *roadMatcherServer) logResponse(kind string, resp *pb.MatchRoadResponse) {
	if !s.debugEnabled() {
		return
	}
	best := resp.GetBest()
	bestName := ""
	bestRoadID := int64(0)
	if best != nil {
		bestName = best.GetName()
		bestRoadID = best.GetRoadId()
	}
	log.Printf(
		"%s response sequence=%d matched=%t confidence=%s best_road_id=%d best=%q candidates=%d processing_ms=%.1f",
		kind,
		resp.GetSequenceId(),
		resp.GetMatched(),
		resp.GetConfidence(),
		bestRoadID,
		bestName,
		len(resp.GetCandidates()),
		resp.GetProcessingDurationMs(),
	)
	for i, candidate := range resp.GetCandidates() {
		log.Printf(
			"%s candidate[%d] road_id=%d name=%q score=%.3f",
			kind,
			i,
			candidate.GetRoadId(),
			candidate.GetName(),
			candidate.GetScore(),
		)
	}
}

func main() {
	configPath, configExplicit := config.PathFromArgs(os.Args[1:])
	cfg, err := config.Load(configPath, configExplicit)
	if err != nil {
		log.Fatal(err)
	}
	flag.StringVar(&configPath, "config", configPath, "config file path")
	flag.StringVar(&cfg.GRPCAddr, "addr", cfg.GRPCAddr, "gRPC listen address")
	flag.StringVar(&cfg.MatcherMode, "matcher-mode", cfg.MatcherMode, "matcher mode: mock or postgis")
	flag.StringVar(&cfg.DatabaseURL, "database-url", cfg.DatabaseURL, "PostgreSQL connection URL for postgis mode")
	flag.StringVar(&cfg.RoadTable, "road-table", cfg.RoadTable, "road table: planet_osm_line, planet_osm_roads, or roads")
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "log level: info or debug")
	flag.StringVar(&cfg.LogFile, "log-file", cfg.LogFile, "common backend log file; empty disables file logging")
	flag.BoolVar(&cfg.TripLogEnabled, "trip-log-enabled", cfg.TripLogEnabled, "write replayable trip JSONL logs")
	flag.StringVar(&cfg.TripLogDir, "trip-log-dir", cfg.TripLogDir, "directory for trip JSONL logs")
	flag.Int64Var(&cfg.TripLogMaxBytes, "trip-log-max-bytes", cfg.TripLogMaxBytes, "rotate and gzip trip logs after this many bytes")
	flag.Parse()

	commonLogFile, err := setupCommonLog(cfg.LogFile)
	if err != nil {
		log.Fatalf("setup common log file: %v", err)
	}
	if commonLogFile != nil {
		defer commonLogFile.Close()
	}

	ctx := context.Background()
	matcher, cleanup := buildMatcher(ctx, cfg)
	defer cleanup()

	var tripLogger *triplog.Logger
	if cfg.TripLogEnabled {
		var err error
		tripLogger, err = triplog.New(cfg.TripLogDir, cfg.TripLogMaxBytes)
		if err != nil {
			log.Fatalf("create trip logger: %v", err)
		}
		defer tripLogger.Close()
	}

	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("listen on %s: %v", cfg.GRPCAddr, err)
	}

	server := grpc.NewServer()
	pb.RegisterRoadMatcherServer(server, &roadMatcherServer{
		matcher:    matcher,
		logLevel:   cfg.LogLevel,
		tripLogger: tripLogger,
	})
	reflection.Register(server)

	go func() {
		log.Printf("road matcher gRPC server listening on %s config=%s mode=%s road_table=%s log_level=%s log_file=%s trip_log_enabled=%t trip_log_dir=%s trip_log_max_bytes=%d", cfg.GRPCAddr, configPath, cfg.MatcherMode, cfg.RoadTable, cfg.LogLevel, cfg.LogFile, cfg.TripLogEnabled, cfg.TripLogDir, cfg.TripLogMaxBytes)
		if err := server.Serve(listener); err != nil {
			log.Fatalf("serve grpc: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Print("shutting down road matcher gRPC server")
	done := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		server.Stop()
	}
}

func setupCommonLog(path string) (*os.File, error) {
	if path == "" {
		return nil, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	log.SetOutput(file)
	return file, nil
}

func buildMatcher(ctx context.Context, cfg config.Config) (matching.Matcher, func()) {
	switch cfg.MatcherMode {
	case "mock":
		return matching.NewMockMatcher(), func() {}
	case "postgis":
		if cfg.DatabaseURL == "" {
			log.Fatal("DATABASE_URL or -database-url is required when matcher mode is postgis")
		}

		pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("create postgis pool: %v", err)
		}
		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			log.Fatalf("ping postgis database: %v", err)
		}

		matcher, err := matching.NewPostGISMatcher(pool, cfg.RoadTable)
		if err != nil {
			pool.Close()
			log.Fatal(err)
		}
		return matcher, pool.Close
	default:
		log.Fatalf("unknown matcher mode %q; use mock or postgis", cfg.MatcherMode)
		return nil, nil
	}
}
