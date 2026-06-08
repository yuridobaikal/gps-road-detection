package matching

import (
	"math"
	"testing"
	"time"

	pb "github.com/yurido/gps-road-detection/backend/gen/roadmatcherpb"
)

func TestPreferredRoadWithHysteresisKeepsSameRoadWhenClose(t *testing.T) {
	candidates := []*pb.RoadCandidate{
		candidate(200, 0.82, 4, "connected"),
		candidate(100, 0.76, 6, "same_road"),
	}
	sortCandidates(candidates, 0)

	preferred := preferredRoadWithHysteresis(candidates, 100, DefaultSameRoadHysteresisMargin, DefaultUnconnectedJumpHysteresisMargin)

	if preferred != 100 {
		t.Fatalf("preferred road = %d, want 100", preferred)
	}
	sortCandidates(candidates, preferred)
	if got := candidates[0].GetRoadId(); got != 100 {
		t.Fatalf("top candidate road = %d, want 100", got)
	}
}

func TestPreferredRoadWithHysteresisAllowsClearSwitch(t *testing.T) {
	candidates := []*pb.RoadCandidate{
		candidate(200, 0.92, 4, "connected"),
		candidate(100, 0.70, 6, "same_road"),
	}
	sortCandidates(candidates, 0)

	preferred := preferredRoadWithHysteresis(candidates, 100, DefaultSameRoadHysteresisMargin, DefaultUnconnectedJumpHysteresisMargin)

	if preferred != 0 {
		t.Fatalf("preferred road = %d, want 0", preferred)
	}
}

func TestPreferredRoadWithHysteresisAvoidsUnconnectedNearTie(t *testing.T) {
	candidates := []*pb.RoadCandidate{
		candidate(300, 0.81, 3, "unconnected"),
		candidate(200, 0.70, 5, "connected"),
	}
	sortCandidates(candidates, 0)

	preferred := preferredRoadWithHysteresis(candidates, 100, DefaultSameRoadHysteresisMargin, DefaultUnconnectedJumpHysteresisMargin)

	if preferred != 200 {
		t.Fatalf("preferred road = %d, want 200", preferred)
	}
}

func TestPreferredRoadWithHysteresisAllowsStrongUnconnectedCandidate(t *testing.T) {
	candidates := []*pb.RoadCandidate{
		candidate(300, 0.95, 3, "unconnected"),
		candidate(200, 0.70, 5, "connected"),
	}
	sortCandidates(candidates, 0)

	preferred := preferredRoadWithHysteresis(candidates, 100, DefaultSameRoadHysteresisMargin, DefaultUnconnectedJumpHysteresisMargin)

	if preferred != 0 {
		t.Fatalf("preferred road = %d, want 0", preferred)
	}
}

func TestSelectEffectiveHeadingUsesHistoryWhenMovementIsMeaningful(t *testing.T) {
	now := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	req := &pb.MatchRoadRequest{
		Timestamp: now.Format(time.RFC3339Nano),
		Lat:       10.001,
		Lon:       106,
		Heading:   90,
		Speed:     8,
		History: []*pb.GpsHistoryPoint{
			{
				Timestamp: now.Add(-5 * time.Second).Format(time.RFC3339Nano),
				Lat:       10,
				Lon:       106,
			},
		},
	}

	heading := selectEffectiveHeading(req, 8, 15, 2)

	if heading.Source != "history" {
		t.Fatalf("heading source = %q, want history", heading.Source)
	}
	if math.Abs(heading.Heading) > 1 {
		t.Fatalf("heading = %.3f, want near 0", heading.Heading)
	}
	if heading.DistanceMeters < 100 {
		t.Fatalf("distance = %.3f, want meaningful movement", heading.DistanceMeters)
	}
}

func TestSelectEffectiveHeadingUsesGpsWhenSpeedIsLow(t *testing.T) {
	req := &pb.MatchRoadRequest{
		Lat:     10.001,
		Lon:     106,
		Heading: 90,
		Speed:   1,
		History: []*pb.GpsHistoryPoint{
			{Lat: 10, Lon: 106},
		},
	}

	heading := selectEffectiveHeading(req, 8, 15, 2)

	if heading.Source != "gps" {
		t.Fatalf("heading source = %q, want gps", heading.Source)
	}
	if heading.Heading != 90 {
		t.Fatalf("heading = %.3f, want 90", heading.Heading)
	}
}

func TestSelectEffectiveHeadingUsesGpsWhenHistoryIsTooClose(t *testing.T) {
	req := &pb.MatchRoadRequest{
		Lat:     10.00001,
		Lon:     106,
		Heading: 90,
		Speed:   8,
		History: []*pb.GpsHistoryPoint{
			{Lat: 10, Lon: 106},
		},
	}

	heading := selectEffectiveHeading(req, 8, 15, 2)

	if heading.Source != "gps" {
		t.Fatalf("heading source = %q, want gps", heading.Source)
	}
}

func TestSpeedClassScoreIsNeutralAtLowSpeed(t *testing.T) {
	score := speedClassScore("service", 4, DefaultMediumSpeedMPS, DefaultHighSpeedMPS)

	if score != 1 {
		t.Fatalf("score = %.3f, want 1", score)
	}
}

func TestSpeedClassScorePenalizesSlowRoadAtHighSpeed(t *testing.T) {
	serviceScore := speedClassScore("service", 20, DefaultMediumSpeedMPS, DefaultHighSpeedMPS)
	primaryScore := speedClassScore("primary", 20, DefaultMediumSpeedMPS, DefaultHighSpeedMPS)

	if serviceScore >= primaryScore {
		t.Fatalf("service score = %.3f, primary score = %.3f; want service lower", serviceScore, primaryScore)
	}
	if serviceScore != 0.10 {
		t.Fatalf("service score = %.3f, want 0.10", serviceScore)
	}
}

func TestCandidateScoreNormalizesConfiguredWeights(t *testing.T) {
	score := candidateScore(1, 0, 0, 0, 0, scoreWeights{Distance: 2, Heading: 2})

	if score != 0.5 {
		t.Fatalf("score = %.3f, want 0.5", score)
	}
}

func candidate(roadID int64, score, distance float64, connectivity string) *pb.RoadCandidate {
	return &pb.RoadCandidate{
		RoadId:         roadID,
		Score:          score,
		DistanceMeters: distance,
		Connectivity:   connectivity,
	}
}
