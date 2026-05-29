CREATE TABLE roads AS
SELECT
    osm_id,
    name,
    highway,
    oneway,
    bridge,
    tunnel,
    layer,
    way AS geom
FROM planet_osm_line
WHERE highway IS NOT NULL
  AND highway IN (
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
  );

ALTER TABLE roads
ADD COLUMN road_id BIGSERIAL PRIMARY KEY;
