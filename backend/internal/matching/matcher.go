package matching

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	pb "github.com/yurido/gps-road-detection/backend/gen/roadmatcherpb"
)

const (
	DefaultSameRoadHysteresisMargin        = 0.12
	DefaultUnconnectedJumpHysteresisMargin = 0.18
	DefaultHistoryBearingMinDistanceMeters = 8.0
	DefaultHistoryBearingMaxAgeSeconds     = 15.0
	DefaultHistoryBearingMinSpeedMPS       = 2.0
	DefaultDistanceScoreWeight             = 0.32
	DefaultHeadingScoreWeight              = 0.23
	DefaultConnectivityScoreWeight         = 0.25
	DefaultRoadClassScoreWeight            = 0.10
	DefaultSpeedScoreWeight                = 0.10
	DefaultMediumSpeedMPS                  = 8.0
	DefaultHighSpeedMPS                    = 15.0
)

type scoreWeights struct {
	Distance     float64
	Heading      float64
	Connectivity float64
	RoadClass    float64
	Speed        float64
}

type effectiveHeading struct {
	Heading        float64
	Source         string
	DistanceMeters float64
	AgeSeconds     float64
}

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
	switch strings.ToLower(strings.TrimSpace(highway)) {
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

func speedClassScore(highway string, speed, mediumSpeedMPS, highSpeedMPS float64) float64 {
	if mediumSpeedMPS < 0 {
		mediumSpeedMPS = DefaultMediumSpeedMPS
	}
	if highSpeedMPS <= mediumSpeedMPS {
		highSpeedMPS = mediumSpeedMPS + 1
	}
	if speed < mediumSpeedMPS {
		return 1.0
	}

	highway = strings.ToLower(strings.TrimSpace(highway))
	highSpeed := speed >= highSpeedMPS

	switch highway {
	case "service":
		if highSpeed {
			return 0.10
		}
		return 0.45
	case "living_street":
		if highSpeed {
			return 0.25
		}
		return 0.55
	case "residential":
		if highSpeed {
			return 0.45
		}
		return 0.80
	case "unclassified":
		if highSpeed {
			return 0.65
		}
		return 0.85
	default:
		return 1.0
	}
}

func candidateScore(distanceScore, headingScore, connectivityScore, roadClassScore, speedScore float64, weights scoreWeights) float64 {
	totalWeight := weights.sum()
	if totalWeight <= 0 {
		weights = defaultScoreWeights()
		totalWeight = weights.sum()
	}
	return (distanceScore*weights.Distance +
		headingScore*weights.Heading +
		connectivityScore*weights.Connectivity +
		roadClassScore*weights.RoadClass +
		speedScore*weights.Speed) / totalWeight
}

func defaultScoreWeights() scoreWeights {
	return scoreWeights{
		Distance:     DefaultDistanceScoreWeight,
		Heading:      DefaultHeadingScoreWeight,
		Connectivity: DefaultConnectivityScoreWeight,
		RoadClass:    DefaultRoadClassScoreWeight,
		Speed:        DefaultSpeedScoreWeight,
	}
}

func (w scoreWeights) sum() float64 {
	return w.Distance + w.Heading + w.Connectivity + w.RoadClass + w.Speed
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

func selectEffectiveHeading(req *pb.MatchRoadRequest, minDistanceMeters, maxAgeSeconds, minSpeedMPS float64) effectiveHeading {
	gpsHeading := normalizeDegrees(req.GetHeading())
	if req.GetSpeed() < minSpeedMPS {
		return effectiveHeading{Heading: gpsHeading, Source: "gps"}
	}

	bearing, distanceMeters, ageSeconds, ok := movementBearingFromHistory(req, minDistanceMeters, maxAgeSeconds)
	if !ok {
		return effectiveHeading{Heading: gpsHeading, Source: "gps"}
	}

	return effectiveHeading{
		Heading:        bearing,
		Source:         "history",
		DistanceMeters: distanceMeters,
		AgeSeconds:     ageSeconds,
	}
}

func movementBearingFromHistory(req *pb.MatchRoadRequest, minDistanceMeters, maxAgeSeconds float64) (float64, float64, float64, bool) {
	if minDistanceMeters <= 0 || maxAgeSeconds <= 0 || !validLatLon(req.GetLat(), req.GetLon()) {
		return 0, 0, 0, false
	}

	reqTime, hasReqTime := parseTimestamp(req.GetTimestamp())
	for _, point := range req.GetHistory() {
		if !validLatLon(point.GetLat(), point.GetLon()) {
			continue
		}

		ageSeconds := 0.0
		if hasReqTime {
			pointTime, ok := parseTimestamp(point.GetTimestamp())
			if !ok {
				continue
			}
			age := reqTime.Sub(pointTime).Seconds()
			if age < 0 || age > maxAgeSeconds {
				continue
			}
			ageSeconds = age
		}

		distanceMeters := haversineMeters(point.GetLat(), point.GetLon(), req.GetLat(), req.GetLon())
		if distanceMeters < minDistanceMeters {
			continue
		}

		return initialBearingDegrees(point.GetLat(), point.GetLon(), req.GetLat(), req.GetLon()), distanceMeters, ageSeconds, true
	}

	return 0, 0, 0, false
}

func parseTimestamp(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func validLatLon(lat, lon float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lon) && lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMeters = 6371000.0
	lat1Rad := degreesToRadians(lat1)
	lat2Rad := degreesToRadians(lat2)
	deltaLat := degreesToRadians(lat2 - lat1)
	deltaLon := degreesToRadians(lon2 - lon1)

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusMeters * c
}

func initialBearingDegrees(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := degreesToRadians(lat1)
	lat2Rad := degreesToRadians(lat2)
	deltaLon := degreesToRadians(lon2 - lon1)

	y := math.Sin(deltaLon) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) -
		math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(deltaLon)
	return normalizeDegrees(radiansToDegrees(math.Atan2(y, x)))
}

func normalizeDegrees(value float64) float64 {
	normalized := math.Mod(value, 360)
	if normalized < 0 {
		normalized += 360
	}
	if math.IsNaN(normalized) {
		return 0
	}
	return normalized
}

func degreesToRadians(value float64) float64 {
	return value * math.Pi / 180
}

func radiansToDegrees(value float64) float64 {
	return value * 180 / math.Pi
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

func sortCandidates(candidates []*pb.RoadCandidate, preferredRoadID int64) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if preferredRoadID != 0 {
			iPreferred := candidates[i].GetRoadId() == preferredRoadID
			jPreferred := candidates[j].GetRoadId() == preferredRoadID
			if iPreferred != jPreferred {
				return iPreferred
			}
		}
		if candidates[i].GetScore() == candidates[j].GetScore() {
			return candidates[i].GetDistanceMeters() < candidates[j].GetDistanceMeters()
		}
		return candidates[i].GetScore() > candidates[j].GetScore()
	})
}

func preferredRoadWithHysteresis(candidates []*pb.RoadCandidate, lastRoadID int64, sameRoadMargin, unconnectedJumpMargin float64) int64 {
	if lastRoadID == 0 || len(candidates) == 0 {
		return 0
	}

	best := candidates[0]
	if best.GetRoadId() == lastRoadID {
		return 0
	}

	if last := findCandidateByRoadID(candidates, lastRoadID); last != nil {
		if best.GetScore()-last.GetScore() <= sameRoadMargin {
			return last.GetRoadId()
		}
	}

	if best.GetConnectivity() == "unconnected" {
		if continuous := bestContinuousCandidate(candidates); continuous != nil {
			if best.GetScore()-continuous.GetScore() <= unconnectedJumpMargin {
				return continuous.GetRoadId()
			}
		}
	}

	return 0
}

func findCandidateByRoadID(candidates []*pb.RoadCandidate, roadID int64) *pb.RoadCandidate {
	for _, candidate := range candidates {
		if candidate.GetRoadId() == roadID {
			return candidate
		}
	}
	return nil
}

func bestContinuousCandidate(candidates []*pb.RoadCandidate) *pb.RoadCandidate {
	var best *pb.RoadCandidate
	for _, candidate := range candidates {
		connectivity := candidate.GetConnectivity()
		if connectivity != "same_road" && connectivity != "connected" {
			continue
		}
		if best == nil || candidate.GetScore() > best.GetScore() ||
			(candidate.GetScore() == best.GetScore() && candidate.GetDistanceMeters() < best.GetDistanceMeters()) {
			best = candidate
		}
	}
	return best
}
