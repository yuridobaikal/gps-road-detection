WITH params AS (
    SELECT
        ST_Transform(ST_SetSRID(ST_MakePoint(:lon, :lat), 4326), 3857) AS gps_geom,
        LEAST(GREATEST(:accuracy::double precision * 2.0, 30.0), 100.0) AS search_radius
),
road_source AS (
    SELECT
        osm_id AS road_id,
        osm_id,
        name,
        highway,
        oneway,
        way AS geom
    FROM planet_osm_line
),
last_roads AS (
    SELECT
        dumped.geom
    FROM road_source r
    CROSS JOIN LATERAL ST_Dump(r.geom) AS dumped(path, geom)
    WHERE r.road_id = :last_road_id
      AND GeometryType(dumped.geom) = 'LINESTRING'
      AND ST_IsValid(dumped.geom)
      AND NOT ST_IsEmpty(dumped.geom)
),
candidate_roads AS (
    SELECT
        r.road_id,
        r.osm_id,
        COALESCE(r.name, '') AS name,
        COALESCE(r.highway, '') AS highway,
        COALESCE(r.oneway, '') AS oneway,
        dumped.geom,
        ST_Distance(dumped.geom, p.gps_geom) AS distance_meters,
        p.search_radius,
        ST_LineLocatePoint(dumped.geom, p.gps_geom) AS line_fraction,
        NULLIF(ST_Length(dumped.geom), 0) AS geom_length
    FROM road_source r
    CROSS JOIN params p
    CROSS JOIN LATERAL ST_Dump(r.geom) AS dumped(path, geom)
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
      AND GeometryType(dumped.geom) = 'LINESTRING'
      AND ST_IsValid(dumped.geom)
      AND NOT ST_IsEmpty(dumped.geom)
      AND ST_DWithin(dumped.geom, p.gps_geom, p.search_radius)
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
        WHEN :last_road_id = 0 THEN 'unknown'
        WHEN road_id = :last_road_id THEN 'same_road'
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
LIMIT (:limit::integer * 2);
