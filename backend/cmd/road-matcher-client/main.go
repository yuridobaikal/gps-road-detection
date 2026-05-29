package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"time"

	pb "github.com/yurido/gps-road-detection/backend/gen/roadmatcherpb"
	"google.golang.org/grpc"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:50051", "gRPC server address")
	deviceID := flag.String("device-id", "dev-cli", "device id to send")
	count := flag.Int("count", 5, "number of GPS messages to send")
	interval := flag.Duration("interval", 500*time.Millisecond, "interval between GPS messages")
	flag.Parse()

	conn, err := grpc.Dial(*addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("dial %s: %v", *addr, err)
	}
	defer conn.Close()

	client := pb.NewRoadMatcherClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*count+5)*time.Second)
	defer cancel()

	stream, err := client.StreamGps(ctx)
	if err != nil {
		log.Fatalf("open StreamGps: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				errCh <- nil
				return
			}
			if err != nil {
				errCh <- err
				return
			}

			bestName := ""
			bestScore := 0.0
			if best := resp.GetBest(); best != nil {
				bestName = best.Name
				bestScore = best.Score
			}

			fmt.Printf(
				"recv sequence=%d matched=%t confidence=%s best=%q score=%.3f candidates=%d\n",
				resp.GetSequenceId(),
				resp.GetMatched(),
				resp.GetConfidence(),
				bestName,
				bestScore,
				len(resp.GetCandidates()),
			)
		}
	}()

	for i := 0; i < *count; i++ {
		req := &pb.MatchRoadRequest{
			DeviceId:   *deviceID,
			SequenceId: uint64(i + 1),
			Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
			Lat:        21.027763 + float64(i)*0.00005,
			Lon:        105.834160 + float64(i)*0.00005,
			Accuracy:   8,
			Speed:      8.5,
			Heading:    92,
			LastRoadId: 0,
			History:    nil,
		}

		if err := stream.Send(req); err != nil {
			log.Fatalf("send sequence %d: %v", req.SequenceId, err)
		}

		fmt.Printf("sent sequence=%d lat=%.6f lon=%.6f\n", req.SequenceId, req.Lat, req.Lon)
		time.Sleep(*interval)
	}

	if err := stream.CloseSend(); err != nil {
		log.Fatalf("close send: %v", err)
	}

	if err := <-errCh; err != nil {
		log.Fatalf("receive stream: %v", err)
	}
}
