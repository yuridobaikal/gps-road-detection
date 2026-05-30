package matching

import (
	"context"
	"fmt"
	"log"
	"sort"
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
last_roads AS (
    SELECT
        r.geom
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
            FROM last_roads lr
            WHERE ST_DWithin(lr.geom, scored.geom, 2.0)
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
}

func NewPostGISMatcher(pool *pgxpool.Pool, roadTable string, debug bool) (*PostGISMatcher, error) {
	sourceSQL, err := roadSourceSQL(roadTable)
	if err != nil {
		return nil, err
	}

	candidateSQL := fmt.Sprintf(candidateSQLTemplateRaw, sourceSQL)
	candidateSQLNoConnectivity := candidateSQL
	if isNormalizedRoadTable(roadTable) {
		candidateSQL = candidateSQLRoadsWithConnectivity
		candidateSQLNoConnectivity = candidateSQLRoadsNoConnectivity
	}

	return &PostGISMatcher{
		pool:                       pool,
		candidateSQL:               candidateSQL,
		candidateSQLNoConnectivity: candidateSQLNoConnectivity,
		debug:                      debug,
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
		Best:       responseCandidate(candidates[0]),
		Candidates: responseCandidates(candidates),
	}, nil
}

func (m *PostGISMatcher) queryCandidates(ctx context.Context, req *pb.MatchRoadRequest, limit int32) ([]*pb.RoadCandidate, error) {
	totalStart := time.Now()
	m.debugLogf("postgis queryCandidates start sequence=%d device=%s lat=%.6f lon=%.6f accuracy=%.1f last_road_id=%d limit=%d", req.GetSequenceId(), req.GetDeviceId(), req.GetLat(), req.GetLon(), req.GetAccuracy(), req.GetLastRoadId(), limit)

	queryStart := time.Now()
	m.debugLogf("postgis query start sequence=%d", req.GetSequenceId())
	query := m.candidateSQL
	if req.GetLastRoadId() == 0 {
		query = m.candidateSQLNoConnectivity
	}
	rows, err := m.pool.Query(ctx, query, req.GetLon(), req.GetLat(), req.GetAccuracy(), limit, req.GetLastRoadId())
	if err != nil {
		return nil, fmt.Errorf("query postgis candidates: %w", err)
	}
	defer rows.Close()
	m.debugLogf("postgis query rows opened sequence=%d query_open_ms=%.1f", req.GetSequenceId(), elapsedMs(queryStart))

	radius := searchRadius(req.GetAccuracy())
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
		headingDiff := headingDiffForRoad(req.GetHeading(), localBearing, candidate.Oneway)
		candidate.DistanceScore = clamp(distanceScore, 0, 1)
		candidate.HeadingDiff = headingDiff
		candidate.HeadingScore = headingScore(headingDiff, req.GetSpeed())
		candidate.ConnectivityScore = connectivityScore(candidate.Connectivity)
		candidate.Score = candidate.DistanceScore*0.35 +
			candidate.HeadingScore*0.25 +
			candidate.ConnectivityScore*0.25 +
			classScore*0.15

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
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].DistanceMeters < candidates[j].DistanceMeters
		}
		return candidates[i].Score > candidates[j].Score
	})
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
