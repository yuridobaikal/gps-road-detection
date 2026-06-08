package matching

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/yurido/gps-road-detection/backend/gen/roadmatcherpb"
)

const candidateSQLTemplateRaw = `
WITH params AS (
    SELECT
        ST_Transform(ST_SetSRID(ST_MakePoint($1, $2), 4326), 3857) AS gps_geom,
        LEAST(GREATEST($3::double precision * 2.0, 30.0), 100.0) AS search_radius
),
road_source AS (
    %s
),
filtered_roads AS (
    SELECT
        r.road_id,
        r.osm_id,
        COALESCE(r.name, '') AS name,
        COALESCE(r.highway, '') AS highway,
        COALESCE(r.oneway, '') AS oneway,
        r.geom,
        p.gps_geom,
        p.search_radius
    FROM road_source r
    CROSS JOIN params p
    WHERE r.highway IS NOT NULL
      AND r.highway IN (
          'motorway',
          'trunk',
          'primary',
          'secondary',
          'tertiary',
          'unclassified',
          'residential',
          'living_street',
          'service',
          'motorway_link',
          'trunk_link',
          'primary_link',
          'secondary_link',
          'tertiary_link'
      )
      AND r.geom && ST_Expand(p.gps_geom, p.search_radius)
      AND ST_DWithin(r.geom, p.gps_geom, p.search_radius)
),
last_roads AS (
    SELECT
        dumped.geom
    FROM road_source r
    CROSS JOIN LATERAL ST_Dump(r.geom) AS dumped(path, geom)
    WHERE $5 <> 0
      AND r.road_id = $5
      AND GeometryType(dumped.geom) = 'LINESTRING'
      AND ST_IsValid(dumped.geom)
      AND NOT ST_IsEmpty(dumped.geom)
),
candidate_roads AS (
    SELECT
        r.road_id,
        r.osm_id,
        r.name,
        r.highway,
        r.oneway,
        dumped.geom,
        ST_Distance(dumped.geom, r.gps_geom) AS distance_meters,
        r.search_radius,
        ST_LineLocatePoint(dumped.geom, r.gps_geom) AS line_fraction,
        NULLIF(ST_Length(dumped.geom), 0) AS geom_length
    FROM filtered_roads r
    CROSS JOIN LATERAL ST_Dump(r.geom) AS dumped(path, geom)
    WHERE GeometryType(dumped.geom) = 'LINESTRING'
      AND ST_IsValid(dumped.geom)
      AND NOT ST_IsEmpty(dumped.geom)
      AND dumped.geom && ST_Expand(r.gps_geom, r.search_radius)
      AND ST_DWithin(dumped.geom, r.gps_geom, r.search_radius)
),
scored AS (
    SELECT
        *,
        GREATEST(0, 1 - distance_meters / search_radius) AS distance_score,
        COALESCE(
            DEGREES(
                ST_Azimuth(
                    ST_LineInterpolatePoint(
                        geom,
                        GREATEST(line_fraction - LEAST(10.0 / geom_length, 0.01), 0.0)
                    ),
                    ST_LineInterpolatePoint(
                        geom,
                        LEAST(line_fraction + LEAST(10.0 / geom_length, 0.01), 1.0)
                    )
                )
            ),
            0.0
        ) AS local_bearing
    FROM candidate_roads
)
SELECT
    road_id,
    osm_id,
    name,
    highway,
    oneway,
    distance_meters,
    distance_score,
    local_bearing,
    CASE
        WHEN $5 = 0 THEN 'unknown'
        WHEN road_id = $5 THEN 'same_road'
        WHEN EXISTS (
            SELECT 1
            FROM last_roads lr
            WHERE ST_DWithin(lr.geom, scored.geom, 2.0)
        ) THEN 'connected'
        ELSE 'unconnected'
    END AS connectivity
FROM scored
ORDER BY distance_score DESC, distance_meters ASC
LIMIT ($4::integer * 2);
`

const candidateSQLRoadsBase = `
WITH params AS (
    SELECT
        ST_Transform(ST_SetSRID(ST_MakePoint($1, $2), 4326), 3857) AS gps_geom,
        LEAST(GREATEST($3::double precision * 2.0, 30.0), 100.0) AS search_radius
),
nearby_roads AS (
    SELECT
        r.road_id,
        r.osm_id,
        COALESCE(r.name, '') AS name,
        COALESCE(r.highway, '') AS highway,
        COALESCE(r.oneway, '') AS oneway,
        r.geom,
        p.gps_geom,
        ST_Distance(r.geom, p.gps_geom) AS distance_meters,
        p.search_radius
    FROM roads r
    CROSS JOIN params p
    WHERE r.highway IS NOT NULL
      AND r.highway IN (
          'motorway',
          'trunk',
          'primary',
          'secondary',
          'tertiary',
          'unclassified',
          'residential',
          'living_street',
          'service',
          'motorway_link',
          'trunk_link',
          'primary_link',
          'secondary_link',
          'tertiary_link'
      )
      AND r.geom && ST_Expand(p.gps_geom, p.search_radius)
      AND ST_DWithin(r.geom, p.gps_geom, p.search_radius)
),
preselected AS (
    SELECT *
    FROM nearby_roads
    ORDER BY distance_meters ASC
    LIMIT ($4::integer * 4)
),
localized AS (
    SELECT
        *,
        ST_LineLocatePoint(geom, gps_geom) AS line_fraction,
        NULLIF(ST_Length(geom), 0) AS geom_length
    FROM preselected
),
scored AS (
    SELECT
        *,
        GREATEST(0, 1 - distance_meters / search_radius) AS distance_score,
        COALESCE(
            DEGREES(
                ST_Azimuth(
                    ST_LineInterpolatePoint(
                        geom,
                        GREATEST(line_fraction - LEAST(10.0 / geom_length, 0.01), 0.0)
                    ),
                    ST_LineInterpolatePoint(
                        geom,
                        LEAST(line_fraction + LEAST(10.0 / geom_length, 0.01), 1.0)
                    )
                )
            ),
            0.0
        ) AS local_bearing
    FROM localized
)
SELECT
    road_id,
    osm_id,
    name,
    highway,
    oneway,
    distance_meters,
    distance_score,
    local_bearing,
    %s AS connectivity
FROM scored
ORDER BY distance_score DESC, distance_meters ASC
LIMIT ($4::integer * 2);
`

var candidateSQLRoadsNoConnectivity = fmt.Sprintf(candidateSQLRoadsBase, "'unknown'")

const candidateSQLRoadsWithConnectivity = `
WITH params AS (
    SELECT
        ST_Transform(ST_SetSRID(ST_MakePoint($1, $2), 4326), 3857) AS gps_geom,
        LEAST(GREATEST($3::double precision * 2.0, 30.0), 100.0) AS search_radius
),
last_road AS (
    SELECT
        r.road_id,
        r.from_node_id,
        r.to_node_id
    FROM roads r
    WHERE r.road_id = $5
),
nearby_roads AS (
    SELECT
        r.road_id,
        r.osm_id,
        COALESCE(r.name, '') AS name,
        COALESCE(r.highway, '') AS highway,
        COALESCE(r.oneway, '') AS oneway,
        r.from_node_id,
        r.to_node_id,
        r.geom,
        p.gps_geom,
        ST_Distance(r.geom, p.gps_geom) AS distance_meters,
        p.search_radius
    FROM roads r
    CROSS JOIN params p
    WHERE r.highway IS NOT NULL
      AND r.highway IN (
          'motorway',
          'trunk',
          'primary',
          'secondary',
          'tertiary',
          'unclassified',
          'residential',
          'living_street',
          'service',
          'motorway_link',
          'trunk_link',
          'primary_link',
          'secondary_link',
          'tertiary_link'
      )
      AND r.geom && ST_Expand(p.gps_geom, p.search_radius)
      AND ST_DWithin(r.geom, p.gps_geom, p.search_radius)
),
preselected AS (
    SELECT *
    FROM nearby_roads
    ORDER BY distance_meters ASC
    LIMIT ($4::integer * 4)
),
localized AS (
    SELECT
        *,
        ST_LineLocatePoint(geom, gps_geom) AS line_fraction,
        NULLIF(ST_Length(geom), 0) AS geom_length
    FROM preselected
),
scored AS (
    SELECT
        *,
        GREATEST(0, 1 - distance_meters / search_radius) AS distance_score,
        COALESCE(
            DEGREES(
                ST_Azimuth(
                    ST_LineInterpolatePoint(
                        geom,
                        GREATEST(line_fraction - LEAST(10.0 / geom_length, 0.01), 0.0)
                    ),
                    ST_LineInterpolatePoint(
                        geom,
                        LEAST(line_fraction + LEAST(10.0 / geom_length, 0.01), 1.0)
                    )
                )
            ),
            0.0
        ) AS local_bearing
    FROM localized
)
SELECT
    road_id,
    osm_id,
    name,
    highway,
    oneway,
    distance_meters,
    distance_score,
    local_bearing,
    CASE
        WHEN road_id = $5 THEN 'same_road'
        WHEN EXISTS (
            SELECT 1
            FROM last_road lr
            WHERE scored.from_node_id IS NOT NULL
              AND scored.to_node_id IS NOT NULL
              AND lr.from_node_id IS NOT NULL
              AND lr.to_node_id IS NOT NULL
              AND (
                  scored.from_node_id = lr.from_node_id
                  OR scored.from_node_id = lr.to_node_id
                  OR scored.to_node_id = lr.from_node_id
                  OR scored.to_node_id = lr.to_node_id
              )
        ) THEN 'connected'
        ELSE 'unconnected'
    END AS connectivity
FROM scored
ORDER BY distance_score DESC, distance_meters ASC
LIMIT ($4::integer * 2);
`

type PostGISMatcher struct {
	pool                       *pgxpool.Pool
	candidateSQL               string
	candidateSQLNoConnectivity string
	debug                      bool
	sameRoadHysteresisMargin   float64
	unconnectedJumpMargin      float64
	historyBearingMinDistance  float64
	historyBearingMaxAge       float64
	historyBearingMinSpeed     float64
	scoreWeights               scoreWeights
	mediumSpeedMPS             float64
	highSpeedMPS               float64
}

type PostGISMatcherOptions struct {
	SameRoadHysteresisMargin        float64
	UnconnectedJumpHysteresisMargin float64
	HistoryBearingMinDistanceMeters float64
	HistoryBearingMaxAgeSeconds     float64
	HistoryBearingMinSpeedMPS       float64
	DistanceScoreWeight             float64
	HeadingScoreWeight              float64
	ConnectivityScoreWeight         float64
	RoadClassScoreWeight            float64
	SpeedScoreWeight                float64
	MediumSpeedMPS                  float64
	HighSpeedMPS                    float64
}

func DefaultPostGISMatcherOptions() PostGISMatcherOptions {
	return PostGISMatcherOptions{
		SameRoadHysteresisMargin:        DefaultSameRoadHysteresisMargin,
		UnconnectedJumpHysteresisMargin: DefaultUnconnectedJumpHysteresisMargin,
		HistoryBearingMinDistanceMeters: DefaultHistoryBearingMinDistanceMeters,
		HistoryBearingMaxAgeSeconds:     DefaultHistoryBearingMaxAgeSeconds,
		HistoryBearingMinSpeedMPS:       DefaultHistoryBearingMinSpeedMPS,
		DistanceScoreWeight:             DefaultDistanceScoreWeight,
		HeadingScoreWeight:              DefaultHeadingScoreWeight,
		ConnectivityScoreWeight:         DefaultConnectivityScoreWeight,
		RoadClassScoreWeight:            DefaultRoadClassScoreWeight,
		SpeedScoreWeight:                DefaultSpeedScoreWeight,
		MediumSpeedMPS:                  DefaultMediumSpeedMPS,
		HighSpeedMPS:                    DefaultHighSpeedMPS,
	}
}

func NewPostGISMatcher(pool *pgxpool.Pool, roadTable string, debug bool, options PostGISMatcherOptions) (*PostGISMatcher, error) {
	sourceSQL, err := roadSourceSQL(roadTable)
	if err != nil {
		return nil, err
	}
	if options.SameRoadHysteresisMargin < 0 {
		options.SameRoadHysteresisMargin = DefaultSameRoadHysteresisMargin
	}
	if options.UnconnectedJumpHysteresisMargin < 0 {
		options.UnconnectedJumpHysteresisMargin = DefaultUnconnectedJumpHysteresisMargin
	}
	if options.HistoryBearingMinDistanceMeters <= 0 {
		options.HistoryBearingMinDistanceMeters = DefaultHistoryBearingMinDistanceMeters
	}
	if options.HistoryBearingMaxAgeSeconds <= 0 {
		options.HistoryBearingMaxAgeSeconds = DefaultHistoryBearingMaxAgeSeconds
	}
	if options.HistoryBearingMinSpeedMPS < 0 {
		options.HistoryBearingMinSpeedMPS = DefaultHistoryBearingMinSpeedMPS
	}
	if options.DistanceScoreWeight < 0 {
		options.DistanceScoreWeight = DefaultDistanceScoreWeight
	}
	if options.HeadingScoreWeight < 0 {
		options.HeadingScoreWeight = DefaultHeadingScoreWeight
	}
	if options.ConnectivityScoreWeight < 0 {
		options.ConnectivityScoreWeight = DefaultConnectivityScoreWeight
	}
	if options.RoadClassScoreWeight < 0 {
		options.RoadClassScoreWeight = DefaultRoadClassScoreWeight
	}
	if options.SpeedScoreWeight < 0 {
		options.SpeedScoreWeight = DefaultSpeedScoreWeight
	}
	weights := scoreWeights{
		Distance:     options.DistanceScoreWeight,
		Heading:      options.HeadingScoreWeight,
		Connectivity: options.ConnectivityScoreWeight,
		RoadClass:    options.RoadClassScoreWeight,
		Speed:        options.SpeedScoreWeight,
	}
	if weights.sum() <= 0 {
		weights = defaultScoreWeights()
	}
	if options.MediumSpeedMPS < 0 {
		options.MediumSpeedMPS = DefaultMediumSpeedMPS
	}
	if options.HighSpeedMPS <= options.MediumSpeedMPS {
		options.HighSpeedMPS = DefaultHighSpeedMPS
		if options.HighSpeedMPS <= options.MediumSpeedMPS {
			options.HighSpeedMPS = options.MediumSpeedMPS + 1
		}
	}

	candidateSQL := fmt.Sprintf(candidateSQLTemplateRaw, sourceSQL)
	candidateSQLNoConnectivity := candidateSQL
	if isNormalizedRoadTable(roadTable) {
		if err := requireRoadGraphColumns(context.Background(), pool); err != nil {
			return nil, err
		}
		candidateSQL = candidateSQLRoadsWithConnectivity
		candidateSQLNoConnectivity = candidateSQLRoadsNoConnectivity
	}

	return &PostGISMatcher{
		pool:                       pool,
		candidateSQL:               candidateSQL,
		candidateSQLNoConnectivity: candidateSQLNoConnectivity,
		debug:                      debug,
		sameRoadHysteresisMargin:   options.SameRoadHysteresisMargin,
		unconnectedJumpMargin:      options.UnconnectedJumpHysteresisMargin,
		historyBearingMinDistance:  options.HistoryBearingMinDistanceMeters,
		historyBearingMaxAge:       options.HistoryBearingMaxAgeSeconds,
		historyBearingMinSpeed:     options.HistoryBearingMinSpeedMPS,
		scoreWeights:               weights,
		mediumSpeedMPS:             options.MediumSpeedMPS,
		highSpeedMPS:               options.HighSpeedMPS,
	}, nil
}

func (m *PostGISMatcher) Match(ctx context.Context, req *pb.MatchRoadRequest) (*pb.MatchRoadResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.GetAccuracy() <= 0 || req.GetAccuracy() > 100 {
		return &pb.MatchRoadResponse{
			SequenceId: req.GetSequenceId(),
			Matched:    false,
			Confidence: "low",
		}, nil
	}

	candidates, err := m.queryCandidates(ctx, req, 10)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return &pb.MatchRoadResponse{
			SequenceId: req.GetSequenceId(),
			Matched:    false,
			Confidence: "low",
		}, nil
	}

	secondScore := 0.0
	if len(candidates) > 1 {
		secondScore = candidates[1].Score
	}

	return &pb.MatchRoadResponse{
		SequenceId: req.GetSequenceId(),
		Matched:    true,
		Confidence: confidence(candidates[0].Score, secondScore),
		Best:       candidates[0],
		Candidates: candidates,
	}, nil
}

func (m *PostGISMatcher) queryCandidates(ctx context.Context, req *pb.MatchRoadRequest, limit int32) ([]*pb.RoadCandidate, error) {
	totalStart := time.Now()
	m.debugLogf("postgis queryCandidates start sequence=%d device=%s lat=%.6f lon=%.6f accuracy=%.1f last_road_id=%d limit=%d", req.GetSequenceId(), req.GetDeviceId(), req.GetLat(), req.GetLon(), req.GetAccuracy(), req.GetLastRoadId(), limit)

	queryStart := time.Now()
	m.debugLogf("postgis query start sequence=%d", req.GetSequenceId())
	query := m.candidateSQL
	args := []any{req.GetLon(), req.GetLat(), req.GetAccuracy(), limit, req.GetLastRoadId()}
	if req.GetLastRoadId() == 0 {
		query = m.candidateSQLNoConnectivity
		args = args[:4]
	}
	rows, err := m.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query postgis candidates: %w", err)
	}
	defer rows.Close()
	m.debugLogf("postgis query rows opened sequence=%d query_open_ms=%.1f", req.GetSequenceId(), elapsedMs(queryStart))

	radius := searchRadius(req.GetAccuracy())
	heading := selectEffectiveHeading(req, m.historyBearingMinDistance, m.historyBearingMaxAge, m.historyBearingMinSpeed)
	if heading.Source == "history" {
		m.debugLogf(
			"history bearing selected sequence=%d heading=%.1f distance_meters=%.1f age_seconds=%.1f gps_heading=%.1f",
			req.GetSequenceId(),
			heading.Heading,
			heading.DistanceMeters,
			heading.AgeSeconds,
			req.GetHeading(),
		)
	}
	candidates := make([]*pb.RoadCandidate, 0, limit)
	scanStart := time.Now()
	scannedRows := 0

	for rows.Next() {
		scannedRows++
		var (
			candidate     pb.RoadCandidate
			distanceScore float64
			localBearing  float64
		)

		if err := rows.Scan(
			&candidate.RoadId,
			&candidate.OsmId,
			&candidate.Name,
			&candidate.Highway,
			&candidate.Oneway,
			&candidate.DistanceMeters,
			&distanceScore,
			&localBearing,
			&candidate.Connectivity,
		); err != nil {
			return nil, fmt.Errorf("scan postgis candidate: %w", err)
		}

		classScore := roadClassScore(candidate.Highway)
		speedScore := speedClassScore(candidate.Highway, req.GetSpeed(), m.mediumSpeedMPS, m.highSpeedMPS)
		headingDiff := headingDiffForRoad(heading.Heading, localBearing, candidate.Oneway)
		candidate.DistanceScore = clamp(distanceScore, 0, 1)
		candidate.HeadingDiff = headingDiff
		candidate.HeadingScore = headingScore(headingDiff, req.GetSpeed())
		candidate.ConnectivityScore = connectivityScore(candidate.Connectivity)
		candidate.Score = candidateScore(
			candidate.DistanceScore,
			candidate.HeadingScore,
			candidate.ConnectivityScore,
			classScore,
			speedScore,
			m.scoreWeights,
		)

		if candidate.DistanceMeters > radius {
			continue
		}

		candidates = append(candidates, &candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate postgis candidates: %w", err)
	}
	m.debugLogf("postgis scan done sequence=%d scanned_rows=%d accepted_candidates=%d scan_ms=%.1f", req.GetSequenceId(), scannedRows, len(candidates), elapsedMs(scanStart))

	sortStart := time.Now()
	sortCandidates(candidates, 0)
	preferredRoadID := preferredRoadWithHysteresis(candidates, req.GetLastRoadId(), m.sameRoadHysteresisMargin, m.unconnectedJumpMargin)
	if preferredRoadID != 0 {
		m.debugLogf("hysteresis preferred road sequence=%d last_road_id=%d preferred_road_id=%d raw_best_road_id=%d", req.GetSequenceId(), req.GetLastRoadId(), preferredRoadID, candidates[0].GetRoadId())
		sortCandidates(candidates, preferredRoadID)
	}
	if len(candidates) > int(limit) {
		candidates = candidates[:limit]
	}
	m.debugLogf("postgis sort/trim done sequence=%d returned_candidates=%d sort_ms=%.1f total_ms=%.1f", req.GetSequenceId(), len(candidates), elapsedMs(sortStart), elapsedMs(totalStart))

	return candidates, nil
}

func (m *PostGISMatcher) debugLogf(format string, args ...any) {
	if !m.debug {
		return
	}
	log.Printf(format, args...)
}

func roadSourceSQL(roadTable string) (string, error) {
	switch strings.TrimSpace(roadTable) {
	case "", "planet_osm_line":
		return `
        SELECT
            osm_id AS road_id,
            osm_id,
            name,
            highway,
            oneway,
            way AS geom
        FROM planet_osm_line`, nil
	case "planet_osm_roads":
		return `
        SELECT
            osm_id AS road_id,
            osm_id,
            name,
            highway,
            oneway,
            way AS geom
        FROM planet_osm_roads`, nil
	case "roads":
		return `
        SELECT
            road_id,
            osm_id,
            name,
            highway,
            oneway,
            geom
        FROM roads`, nil
	default:
		return "", fmt.Errorf("unsupported road table %q; use planet_osm_line, planet_osm_roads, or roads", roadTable)
	}
}

func isNormalizedRoadTable(roadTable string) bool {
	return strings.TrimSpace(roadTable) == "roads"
}

func requireRoadGraphColumns(ctx context.Context, pool *pgxpool.Pool) error {
	var hasFromNodeID bool
	var hasToNodeID bool
	err := pool.QueryRow(ctx, `
SELECT
    EXISTS (
        SELECT 1
        FROM pg_attribute
        WHERE attrelid = 'roads'::regclass
          AND attname = 'from_node_id'
          AND NOT attisdropped
    ),
    EXISTS (
        SELECT 1
        FROM pg_attribute
        WHERE attrelid = 'roads'::regclass
          AND attname = 'to_node_id'
          AND NOT attisdropped
    );
`).Scan(&hasFromNodeID, &hasToNodeID)
	if err != nil {
		return fmt.Errorf("check roads graph columns: %w", err)
	}
	if !hasFromNodeID || !hasToNodeID {
		return fmt.Errorf("roads graph columns are missing; run: psql \"$DATABASE_URL\" -f backend/sql/003_build_road_graph.sql")
	}
	return nil
}

func headingDiffForRoad(gpsHeading, localBearing float64, oneway string) float64 {
	normalizedOneway := strings.ToLower(strings.TrimSpace(oneway))
	if normalizedOneway == "-1" || normalizedOneway == "reverse" {
		return normalizeHeadingDiff(gpsHeading, localBearing+180)
	}
	if normalizedOneway == "yes" || normalizedOneway == "1" || normalizedOneway == "true" {
		return normalizeHeadingDiff(gpsHeading, localBearing)
	}

	forward := normalizeHeadingDiff(gpsHeading, localBearing)
	reverse := normalizeHeadingDiff(gpsHeading, localBearing+180)
	if reverse < forward {
		return reverse
	}
	return forward
}
