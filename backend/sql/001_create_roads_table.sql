DROP TABLE IF EXISTS roads;

CREATE TABLE roads AS
WITH dumped_roads AS (
    SELECT
        l.osm_id,
        l.name,
        l.highway,
        l.oneway,
        l.bridge,
        l.tunnel,
        l.layer,
        dumped.path AS segment_path,
        dumped.geom AS geom
    FROM planet_osm_line l
    CROSS JOIN LATERAL ST_Dump(l.way) AS dumped(path, geom)
    WHERE l.highway IS NOT NULL
      AND l.highway IN (
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
)
SELECT
    row_number() OVER (ORDER BY osm_id, segment_path) AS road_id,
    osm_id,
    name,
    highway,
    oneway,
    bridge,
    tunnel,
    layer,
    segment_path,
    geom,
    ST_Length(geom) AS length_m
FROM dumped_roads;

ALTER TABLE roads
ADD PRIMARY KEY (road_id);

CREATE INDEX roads_geom_gix
ON roads
USING GIST (geom);

CREATE INDEX roads_highway_idx
ON roads (highway);

CREATE INDEX roads_osm_id_idx
ON roads (osm_id);

CREATE INDEX roads_oneway_idx
ON roads (oneway);

ANALYZE roads;
