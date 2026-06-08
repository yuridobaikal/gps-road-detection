DO $$
BEGIN
  IF to_regclass('roads') IS NULL THEN
    RAISE EXCEPTION 'roads table does not exist. Run backend/sql/001_create_roads_table.sql first.';
  END IF;
END $$;

DROP TABLE IF EXISTS road_nodes;

CREATE TABLE road_nodes AS
WITH endpoints AS (
    SELECT
        ROUND(ST_X(ST_StartPoint(geom)) / 0.01)::bigint AS x_key,
        ROUND(ST_Y(ST_StartPoint(geom)) / 0.01)::bigint AS y_key
    FROM roads
    WHERE geom IS NOT NULL
      AND GeometryType(geom) = 'LINESTRING'
      AND NOT ST_IsEmpty(geom)

    UNION

    SELECT
        ROUND(ST_X(ST_EndPoint(geom)) / 0.01)::bigint AS x_key,
        ROUND(ST_Y(ST_EndPoint(geom)) / 0.01)::bigint AS y_key
    FROM roads
    WHERE geom IS NOT NULL
      AND GeometryType(geom) = 'LINESTRING'
      AND NOT ST_IsEmpty(geom)
),
distinct_nodes AS (
    SELECT DISTINCT x_key, y_key
    FROM endpoints
)
SELECT
    row_number() OVER (ORDER BY x_key, y_key) AS node_id,
    x_key,
    y_key,
    ST_SetSRID(ST_MakePoint(x_key * 0.01, y_key * 0.01), 3857) AS geom
FROM distinct_nodes;

ALTER TABLE road_nodes
ADD PRIMARY KEY (node_id);

CREATE UNIQUE INDEX road_nodes_xy_key_idx
ON road_nodes (x_key, y_key);

CREATE INDEX road_nodes_geom_gix
ON road_nodes
USING GIST (geom);

ALTER TABLE roads
ADD COLUMN IF NOT EXISTS from_node_id bigint;

ALTER TABLE roads
ADD COLUMN IF NOT EXISTS to_node_id bigint;

WITH road_endpoint_keys AS (
    SELECT
        road_id,
        ROUND(ST_X(ST_StartPoint(geom)) / 0.01)::bigint AS from_x_key,
        ROUND(ST_Y(ST_StartPoint(geom)) / 0.01)::bigint AS from_y_key,
        ROUND(ST_X(ST_EndPoint(geom)) / 0.01)::bigint AS to_x_key,
        ROUND(ST_Y(ST_EndPoint(geom)) / 0.01)::bigint AS to_y_key
    FROM roads
    WHERE geom IS NOT NULL
      AND GeometryType(geom) = 'LINESTRING'
      AND NOT ST_IsEmpty(geom)
)
UPDATE roads r
SET
    from_node_id = from_node.node_id,
    to_node_id = to_node.node_id
FROM road_endpoint_keys k
JOIN road_nodes from_node
  ON from_node.x_key = k.from_x_key
 AND from_node.y_key = k.from_y_key
JOIN road_nodes to_node
  ON to_node.x_key = k.to_x_key
 AND to_node.y_key = k.to_y_key
WHERE r.road_id = k.road_id;

CREATE INDEX IF NOT EXISTS roads_from_node_id_idx
ON roads (from_node_id);

CREATE INDEX IF NOT EXISTS roads_to_node_id_idx
ON roads (to_node_id);

ANALYZE road_nodes;
ANALYZE roads;
