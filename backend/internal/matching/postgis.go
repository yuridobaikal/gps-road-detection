package matching

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

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
    END AS connectivity,
    ST_AsGeoJSON(ST_Transform(geom, 4326)) AS geometry
FROM scored
ORDER BY distance_score DESC, distance_meters ASC
LIMIT ($4::integer * 2);
`

const candidateSQLTemplateRoads = `
WITH params AS (
    SELECT
        ST_Transform(ST_SetSRID(ST_MakePoint($1, $2), 4326), 3857) AS gps_geom,
        LEAST(GREATEST($3::double precision * 2.0, 30.0), 100.0) AS search_radius
),
road_source AS (
    %s
),
last_roads AS (
    SELECT
        r.geom
    FROM road_source r
    WHERE $5 <> 0
      AND r.road_id = $5
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
        WHEN $5 = 0 THEN 'unknown'
        WHEN road_id = $5 THEN 'same_road'
        WHEN EXISTS (
            SELECT 1
            FROM last_roads lr
            WHERE ST_DWithin(lr.geom, scored.geom, 2.0)
        ) THEN 'connected'
        ELSE 'unconnected'
    END AS connectivity,
    ST_AsGeoJSON(ST_Transform(geom, 4326)) AS geometry
FROM scored
ORDER BY distance_score DESC, distance_meters ASC
LIMIT ($4::integer * 2);
`

type PostGISMatcher struct {
	pool         *pgxpool.Pool
	candidateSQL string
}

type geoJSONLineString struct {
	Type        string      `json:"type"`
	Coordinates [][]float64 `json:"coordinates"`
}

func NewPostGISMatcher(pool *pgxpool.Pool, roadTable string) (*PostGISMatcher, error) {
	sourceSQL, err := roadSourceSQL(roadTable)
	if err != nil {
		return nil, err
	}

	template := candidateSQLTemplateRaw
	if isNormalizedRoadTable(roadTable) {
		template = candidateSQLTemplateRoads
	}

	return &PostGISMatcher{
		pool:         pool,
		candidateSQL: fmt.Sprintf(template, sourceSQL),
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
	rows, err := m.pool.Query(ctx, m.candidateSQL, req.GetLon(), req.GetLat(), req.GetAccuracy(), limit, req.GetLastRoadId())
	if err != nil {
		return nil, fmt.Errorf("query postgis candidates: %w", err)
	}
	defer rows.Close()

	radius := searchRadius(req.GetAccuracy())
	candidates := make([]*pb.RoadCandidate, 0, limit)

	for rows.Next() {
		var (
			candidate     pb.RoadCandidate
			distanceScore float64
			localBearing  float64
			geometryJSON  string
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
			&geometryJSON,
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
		candidate.Geometry = parseLineStringGeometry(geometryJSON)

		if candidate.DistanceMeters > radius {
			continue
		}

		candidates = append(candidates, &candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate postgis candidates: %w", err)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].DistanceMeters < candidates[j].DistanceMeters
		}
		return candidates[i].Score > candidates[j].Score
	})
	if len(candidates) > int(limit) {
		candidates = candidates[:limit]
	}

	return candidates, nil
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

func parseLineStringGeometry(raw string) []*pb.LatLon {
	var geometry geoJSONLineString
	if err := json.Unmarshal([]byte(raw), &geometry); err != nil {
		return nil
	}
	if geometry.Type != "LineString" {
		return nil
	}

	points := make([]*pb.LatLon, 0, len(geometry.Coordinates))
	for _, coordinate := range geometry.Coordinates {
		if len(coordinate) < 2 {
			continue
		}
		points = append(points, &pb.LatLon{
			Lon: coordinate[0],
			Lat: coordinate[1],
		})
	}
	return points
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
