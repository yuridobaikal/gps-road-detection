package matching

import (
	"context"
	"math"
	"time"

	pb "github.com/yurido/gps-road-detection/backend/gen/roadmatcherpb"
)

type Matcher interface {
	Match(ctx context.Context, req *pb.MatchRoadRequest) (*pb.MatchRoadResponse, error)
}

func elapsedMs(start time.Time) float64 {
	return float64(time.Since(start).Microseconds()) / 1000.0
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func searchRadius(accuracy float64) float64 {
	return clamp(accuracy*2, 30, 100)
}

func roadClassScore(highway string) float64 {
	switch highway {
	case "motorway":
		return 1.00
	case "trunk":
		return 0.95
	case "primary":
		return 0.90
	case "secondary":
		return 0.80
	case "tertiary":
		return 0.70
	case "motorway_link":
		return 0.75
	case "trunk_link":
		return 0.70
	case "primary_link":
		return 0.65
	case "secondary_link":
		return 0.60
	case "unclassified", "tertiary_link":
		return 0.55
	case "residential":
		return 0.50
	case "living_street":
		return 0.45
	case "service":
		return 0.30
	default:
		return 0.20
	}
}

func confidence(bestScore, secondScore float64) string {
	gap := bestScore - secondScore
	switch {
	case bestScore >= 0.85 && gap >= 0.15:
		return "high"
	case bestScore >= 0.65 && gap >= 0.08:
		return "medium"
	default:
		return "low"
	}
}

func normalizeHeadingDiff(a, b float64) float64 {
	diff := math.Abs(math.Mod(a-b+540, 360) - 180)
	if math.IsNaN(diff) {
		return 180
	}
	return diff
}

func headingScore(diff, speed float64) float64 {
	if speed < 2 {
		return 0.7
	}
	switch {
	case diff <= 10:
		return 1.0
	case diff <= 30:
		return 0.75
	case diff <= 60:
		return 0.4
	case diff <= 90:
		return 0.1
	default:
		return 0
	}
}

func connectivityScore(connectivity string) float64 {
	switch connectivity {
	case "same_road":
		return 1.0
	case "connected":
		return 0.75
	case "unknown":
		return 0.4
	default:
		return 0.1
	}
}
