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
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/yurido/gps-road-detection/backend/gen/roadmatcherpb"
	"github.com/yurido/gps-road-detection/backend/internal/config"
	"github.com/yurido/gps-road-detection/backend/internal/matching"
	"github.com/yurido/gps-road-detection/backend/internal/triplog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
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
	s.debugLogf("unary match start sequence=%d device=%s", req.GetSequenceId(), req.GetDeviceId())
	resp, err := s.matcher.Match(ctx, req)
	duration := time.Since(start)
	setProcessingDuration(resp, duration)
	s.debugLogf("unary match done sequence=%d device=%s match_ms=%.1f err=%v", req.GetSequenceId(), req.GetDeviceId(), durationMs(duration), err)
	tripStart := time.Now()
	s.logTrip("unary", req, resp, duration, err)
	s.debugLogf("unary trip log done sequence=%d device=%s trip_log_ms=%.1f", req.GetSequenceId(), req.GetDeviceId(), elapsedMs(tripStart))
	if err != nil {
		log.Printf("unary error sequence=%d device=%s: %v", req.GetSequenceId(), req.GetDeviceId(), err)
		return nil, err
	}
	clientResp := clientResponse(resp)
	s.logResponse("unary", clientResp)
	s.debugLogf("unary response ready sequence=%d device=%s total_ms=%.1f", req.GetSequenceId(), req.GetDeviceId(), elapsedMs(start))
	return clientResp, nil
}

func (s *roadMatcherServer) StreamGps(stream pb.RoadMatcher_StreamGpsServer) error {
	ctx := stream.Context()

	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			s.debugLogf("stream closed by client")
			return nil
		}
		if err != nil {
			log.Printf("stream receive error: %v", err)
			return err
		}

		receivedAt := time.Now()
		s.debugLogf("stream request received sequence=%d device=%s", req.GetSequenceId(), req.GetDeviceId())
		s.logRequest("stream", req)
		start := time.Now()
		s.debugLogf("stream match start sequence=%d device=%s receive_to_match_ms=%.1f", req.GetSequenceId(), req.GetDeviceId(), elapsedMs(receivedAt))
		resp, err := s.matcher.Match(ctx, req)
		duration := time.Since(start)
		setProcessingDuration(resp, duration)
		s.debugLogf("stream match done sequence=%d device=%s match_ms=%.1f err=%v", req.GetSequenceId(), req.GetDeviceId(), durationMs(duration), err)
		tripStart := time.Now()
		s.logTrip("stream", req, resp, duration, err)
		s.debugLogf("stream trip log done sequence=%d device=%s trip_log_ms=%.1f", req.GetSequenceId(), req.GetDeviceId(), elapsedMs(tripStart))
		if err != nil {
			log.Printf("stream error sequence=%d device=%s: %v", req.GetSequenceId(), req.GetDeviceId(), err)
			return err
		}
		clientResp := clientResponse(resp)
		sendStart := time.Now()
		s.debugLogf("stream send start sequence=%d device=%s", req.GetSequenceId(), req.GetDeviceId())
		if err := stream.Send(clientResp); err != nil {
			log.Printf("stream send error sequence=%d device=%s: %v", req.GetSequenceId(), req.GetDeviceId(), err)
			return err
		}
		s.debugLogf("stream send done sequence=%d device=%s send_ms=%.1f total_ms=%.1f", req.GetSequenceId(), req.GetDeviceId(), elapsedMs(sendStart), elapsedMs(receivedAt))
		s.logResponse("stream", clientResp)
	}
}

func (s *roadMatcherServer) debugEnabled() bool {
	return s.logLevel == "debug"
}

func (s *roadMatcherServer) debugLogf(format string, args ...any) {
	if !s.debugEnabled() {
		return
	}
	log.Printf(format, args...)
}

func elapsedMs(start time.Time) float64 {
	return float64(time.Since(start).Microseconds()) / 1000.0
}

func durationMs(duration time.Duration) float64 {
	return float64(duration.Microseconds()) / 1000.0
}

func setProcessingDuration(resp *pb.MatchRoadResponse, duration time.Duration) {
	if resp == nil {
		return
	}
	resp.ProcessingDurationMs = float64(duration.Microseconds()) / 1000.0
}

func clientResponse(resp *pb.MatchRoadResponse) *pb.MatchRoadResponse {
	if resp == nil {
		return nil
	}
	return &pb.MatchRoadResponse{
		SequenceId:           resp.GetSequenceId(),
		Matched:              resp.GetMatched(),
		Confidence:           resp.GetConfidence(),
		Best:                 clientCandidate(resp.GetBest()),
		Candidates:           clientCandidates(resp.GetCandidates()),
		ProcessingDurationMs: resp.GetProcessingDurationMs(),
	}
}

func clientCandidate(candidate *pb.RoadCandidate) *pb.RoadCandidate {
	if candidate == nil {
		return nil
	}
	return &pb.RoadCandidate{
		RoadId: candidate.GetRoadId(),
		Name:   candidate.GetName(),
		Score:  candidate.GetScore(),
	}
}

func clientCandidates(candidates []*pb.RoadCandidate) []*pb.RoadCandidate {
	trimmed := make([]*pb.RoadCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		trimmed = append(trimmed, clientCandidate(candidate))
	}
	return trimmed
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
	flag.Float64Var(&cfg.SameRoadHysteresisMargin, "same-road-hysteresis-margin", cfg.SameRoadHysteresisMargin, "score margin for keeping the same road")
	flag.Float64Var(&cfg.UnconnectedJumpHysteresisMargin, "unconnected-jump-hysteresis-margin", cfg.UnconnectedJumpHysteresisMargin, "score margin for avoiding jumps to unconnected roads")
	flag.Float64Var(&cfg.HistoryBearingMinDistanceMeters, "history-bearing-min-distance-meters", cfg.HistoryBearingMinDistanceMeters, "minimum GPS history movement distance to use history bearing")
	flag.Float64Var(&cfg.HistoryBearingMaxAgeSeconds, "history-bearing-max-age-seconds", cfg.HistoryBearingMaxAgeSeconds, "maximum age of GPS history point to use history bearing")
	flag.Float64Var(&cfg.HistoryBearingMinSpeedMPS, "history-bearing-min-speed-mps", cfg.HistoryBearingMinSpeedMPS, "minimum speed to use history bearing")
	flag.Float64Var(&cfg.DistanceScoreWeight, "distance-score-weight", cfg.DistanceScoreWeight, "distance score weight")
	flag.Float64Var(&cfg.HeadingScoreWeight, "heading-score-weight", cfg.HeadingScoreWeight, "heading score weight")
	flag.Float64Var(&cfg.ConnectivityScoreWeight, "connectivity-score-weight", cfg.ConnectivityScoreWeight, "connectivity score weight")
	flag.Float64Var(&cfg.RoadClassScoreWeight, "road-class-score-weight", cfg.RoadClassScoreWeight, "road class score weight")
	flag.Float64Var(&cfg.SpeedScoreWeight, "speed-score-weight", cfg.SpeedScoreWeight, "speed-aware road class score weight")
	flag.Float64Var(&cfg.MediumSpeedMPS, "medium-speed-mps", cfg.MediumSpeedMPS, "speed threshold for medium speed road scoring")
	flag.Float64Var(&cfg.HighSpeedMPS, "high-speed-mps", cfg.HighSpeedMPS, "speed threshold for high speed road scoring")
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
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	defer recoverAndExit("main")
	log.Printf("road matcher process starting pid=%d", os.Getpid())

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

	server := grpc.NewServer(
		grpc.UnaryInterceptor(recoverUnaryInterceptor),
		grpc.StreamInterceptor(recoverStreamInterceptor),
	)
	pb.RegisterRoadMatcherServer(server, &roadMatcherServer{
		matcher:    matcher,
		logLevel:   cfg.LogLevel,
		tripLogger: tripLogger,
	})
	reflection.Register(server)

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("road matcher gRPC server listening on %s config=%s mode=%s road_table=%s log_level=%s log_file=%s same_road_hysteresis_margin=%.3f unconnected_jump_hysteresis_margin=%.3f history_bearing_min_distance_meters=%.1f history_bearing_max_age_seconds=%.1f history_bearing_min_speed_mps=%.1f distance_score_weight=%.3f heading_score_weight=%.3f connectivity_score_weight=%.3f road_class_score_weight=%.3f speed_score_weight=%.3f medium_speed_mps=%.1f high_speed_mps=%.1f trip_log_enabled=%t trip_log_dir=%s trip_log_max_bytes=%d", cfg.GRPCAddr, configPath, cfg.MatcherMode, cfg.RoadTable, cfg.LogLevel, cfg.LogFile, cfg.SameRoadHysteresisMargin, cfg.UnconnectedJumpHysteresisMargin, cfg.HistoryBearingMinDistanceMeters, cfg.HistoryBearingMaxAgeSeconds, cfg.HistoryBearingMinSpeedMPS, cfg.DistanceScoreWeight, cfg.HeadingScoreWeight, cfg.ConnectivityScoreWeight, cfg.RoadClassScoreWeight, cfg.SpeedScoreWeight, cfg.MediumSpeedMPS, cfg.HighSpeedMPS, cfg.TripLogEnabled, cfg.TripLogDir, cfg.TripLogMaxBytes)
		if err := server.Serve(listener); err != nil {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()
	heartbeatStop := startHeartbeat()
	defer heartbeatStop()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	select {
	case sig := <-stop:
		log.Printf("received signal=%s", sig)
	case err := <-serveErr:
		if err != nil {
			log.Printf("serve grpc stopped with error: %v", err)
		} else {
			log.Print("serve grpc stopped")
		}
		return
	}

	log.Print("shutting down road matcher gRPC server")
	done := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		log.Print("road matcher gRPC server stopped gracefully")
	case <-time.After(5 * time.Second):
		log.Print("graceful shutdown timed out; forcing gRPC server stop")
		server.Stop()
	}
	log.Print("road matcher process stopped")
}

func recoverUnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("panic in unary method=%s panic=%v\n%s", info.FullMethod, recovered, debug.Stack())
			err = status.Error(codes.Internal, "internal server error")
		}
	}()
	return handler(ctx, req)
}

func recoverStreamInterceptor(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("panic in stream method=%s panic=%v\n%s", info.FullMethod, recovered, debug.Stack())
			err = status.Error(codes.Internal, "internal server error")
		}
	}()
	return handler(srv, stream)
}

func recoverAndExit(scope string) {
	if recovered := recover(); recovered != nil {
		log.Printf("panic in %s panic=%v\n%s", scope, recovered, debug.Stack())
		os.Exit(2)
	}
}

func startHeartbeat() func() {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				var stats runtime.MemStats
				runtime.ReadMemStats(&stats)
				log.Printf(
					"heartbeat pid=%d goroutines=%d heap_alloc_mb=%.1f sys_mb=%.1f",
					os.Getpid(),
					runtime.NumGoroutine(),
					float64(stats.HeapAlloc)/(1024*1024),
					float64(stats.Sys)/(1024*1024),
				)
			case <-stop:
				return
			}
		}
	}()
	return func() {
		close(stop)
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

		matcher, err := matching.NewPostGISMatcher(pool, cfg.RoadTable, cfg.LogLevel == "debug", matching.PostGISMatcherOptions{
			SameRoadHysteresisMargin:        cfg.SameRoadHysteresisMargin,
			UnconnectedJumpHysteresisMargin: cfg.UnconnectedJumpHysteresisMargin,
			HistoryBearingMinDistanceMeters: cfg.HistoryBearingMinDistanceMeters,
			HistoryBearingMaxAgeSeconds:     cfg.HistoryBearingMaxAgeSeconds,
			HistoryBearingMinSpeedMPS:       cfg.HistoryBearingMinSpeedMPS,
			DistanceScoreWeight:             cfg.DistanceScoreWeight,
			HeadingScoreWeight:              cfg.HeadingScoreWeight,
			ConnectivityScoreWeight:         cfg.ConnectivityScoreWeight,
			RoadClassScoreWeight:            cfg.RoadClassScoreWeight,
			SpeedScoreWeight:                cfg.SpeedScoreWeight,
			MediumSpeedMPS:                  cfg.MediumSpeedMPS,
			HighSpeedMPS:                    cfg.HighSpeedMPS,
		})
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
