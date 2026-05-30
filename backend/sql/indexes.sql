DO $$
BEGIN
  IF to_regclass('roads') IS NOT NULL THEN
    CREATE INDEX IF NOT EXISTS roads_geom_gix
      ON roads
      USING GIST (geom);

    CREATE INDEX IF NOT EXISTS roads_highway_idx
      ON roads (highway);

    CREATE INDEX IF NOT EXISTS roads_osm_id_idx
      ON roads (osm_id);

    CREATE INDEX IF NOT EXISTS roads_oneway_idx
      ON roads (oneway);

    ANALYZE roads;
  END IF;

  IF to_regclass('planet_osm_line') IS NOT NULL THEN
    CREATE INDEX IF NOT EXISTS planet_osm_line_way_gix
      ON planet_osm_line
      USING GIST (way);

    CREATE INDEX IF NOT EXISTS planet_osm_line_highway_idx
      ON planet_osm_line (highway);

    CREATE INDEX IF NOT EXISTS planet_osm_line_osm_id_idx
      ON planet_osm_line (osm_id);

    ANALYZE planet_osm_line;
  END IF;

  IF to_regclass('planet_osm_roads') IS NOT NULL THEN
    CREATE INDEX IF NOT EXISTS planet_osm_roads_way_gix
      ON planet_osm_roads
      USING GIST (way);

    CREATE INDEX IF NOT EXISTS planet_osm_roads_highway_idx
      ON planet_osm_roads (highway);

    CREATE INDEX IF NOT EXISTS planet_osm_roads_osm_id_idx
      ON planet_osm_roads (osm_id);

    ANALYZE planet_osm_roads;
  END IF;
END $$;
