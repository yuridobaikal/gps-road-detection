package triplog

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	pb "github.com/yurido/gps-road-detection/backend/gen/roadmatcherpb"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const schemaVersion = 1

type Logger struct {
	mu       sync.Mutex
	dir      string
	maxBytes int64
	now      func() time.Time

	file      *os.File
	path      string
	sizeBytes int64
}

type Event struct {
	SchemaVersion int             `json:"schema_version"`
	LoggedAt      string          `json:"logged_at"`
	Kind          string          `json:"kind"`
	DurationMs    float64         `json:"duration_ms"`
	Request       json.RawMessage `json:"request"`
	Response      json.RawMessage `json:"response,omitempty"`
	Error         string          `json:"error,omitempty"`
}

func New(dir string, maxBytes int64) (*Logger, error) {
	if dir == "" {
		return nil, fmt.Errorf("trip log dir is required")
	}
	if maxBytes <= 0 {
		return nil, fmt.Errorf("trip log max bytes must be positive")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create trip log dir: %w", err)
	}

	logger := &Logger{
		dir:      dir,
		maxBytes: maxBytes,
		now:      time.Now,
	}
	if err := logger.openLocked(); err != nil {
		return nil, err
	}
	return logger, nil
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}

func (l *Logger) LogMatch(kind string, req *pb.MatchRoadRequest, resp *pb.MatchRoadResponse, duration time.Duration, matchErr error) error {
	if l == nil {
		return nil
	}

	requestJSON, err := marshalProto(req)
	if err != nil {
		return fmt.Errorf("marshal trip request: %w", err)
	}

	var responseJSON json.RawMessage
	if resp != nil {
		responseJSON, err = marshalProto(resp)
		if err != nil {
			return fmt.Errorf("marshal trip response: %w", err)
		}
	}

	event := Event{
		SchemaVersion: schemaVersion,
		LoggedAt:      l.now().UTC().Format(time.RFC3339Nano),
		Kind:          kind,
		DurationMs:    float64(duration.Microseconds()) / 1000.0,
		Request:       requestJSON,
		Response:      responseJSON,
	}
	if matchErr != nil {
		event.Error = matchErr.Error()
	}

	line, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal trip event: %w", err)
	}
	line = append(line, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		if err := l.openLocked(); err != nil {
			return err
		}
	}
	if l.sizeBytes >= l.maxBytes {
		if err := l.rotateLocked(); err != nil {
			return err
		}
	}

	n, err := l.file.Write(line)
	l.sizeBytes += int64(n)
	if err != nil {
		return fmt.Errorf("write trip log: %w", err)
	}
	if n != len(line) {
		return io.ErrShortWrite
	}
	if l.sizeBytes >= l.maxBytes {
		if err := l.rotateLocked(); err != nil {
			return err
		}
	}
	return nil
}

func marshalProto(message proto.Message) (json.RawMessage, error) {
	marshaler := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: true,
	}
	bytes, err := marshaler.Marshal(message)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(bytes), nil
}

func (l *Logger) openLocked() error {
	ts := l.now().UTC().Format("20060102T150405.000000000Z")
	path := filepath.Join(l.dir, fmt.Sprintf("trip-%s.jsonl", ts))

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("open trip log: %w", err)
	}

	l.file = file
	l.path = path
	l.sizeBytes = 0
	return nil
}

func (l *Logger) rotateLocked() error {
	if l.file == nil {
		return l.openLocked()
	}

	path := l.path
	if err := l.file.Close(); err != nil {
		l.file = nil
		return fmt.Errorf("close trip log before rotation: %w", err)
	}
	l.file = nil
	l.path = ""
	l.sizeBytes = 0

	if err := compressFile(path, path+".gz"); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove uncompressed trip log after rotation: %w", err)
	}
	return l.openLocked()
}

func compressFile(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open trip log for compression: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create compressed trip log: %w", err)
	}
	defer dst.Close()

	gzipWriter := gzip.NewWriter(dst)
	gzipWriter.Name = filepath.Base(srcPath)
	gzipWriter.ModTime = time.Now()

	if _, err := io.Copy(gzipWriter, src); err != nil {
		gzipWriter.Close()
		return fmt.Errorf("compress trip log: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("close compressed trip log: %w", err)
	}
	return nil
}
