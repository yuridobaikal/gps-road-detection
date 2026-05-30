package triplog

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	pb "github.com/yurido/gps-road-detection/backend/gen/roadmatcherpb"
)

func TestLogMatchWritesRequestResponse(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(dir, 10*1024*1024)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer logger.Close()

	req := testRequest()
	resp := testResponse()
	if err := logger.LogMatch("stream", req, resp, 25*time.Millisecond, nil); err != nil {
		t.Fatalf("LogMatch: %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	paths, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("jsonl files = %d, want 1", len(paths))
	}

	event := readFirstEvent(t, paths[0])
	if event.SchemaVersion != 1 {
		t.Fatalf("schema version = %d, want 1", event.SchemaVersion)
	}
	if event.Kind != "stream" {
		t.Fatalf("kind = %q, want stream", event.Kind)
	}
	if event.DurationMs != 25 {
		t.Fatalf("duration = %v, want 25", event.DurationMs)
	}
	if got := jsonPathString(t, event.Request, "device_id"); got != "test-device" {
		t.Fatalf("request device_id = %q", got)
	}
	if got := jsonPathString(t, event.Response, "confidence"); got != "high" {
		t.Fatalf("response confidence = %q", got)
	}
	if got := jsonPathString(t, event.Response, "best", "name"); got != "Main Road" {
		t.Fatalf("best name = %q", got)
	}
}

func TestLogMatchRotatesAndCompresses(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(dir, 300)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer logger.Close()

	if err := logger.LogMatch("stream", testRequest(), testResponse(), time.Millisecond, nil); err != nil {
		t.Fatalf("LogMatch first: %v", err)
	}
	if err := logger.LogMatch("stream", testRequest(), testResponse(), time.Millisecond, nil); err != nil {
		t.Fatalf("LogMatch second: %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	gzPaths, err := filepath.Glob(filepath.Join(dir, "*.jsonl.gz"))
	if err != nil {
		t.Fatalf("Glob gz: %v", err)
	}
	if len(gzPaths) == 0 {
		t.Fatal("gz files = 0, want at least 1")
	}

	gzFile, err := os.Open(gzPaths[0])
	if err != nil {
		t.Fatalf("open gz: %v", err)
	}
	defer gzFile.Close()

	gzReader, err := gzip.NewReader(gzFile)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer gzReader.Close()

	scanner := bufio.NewScanner(gzReader)
	if !scanner.Scan() {
		t.Fatal("compressed log is empty")
	}
	var event Event
	if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
		t.Fatalf("unmarshal compressed event: %v", err)
	}
	if event.Request == nil || event.Response == nil {
		t.Fatal("compressed event is missing request or response")
	}
}

func readFirstEvent(t *testing.T, path string) Event {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open log: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatal("log is empty")
	}
	var event Event
	if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	return event
}

func jsonPathString(t *testing.T, raw json.RawMessage, path ...string) string {
	t.Helper()

	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("unmarshal json path: %v", err)
	}
	for _, key := range path {
		object, ok := value.(map[string]any)
		if !ok {
			t.Fatalf("path %v is not an object at %q", path, key)
		}
		value = object[key]
	}
	stringValue, ok := value.(string)
	if !ok {
		t.Fatalf("path %v is %T, want string", path, value)
	}
	return stringValue
}

func testRequest() *pb.MatchRoadRequest {
	return &pb.MatchRoadRequest{
		DeviceId:   "test-device",
		SequenceId: 42,
		Timestamp:  "2026-05-29T00:00:00Z",
		Lat:        21.027763,
		Lon:        105.834160,
		Accuracy:   8,
		Speed:      8.5,
		Heading:    92,
		History: []*pb.GpsHistoryPoint{
			{
				Timestamp: "2026-05-29T00:00:00Z",
				Lat:       21.027700,
				Lon:       105.834100,
				Accuracy:  8,
			},
		},
	}
}

func testResponse() *pb.MatchRoadResponse {
	return &pb.MatchRoadResponse{
		SequenceId: 42,
		Matched:    true,
		Confidence: "high",
		Best: &pb.RoadCandidate{
			RoadId:            123,
			OsmId:             123,
			Name:              "Main Road",
			Highway:           "primary",
			Oneway:            "no",
			DistanceMeters:    4.8,
			HeadingDiff:       3,
			Connectivity:      "same_road",
			DistanceScore:     0.9,
			HeadingScore:      1,
			ConnectivityScore: 1,
			Score:             0.91,
			Geometry: []*pb.LatLon{
				{Lat: 21.027760, Lon: 105.834150},
				{Lat: 21.027770, Lon: 105.834170},
			},
		},
	}
}
