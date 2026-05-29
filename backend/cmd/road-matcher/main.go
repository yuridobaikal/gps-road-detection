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
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/yurido/gps-road-detection/backend/gen/roadmatcherpb"
	"github.com/yurido/gps-road-detection/backend/internal/config"
	"github.com/yurido/gps-road-detection/backend/internal/matching"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type roadMatcherServer struct {
	pb.UnimplementedRoadMatcherServer
	matcher  matching.Matcher
	logLevel string
}

func (s *roadMatcherServer) MatchRoad(ctx context.Context, req *pb.MatchRoadRequest) (*pb.MatchRoadResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.logRequest("unary", req)
	resp, err := s.matcher.Match(ctx, req)
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
		resp, err := s.matcher.Match(ctx, req)
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
		"%s response sequence=%d matched=%t confidence=%s best_road_id=%d best=%q candidates=%d",
		kind,
		resp.GetSequenceId(),
		resp.GetMatched(),
		resp.GetConfidence(),
		bestRoadID,
		bestName,
		len(resp.GetCandidates()),
	)
	for i, candidate := range resp.GetCandidates() {
		log.Printf(
			"%s candidate[%d] road_id=%d osm_id=%d name=%q highway=%s distance=%.1f heading_diff=%.1f distance_score=%.3f heading_score=%.3f connectivity=%s connectivity_score=%.3f score=%.3f",
			kind,
			i,
			candidate.GetRoadId(),
			candidate.GetOsmId(),
			candidate.GetName(),
			candidate.GetHighway(),
			candidate.GetDistanceMeters(),
			candidate.GetHeadingDiff(),
			candidate.GetDistanceScore(),
			candidate.GetHeadingScore(),
			candidate.GetConnectivity(),
			candidate.GetConnectivityScore(),
			candidate.GetScore(),
		)
	}
}

func main() {
	cfg := config.FromEnv()
	flag.StringVar(&cfg.GRPCAddr, "addr", cfg.GRPCAddr, "gRPC listen address")
	flag.StringVar(&cfg.MatcherMode, "matcher-mode", cfg.MatcherMode, "matcher mode: mock or postgis")
	flag.StringVar(&cfg.DatabaseURL, "database-url", cfg.DatabaseURL, "PostgreSQL connection URL for postgis mode")
	flag.StringVar(&cfg.RoadTable, "road-table", cfg.RoadTable, "road table: planet_osm_line, planet_osm_roads, or roads")
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "log level: info or debug")
	flag.Parse()

	ctx := context.Background()
	matcher, cleanup := buildMatcher(ctx, cfg)
	defer cleanup()

	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("listen on %s: %v", cfg.GRPCAddr, err)
	}

	server := grpc.NewServer()
	pb.RegisterRoadMatcherServer(server, &roadMatcherServer{
		matcher:  matcher,
		logLevel: cfg.LogLevel,
	})
	reflection.Register(server)

	go func() {
		log.Printf("road matcher gRPC server listening on %s mode=%s road_table=%s log_level=%s", cfg.GRPCAddr, cfg.MatcherMode, cfg.RoadTable, cfg.LogLevel)
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
