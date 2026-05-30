package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	pb "github.com/yurido/gps-road-detection/backend/gen/roadmatcherpb"
	"github.com/yurido/gps-road-detection/backend/internal/triplog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:50051", "gRPC server address")
	input := flag.String("input", "", "trip log input file, .jsonl or .jsonl.gz")
	limit := flag.Int("limit", 0, "maximum events to replay; 0 means all")
	delay := flag.Duration("delay", 0, "delay between replayed requests")
	flag.Parse()

	if *input == "" {
		log.Fatal("-input is required")
	}

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("connect to %s: %v", *addr, err)
	}
	defer conn.Close()

	client := pb.NewRoadMatcherClient(conn)

	reader, closeInput, err := openInput(*input)
	if err != nil {
		log.Fatal(err)
	}
	defer closeInput()

	summary := replaySummary{}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		if *limit > 0 && summary.Total >= *limit {
			break
		}

		var event triplog.Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			summary.Invalid++
			log.Printf("skip invalid event: %v", err)
			continue
		}

		var req pb.MatchRoadRequest
		if err := protojson.Unmarshal(event.Request, &req); err != nil {
			summary.Invalid++
			log.Printf("skip event with invalid request: %v", err)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := client.MatchRoad(ctx, &req)
		cancel()

		summary.Total++
		if err != nil {
			summary.Errors++
			log.Printf("replay error sequence=%d device=%s: %v", req.GetSequenceId(), req.GetDeviceId(), err)
			continue
		}

		if matchesOriginalBest(event, resp) {
			summary.BestMatches++
		} else {
			summary.BestDiffers++
			log.Printf(
				"best differs sequence=%d logged_road_id=%d replay_road_id=%d logged_confidence=%s replay_confidence=%s",
				req.GetSequenceId(),
				loggedBestRoadID(event),
				bestRoadID(resp),
				loggedConfidence(event),
				resp.GetConfidence(),
			)
		}

		if *delay > 0 {
			time.Sleep(*delay)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("read input: %v", err)
	}

	fmt.Printf(
		"replayed=%d best_matches=%d best_differs=%d errors=%d invalid=%d\n",
		summary.Total,
		summary.BestMatches,
		summary.BestDiffers,
		summary.Errors,
		summary.Invalid,
	)
}

type replaySummary struct {
	Total       int
	BestMatches int
	BestDiffers int
	Errors      int
	Invalid     int
}

func openInput(path string) (io.Reader, func(), error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open input: %w", err)
	}

	if len(path) >= len(".gz") && path[len(path)-len(".gz"):] == ".gz" {
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			file.Close()
			return nil, nil, fmt.Errorf("open gzip input: %w", err)
		}
		return gzipReader, func() {
			gzipReader.Close()
			file.Close()
		}, nil
	}

	return file, func() {
		file.Close()
	}, nil
}

func matchesOriginalBest(event triplog.Event, resp *pb.MatchRoadResponse) bool {
	loggedID := loggedBestRoadID(event)
	replayID := bestRoadID(resp)
	if loggedID == 0 || replayID == 0 {
		return loggedID == replayID
	}
	return loggedID == replayID
}

func loggedBestRoadID(event triplog.Event) int64 {
	var response pb.MatchRoadResponse
	if len(event.Response) == 0 {
		return 0
	}
	if err := protojson.Unmarshal(event.Response, &response); err != nil {
		return 0
	}
	return bestRoadID(&response)
}

func loggedConfidence(event triplog.Event) string {
	var response pb.MatchRoadResponse
	if len(event.Response) == 0 {
		return ""
	}
	if err := protojson.Unmarshal(event.Response, &response); err != nil {
		return ""
	}
	return response.GetConfidence()
}

func bestRoadID(resp *pb.MatchRoadResponse) int64 {
	if resp == nil || resp.GetBest() == nil {
		return 0
	}
	return resp.GetBest().GetRoadId()
}
