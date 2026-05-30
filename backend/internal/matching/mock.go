package matching

import (
	"context"
	"math"

	pb "github.com/yurido/gps-road-detection/backend/gen/roadmatcherpb"
)

type MockMatcher struct{}

func NewMockMatcher() *MockMatcher {
	return &MockMatcher{}
}

func (m *MockMatcher) Match(ctx context.Context, req *pb.MatchRoadRequest) (*pb.MatchRoadResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	roadID := req.GetLastRoadId()
	connectivity := "same_road"
	connectivityScore := 1.0

	if roadID == 0 {
		roadID = 1001
		connectivity = "initial_match"
		connectivityScore = 0.6
	}

	headingDiff := normalizeHeadingDiff(req.GetHeading(), 92)
	headingScore := headingScore(headingDiff, req.GetSpeed())
	distanceMeters := math.Min(math.Max(req.GetAccuracy()*0.5, 3), 25)
	radius := searchRadius(req.GetAccuracy())
	distanceScore := clamp(1-distanceMeters/radius, 0, 1)

	best := &pb.RoadCandidate{
		RoadId:            roadID,
		OsmId:             roadID,
		Name:              "Mock Current Road",
		Highway:           "primary",
		Oneway:            "no",
		DistanceMeters:    distanceMeters,
		HeadingDiff:       headingDiff,
		Connectivity:      connectivity,
		DistanceScore:     distanceScore,
		HeadingScore:      headingScore,
		ConnectivityScore: connectivityScore,
		Score:             distanceScore*0.30 + headingScore*0.20 + connectivityScore*0.20 + 0.15 + 0.09 + 0.04,
		Geometry:          mockLine(req.GetLat(), req.GetLon(), 0.0012),
	}

	second := &pb.RoadCandidate{
		RoadId:            roadID + 1,
		OsmId:             roadID + 1,
		Name:              "Mock Nearby Service Road",
		Highway:           "service",
		Oneway:            "no",
		DistanceMeters:    distanceMeters + 8,
		HeadingDiff:       normalizeHeadingDiff(req.GetHeading(), 135),
		Connectivity:      "nearby_unconnected",
		DistanceScore:     clamp(1-(distanceMeters+8)/radius, 0, 1),
		HeadingScore:      0.4,
		ConnectivityScore: 0.1,
		Score:             0.52,
		Geometry:          mockLine(req.GetLat()+0.00008, req.GetLon()+0.00008, 0.0010),
	}

	return &pb.MatchRoadResponse{
		SequenceId: req.GetSequenceId(),
		Matched:    true,
		Confidence: confidence(best.Score, second.Score),
		Best:       responseCandidate(best),
		Candidates: responseCandidates([]*pb.RoadCandidate{best, second}),
	}, nil
}

func mockLine(lat, lon, delta float64) []*pb.LatLon {
	return []*pb.LatLon{
		{Lat: lat - delta, Lon: lon - delta},
		{Lat: lat, Lon: lon},
		{Lat: lat + delta, Lon: lon + delta},
	}
}
