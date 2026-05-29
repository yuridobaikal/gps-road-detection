CREATE INDEX roads_geom_gix
ON roads
USING GIST (geom);

CREATE INDEX roads_highway_idx
ON roads (highway);

CREATE INDEX roads_osm_id_idx
ON roads (osm_id);

CREATE INDEX roads_oneway_idx
ON roads (oneway);
