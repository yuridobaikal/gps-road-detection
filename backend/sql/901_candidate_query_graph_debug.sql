WITH params AS (
    SELECT
        ST_Transform(ST_SetSRID(ST_MakePoint(:lon, :lat), 4326), 3857) AS gps_geom,
        LEAST(GREATEST(:accuracy::double precision * 2.0, 30.0), 100.0) AS search_radius
),
last_road AS (
    SELECT
        r.road_id,
        r.from_node_id,
        r.to_node_id
    FROM roads r
    WHERE r.road_id = :last_road_id
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
    LIMIT (:limit::integer * 4)
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
    from_node_id,
    to_node_id,
    distance_meters,
    distance_score,
    local_bearing,
    CASE
        WHEN road_id = :last_road_id THEN 'same_road'
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
    END AS connectivity,
    ST_AsGeoJSON(ST_Transform(geom, 4326)) AS geometry
FROM scored
ORDER BY distance_score DESC, distance_meters ASC
LIMIT (:limit::integer * 2);
