# VietnamOSM Current Road Detector App — Product and Technical Specification

## 1. Project Goal

Build a cross-platform mobile app that detects the user's **current road** from live GPS data.

The app will collect GPS and motion-related data from the phone, send it to a backend API, and use an OpenStreetMap road graph stored in PostgreSQL/PostGIS to identify the most likely road the user is currently traveling on.

Primary target platforms:

- Android
- iOS

Recommended mobile framework:

- Flutter

Recommended architecture:

```text
Flutter app
  -> GPS, heading, speed, optional sensors
  -> bidirectional gRPC stream over HTTP/2
Backend service
  -> map-matching algorithm
  -> PostgreSQL + PostGIS
OSM road database
  -> imported from OpenStreetMap Vietnam data
```

## 2. Core User Flow

```text
1. User opens app.
2. App requests location permission.
3. App starts live GPS tracking.
4. App receives GPS updates: lat, lon, speed, heading, accuracy, timestamp.
5. App keeps short GPS history locally.
6. App keeps last matched road and recent candidate geometries locally.
7. App immediately updates GPS display and predicts/stabilizes current road from cached state.
8. App sends current GPS state to backend using a speed-aware stream message interval.
9. Backend queries nearby road candidates from PostGIS or cache.
10. Backend scores candidates using distance, heading, last road, history, speed, and graph connectivity.
11. Backend returns best road, confidence, candidate list, geometry, and continuity metadata.
12. App accepts the response only if it is newer than the current UI state.
13. App displays current road and confidence.
14. App optionally draws candidate roads on map for debugging.
```

The mobile UI should not block on each backend response. While backend matching is pending, the app should continue showing the last confident road and current GPS point.

## 3. Main Data Inputs

The road detector should not use only the current GPS point. It should use a multi-signal map-matching approach.

Recommended input set:

| Input | Priority | Notes |
|---|---:|---|
| Current GPS point | Required | Latitude and longitude |
| GPS accuracy | Required | Used to choose search radius and confidence |
| GPS history | Required | Helps smooth noisy data and determine movement direction |
| Last matched road | Required | Prevents jumping between nearby parallel roads |
| Heading/course | High | Useful when moving faster than walking speed |
| Speed | High | Helps reject impossible transitions |
| Road graph connectivity | High | Keeps movement on connected roads |
| OSM `oneway` tag | High | Prevents wrong-direction match on one-way roads |
| OSM `highway` tag | Medium | Gives road class priority |
| Accelerometer | Later | Optional secondary signal |
| Gyroscope | Later | Better than accelerometer for turn detection |

Recommended MVP inputs:

```text
current GPS point
GPS accuracy
last 5-10 GPS points
last matched road
heading
speed
road graph connectivity
oneway
highway type
cached recent candidate geometries
message sequence ID
```

Do not use accelerometer/gyroscope in the first MVP. Add them after the GPS + road graph matcher works.

## 4. Flutter Location Collection

### 4.1 Basic GPS Service

```dart
import 'package:geolocator/geolocator.dart';

class GpsService {
  Future<bool> requestPermission() async {
    final serviceEnabled = await Geolocator.isLocationServiceEnabled();
    if (!serviceEnabled) {
      return false;
    }

    LocationPermission permission = await Geolocator.checkPermission();

    if (permission == LocationPermission.denied) {
      permission = await Geolocator.requestPermission();
    }

    return permission == LocationPermission.always ||
        permission == LocationPermission.whileInUse;
  }

  Stream<Position> locationStream() {
    const settings = LocationSettings(
      accuracy: LocationAccuracy.bestForNavigation,
      distanceFilter: 5,
    );

    return Geolocator.getPositionStream(
      locationSettings: settings,
    );
  }
}
```

### 4.2 Recommended GPS Fields

Each GPS update should collect:

```text
latitude
longitude
accuracy
speed
heading
timestamp
altitude, optional
```

Example internal model:

```dart
class GpsPoint {
  final double lat;
  final double lon;
  final double accuracy;
  final double? speed;
  final double? heading;
  final DateTime timestamp;

  GpsPoint({
    required this.lat,
    required this.lon,
    required this.accuracy,
    required this.timestamp,
    this.speed,
    this.heading,
  });
}
```

### 4.3 GPS History Buffer

Keep the last 5-20 valid GPS points.

Recommended initial value:

```text
history size = 10 points
```

Use history for:

- movement bearing
- smoothing
- speed validation
- impossible jump detection
- road continuity
- turn detection later

Discard bad GPS points when:

```text
accuracy <= 0
accuracy > 100 meters
timestamp is stale
position jumps unrealistically from previous point
```

## 5. API Design

Use gRPC for the MVP API.

Recommended protocol:

```text
Flutter app -> bidirectional gRPC stream over HTTP/2 -> Backend -> PostgreSQL/PostGIS
```

Do not use a raw database connection from the mobile app.

gRPC is preferred for the real-trip MVP because:

- It supports efficient binary protobuf messages.
- It gives a strict typed API contract between Flutter and backend.
- It supports bidirectional streaming for live trips and unary calls for fallback/debug.
- It avoids repeated JSON encode/decode overhead during high-frequency matching.
- HTTP/2 connection reuse avoids opening a new connection for each match.

REST/JSON can still be added later as a debug or replay endpoint, but it should not be the primary mobile matching path.

### 5.1 gRPC Service

```proto
service RoadMatcher {
  rpc StreamGps(stream MatchRoadRequest) returns (stream MatchRoadResponse);
  rpc MatchRoad(MatchRoadRequest) returns (MatchRoadResponse);
}
```

Purpose:

- Receive GPS state from mobile app.
- Return best matched road and candidate roads.

Use `StreamGps` as the primary live-trip MVP method. The app should open one long-lived gRPC stream while tracking is active, send GPS updates according to the speed-aware scheduling policy, and receive match responses on the same stream.

Keep unary `MatchRoad` as a fallback/debug method for one-off checks, replay tooling, backend tests, and easier diagnostics.

### 5.2 Request Message

```proto
message MatchRoadRequest {
  string device_id = 1;
  uint64 sequence_id = 2;
  string timestamp = 3;
  double lat = 4;
  double lon = 5;
  double accuracy = 6;
  optional double speed = 7;
  optional double heading = 8;
  optional int64 last_road_id = 9;
  repeated GpsHistoryPoint history = 10;
}

message GpsHistoryPoint {
  string timestamp = 1;
  double lat = 2;
  double lon = 3;
  double accuracy = 4;
  optional double speed = 5;
  optional double heading = 6;
}
```

Example logical payload:

```json
{
  "device_id": "abc123",
  "sequence_id": 1042,
  "timestamp": "2026-05-27T10:00:04Z",
  "lat": 21.027763,
  "lon": 105.834160,
  "accuracy": 8.0,
  "speed": 8.5,
  "heading": 92.0,
  "last_road_id": 123456789,
  "history": [
    {
      "timestamp": "2026-05-27T10:00:01Z",
      "lat": 21.027700,
      "lon": 105.834010,
      "accuracy": 7.0
    },
    {
      "timestamp": "2026-05-27T10:00:02Z",
      "lat": 21.027720,
      "lon": 105.834050,
      "accuracy": 7.0
    }
  ]
}
```

The frontend must include either a monotonically increasing `sequence_id` or another equivalent request ordering value. The app should ignore stale backend responses that arrive after a newer GPS update or newer matching response.

### 5.3 Response Message

```proto
message MatchRoadResponse {
  uint64 sequence_id = 1;
  bool matched = 2;
  string confidence = 3;
  optional RoadCandidate best = 4;
  repeated RoadCandidate candidates = 5;
  double processing_duration_ms = 6;
}

message RoadCandidate {
  int64 road_id = 1;
  string name = 3;
  double score = 12;
}

message LatLon {
  double lat = 1;
  double lon = 2;
}
```

Example logical response:

```json
{
  "sequence_id": 1042,
  "matched": true,
  "confidence": "high",
  "best": {
    "road_id": 123456789,
    "name": "Khuất Duy Tiến",
    "score": 0.91
  },
  "candidates": [
    {
      "road_id": 123456789,
      "name": "Khuất Duy Tiến",
      "score": 0.91
    },
    {
      "road_id": 987654321,
      "name": "Service Road",
      "score": 0.52
    }
  ]
}
```

### 5.4 Stream Message Interval Policy

Backend matching messages should be speed-aware. The app should update local GPS UI on every location update, but it should throttle gRPC stream messages based on movement.

Recommended MVP policy:

```text
stopped / speed < 1 m/s       -> every 2-5 seconds
walking / 1-2 m/s             -> every 1-2 seconds
urban driving / 2-15 m/s      -> every 500-1000 ms
fast driving / speed > 15 m/s -> every 250-500 ms
```

Better trigger rule:

```text
Send a backend matching message when:
- moved at least 8-15 meters since last sent matching message
OR
- 1 second passed while moving
OR
- 3-5 seconds passed while stopped
OR
- heading changed sharply near an intersection
```

Suggested initial implementation:

```text
if speed < 1:
    interval = 3000 ms
    min_distance = 0 m
elif speed < 2:
    interval = 1500 ms
    min_distance = 5 m
elif speed < 10:
    interval = 1000 ms
    min_distance = 8 m
elif speed < 20:
    interval = 500 ms
    min_distance = 10 m
else:
    interval = 300 ms
    min_distance = 12 m
```

Rules:

- Minimum stream message interval should normally be 300 ms.
- For unary fallback mode, if a request is already in flight, either skip the next request or cancel/replace it.
- In streaming mode, keep only the newest unsent GPS state if the client is backpressured.
- The frontend must not let an older response overwrite a newer displayed match.
- Backend target matching response time should be around 100-300 ms for normal messages.

### 5.5 Frontend Immediate Result Strategy

For real trips, the frontend should not wait for the backend before showing road status. It should maintain local state:

```text
last confident road
last matched road ID
recent GPS history
recent backend responses
current message sequence ID
```

On each GPS update:

```text
1. Update current GPS dot, speed, heading, and accuracy immediately.
2. Keep displaying the last confident road while backend matching is pending.
3. Keep the current GPS point visible on the map.
4. Send a backend stream message only when interval or distance rules allow.
5. Accept backend result only if sequence_id is current.
6. Switch roads only when score, confidence, heading, and connectivity justify it.
```

This allows immediate frontend behavior during fast driving while keeping the backend as the authoritative matcher.

### 5.6 Caching Strategy

Caching is required for the real-trip MVP. Backend caching is the first priority. Frontend caching is required for immediate display and smooth behavior between backend responses.

Backend cache should include:

```text
road geometry by road_id
recent candidate query results by spatial tile/cell
road connectivity lookups
local road bearing metadata, if precomputed
recent device matching state, if useful
```

Frontend cache should include:

```text
last matched road
last confident road
last 10-20 valid GPS points
recent candidate geometries
recent candidate scores/confidence
last accepted sequence_id
```

Cache goals:

- Avoid repeated database work for nearby GPS updates.
- Keep matching fast during real driving.
- Allow the frontend to show an immediate stable result while backend matching is pending.
- Reduce road jumping by preserving continuity state.

Suggested backend cache keys:

```text
road:{road_id}
connectivity:{road_id}
candidate_tile:{tile_id}:{radius_bucket}
device_state:{device_id}
```

Candidate tile cache should have a short TTL because road data is static but user location changes quickly. Road geometry and connectivity caches can have a long TTL because OSM road data changes slowly after import.

### 5.7 Trip Logging and Replay

Backend should write replayable trip logs for every match request.

Common backend logs should also be written to a normal `.log` file. These logs are for server lifecycle, configuration, debug summaries, PostGIS errors, gRPC errors, and shutdown messages. When file logging is enabled, common logs should go to the file only instead of also printing to the console.

Keep common logs separate from trip logs:

- Common log: human-readable `.log`
- Trip log: replayable `.jsonl` / `.jsonl.gz`

Each log line should contain:

- schema version
- log timestamp
- request type: unary or stream
- matcher duration
- full GPS request, including history and last road ID
- backend response, including best road, candidates, and scores
- error details, if matching failed

Use JSON Lines so each GPS update is one independent record. Rotate the active trip log when it reaches 10MB, gzip the rotated file, and continue writing to a new active file.

Replay tooling should read `.jsonl` and `.jsonl.gz` files, resend logged GPS requests to the backend, and compare replayed best road IDs against the original logged response.

## 6. Database Design

The raw `planet_osm_line` table can be used for early testing, but the recommended real-trip design is to create a clean `roads` table for matching.

The `roads` table should contain one LineString road segment per row. Do the expensive `ST_Dump` step once during setup, not on every GPS request.

### 6.1 Create Roads Table

Assumption:

- OSM data was imported by `osm2pgsql`.
- `planet_osm_line.way` is in EPSG:3857.

```sql
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
```

### 6.2 Add Primary Key

Use a generated `road_id` because `osm_id` is not guaranteed to be unique after splitting ways into LineString segments.

```sql
ALTER TABLE roads
ADD COLUMN road_id BIGSERIAL PRIMARY KEY;
```

### 6.3 Add Indexes

```sql
CREATE INDEX roads_geom_gix
ON roads
USING GIST (geom);

CREATE INDEX roads_highway_idx
ON roads (highway);

CREATE INDEX roads_osm_id_idx
ON roads (osm_id);

CREATE INDEX roads_oneway_idx
ON roads (oneway);
```

## 7. Candidate Road Query

### 7.1 Search Radius

Use GPS accuracy to choose search radius.

Recommended formula:

```text
search_radius = clamp(accuracy * 2.0, 30 meters, 100 meters)
```

Examples:

```text
accuracy = 5m  -> radius = 30m
accuracy = 20m -> radius = 40m
accuracy = 80m -> radius = 100m
```

### 7.2 Candidate Selection Logic

The backend should query nearby drivable road lines inside the GPS search radius.

Preferred runtime table is normalized `roads`. Raw imported OSM tables such as `planet_osm_line` remain supported for debugging, but they are slower because they may require per-request geometry dumping.

Candidate data needed by the scorer:

- stable road ID
- OSM ID
- road name
- highway class
- oneway value
- distance from GPS point
- local road bearing near the GPS point
- connectivity relation to the last matched road

Return more candidates than the UI needs, then score and keep the top results.

The response should stay small. Return only road ID, road name, and score for best road and candidates. Do not return road geometry in the live response unless a separate debug mode is enabled.

## 8. Candidate Scoring Algorithm

### 8.1 Goal

Do not choose only the closest road. Choose the road that is most plausible for the current GPS point, heading, speed, last matched road, and road graph.

### 8.2 Score Components

Scoring V1 uses four components:

- Distance: closer to the GPS point is better.
- Heading: the phone course should align with the road bearing.
- Connectivity: staying on the same road or moving to a connected road is better.
- Road class: major roads get a small tie-breaker advantage over minor/service roads.

Current V1 weights:

- Distance: 35%
- Heading: 25%
- Connectivity: 25%
- Road class: 15%

Speed is not a standalone weighted score yet. It controls how much the backend trusts heading. GPS history is collected and sent, but deeper history smoothing is a next scoring step.

### 8.3 Distance Score

Distance is strongest when the candidate is very close to the GPS point and weakest near the edge of the search radius.

The search radius comes from GPS accuracy, clamped between 30m and 100m. This avoids overreacting to noisy GPS while still keeping the candidate set small.

### 8.4 Heading Score

Heading uses the local road bearing near the projected GPS point.

For two-way roads, compare the GPS heading with both forward and reverse road direction. For one-way roads, compare only the allowed direction.

When speed is below about 2 m/s, heading is weak because phone course is unstable when stopped or moving slowly. At driving speed, close heading alignment strongly increases confidence.

### 8.5 Connectivity Score

Use the last matched road and road graph.

Continuity priority:

- Same road as last match: strongest.
- Directly connected road: strong.
- Unknown because no last road exists yet: neutral.
- Nearby but unconnected road: weak.

This is very important for preventing jumpy results around parallel roads, overpasses, and intersections.

### 8.6 History Score

GPS history should become a weighted scoring component after the current V1 is stable.

Expected use:

- prefer roads aligned with recent movement bearing
- reject impossible jumps
- reduce road switching across several updates
- help detect turns near intersections

### 8.7 Road Class Score

Road class is a tie-breaker, not the main decision.

General priority:

- motorway, trunk, primary: high
- secondary, tertiary: medium-high
- residential, living street, unclassified: medium
- service roads and unknown classes: low

This helps when GPS is between a main road and a service road, but it must not override strong distance, heading, or connectivity evidence.

### 8.8 Speed Score

Speed is used mainly to decide whether heading is reliable.

Later, speed can also reject implausible transitions, such as jumping to a road segment that cannot be reached at the current speed.

Do not overuse speed by road class, because traffic can be slow on major roads and fast enough on smaller roads.

### 8.9 Direction Score

Use OSM `oneway`.

One-way behavior is currently part of heading scoring. If a road is one-way, opposite-direction heading should strongly hurt that candidate.

Later, direction can become a separate score if we need finer tuning.

### 8.10 Candidate Ordering and Confidence

After scoring, sort by final score first and distance second.

Confidence depends on both the best score and the gap between the best and second candidate. A high best score is not enough if the second candidate is almost equal.

The API should return the top candidates even when confidence is low, because the candidate list is required for map debugging and score tuning.

## 9. Road Bearing Calculation

For heading comparison, calculate road bearing near the projected GPS point.

Concept:

1. Project GPS point onto the candidate road line.
2. Find a short local segment around that projected point.
3. Calculate the bearing of that local segment.
4. Compare that local bearing with GPS heading.

This should stay in backend matching logic. It is easier to test and tune there than in Flutter.

## 10. Last Road and Graph Connectivity

### 10.1 Why Last Road Matters

Without last-road memory, the result may jump between adjacent roads:

```text
Road A -> Road B -> Road A -> Road B
```

This happens often around:

- service roads beside main roads
- parallel roads
- bridges
- tunnels
- large intersections
- urban canyons

For the real-trip MVP, last-road continuity and graph/connectivity scoring are required backend features. A distance-only matcher is not sufficient for field testing because it will jump between nearby roads too often.

### 10.2 Connectivity Table

Eventually, create a graph table from OSM road segments.

Long term, use road nodes and road edges extracted from OSM. This will support proper turn validation, route continuity, and impossible-transition checks.

For the MVP, approximate connectivity by checking whether the last matched road geometry touches or is very close to the candidate geometry near an intersection.

This should be improved later by extracting proper graph nodes and edges.

### 10.3 MVP Connectivity Behavior

The backend should score candidate continuity at minimum as:

```text
same road as last matched road -> strongest continuity
directly connected road        -> strong continuity
near intersection              -> medium continuity
unconnected nearby road        -> weak continuity
```

The API response should include enough metadata for debugging:

```text
candidate road_id
last_road_id
connectivity type
connectivity score
distance from GPS
heading difference
final score
```

This makes real trip replay and manual inspection possible.

## 11. Confidence Calculation

Confidence should depend not only on best score, but also on the gap between best and second candidate.

A strong match needs both a good absolute score and clear separation from the next candidate.

Recommended thresholds:

- High confidence: best score is very strong and clearly ahead of the second candidate.
- Medium confidence: best score is acceptable and has a useful gap.
- Low confidence: score is weak, candidates are too close, or no candidate is reliable.

Return candidates even when confidence is low. The app can display debug information.

## 12. Backend Matching Flow

1. Validate GPS accuracy, timestamp, and coordinates.
2. Choose search radius from GPS accuracy.
3. Load nearby road candidates from cache or PostGIS.
4. Attach road metadata, geometry, distance, local bearing, and connectivity relation.
5. Score each candidate using distance, heading, connectivity, and road class.
6. Sort candidates by final score, then distance.
7. Calculate confidence from best score and best-vs-second score gap.
8. Return sequence ID, best road, confidence, candidate list, and processing duration.

## 13. Accelerometer and Gyroscope Plan

### 13.1 Accelerometer

Accelerometer can help detect:

- stopped vehicle
- acceleration
- braking
- rough movement
- phone shake

But accelerometer does not directly identify the road.

Problems:

- phone orientation changes
- vehicle vibration
- sensor noise
- user may hold the phone differently
- battery impact
- calibration complexity

Recommendation:

```text
Do not use accelerometer in MVP.
Use it later only as a secondary signal.
```

### 13.2 Gyroscope

Gyroscope is more useful for turn detection.

Example:

```text
GPS says user is near intersection.
Gyroscope detects right turn.
Candidate road turns right.
Boost that candidate.
```

Recommendation:

```text
Add gyroscope after basic matching works.
Use it mostly near intersections.
```

Sensor priority:

```text
GPS history > heading/course > road graph > speed > gyroscope > accelerometer
```

## 14. Flutter UI Plan

### 14.1 Main Screen

Show:

```text
Current GPS coordinate
Accuracy
Speed
Heading
Backend processing time
Frontend round-trip time
Displayed response age
Matched road ID
Matched road name
Road type
Confidence
Candidate list
```

Example:

```text
Current Road
road_id: 123456789
Khuất Duy Tiến
primary
Confidence: high
Score: 0.91

GPS
Lat: 21.027763
Lon: 105.834160
Accuracy: 8m
Speed: 31 km/h
Heading: 92 degrees
```

### 14.2 Candidate Debug Panel

Show top candidates:

```text
1. 123456789  Khuất Duy Tiến
   score: 0.91

2. 987654321  Service Road
   score: 0.52
```

### 14.3 Map Debug Display

Draw:

```text
blue dot = current GPS point
```

Keep live map rendering light. Candidate road geometry can be reintroduced later behind an explicit debug mode.

## 15. Implementation Phases

### Phase 1 — Flutter GPS MVP

Goal:

- Build Flutter app.
- Request location permission.
- Stream GPS updates.
- Show lat/lon/accuracy/speed/heading.

Deliverables:

- Flutter project
- GPS permission handling
- Location stream
- Debug UI

### Phase 2 — Backend Candidate API, Cache, and Continuity

Goal:

- Create backend API.
- Receive GPS point.
- Query PostGIS for nearby roads from raw OSM tables or a normalized road table.
- Use backend cache for road geometry and repeated candidate lookups.
- Score candidates with distance, road class, heading, last road, and graph/connectivity.
- Return top candidates, best match, confidence, and sequence ID.

Deliverables:

- `RoadMatcher.StreamGps` gRPC method
- `RoadMatcher.MatchRoad` fallback/debug gRPC method
- protobuf service and message definitions
- PostGIS candidate SQL
- backend road geometry cache
- connectivity lookup/cache
- speed-aware stream message contract
- typed gRPC response

### Phase 3 — Candidate Map Display

Goal:

- Draw GPS point on map.
- Keep live map rendering lightweight.

Deliverables:

- `flutter_map` screen
- current GPS marker

### Phase 4 — Scoring V1

Goal:

- Add distance score.
- Add road class score.
- Add heading score.
- Add last-road continuity score.
- Add graph/connectivity score.
- Add confidence.
- Use speed to control heading trust.

Deliverables:

- candidate score
- best candidate
- confidence label
- connectivity metadata for debugging
- component scores for debugging

### Phase 5 — Continuity and Last Road

Goal:

- Keep last matched road.
- Add same-road bonus.
- Add connected-road bonus.
- Reduce road jumping.
- Stabilize frontend display while backend matching is pending.

Deliverables:

- last road input/output
- connectivity scoring
- basic intersection handling
- frontend last confident road state
- stale response protection with sequence ID

### Phase 6 — GPS History Smoothing

Goal:

- Use last N GPS points.
- Reject noisy jumps.
- Use movement bearing from GPS history.

Deliverables:

- history buffer in Flutter
- history score in backend
- movement bearing calculation

### Phase 7 — Trip Logging and Replay

Goal:

- Record real GPS trips.
- Preserve backend response details for later analysis.
- Replay logged trips against new matcher versions.

Deliverables:

- JSONL trip logs
- gzip rotation every 10MB
- replay command for `.jsonl` and `.jsonl.gz`

### Phase 8 — Advanced Sensors

Goal:

- Add gyroscope-assisted turn detection.
- Optionally add accelerometer stop/start detection.

Deliverables:

- optional sensor stream
- turn event detection
- intersection candidate boosting

## 16. Suggested Backend Tech Stack

Any of these are suitable:

| Backend | Notes |
|---|---|
| Go + gRPC | High performance, strong fit for protobuf/gRPC services |
| Node.js + gRPC/NestJS | Good for fast API development |
| Python + grpcio | Good for algorithm prototyping, but benchmark under real trip load |
| Java/Kotlin + gRPC | Enterprise style, robust, strong typed contracts |

Recommended for first implementation:

```text
Go + gRPC or Node.js + gRPC
```

Reason:

- Good gRPC support.
- Efficient long-lived HTTP/2 connections.
- Easier typed contract sharing with Flutter through protobuf.
- Good PostgreSQL/PostGIS libraries.

Python remains useful for prototyping the algorithm, replay tooling, and offline scoring experiments.

## 17. Key Engineering Notes

1. The first version should focus on visibility, continuity, and debugging, not perfect matching.
2. Always return multiple candidates, not only the best road.
3. Keep live response payloads small; do not return geometry by default.
4. Use GPS accuracy to adjust search radius.
5. Use last road and graph connectivity to reduce road jumping.
6. Add accelerometer and gyroscope only after the GPS/PostGIS algorithm works.
7. Keep the matching algorithm on the backend, not in Flutter.
8. Keep scoring weights configurable.
9. Log test GPS traces for replay and tuning.
10. Build a replay tool later to test matching against recorded trips.
11. Use gRPC over HTTP/2 for the MVP API.
12. Use speed-aware gRPC stream message scheduling instead of a fixed request interval.
13. Use backend cache for road geometry, candidate lookups, and connectivity.
14. Use frontend cache for immediate display and stale response protection.
15. Do not allow old API responses to overwrite newer matching state.
16. Support imported OSM tables such as `planet_osm_line` before requiring a cleaned `roads` table.
17. Include road ID everywhere debug output shows a road name.
18. Record replayable trip logs during real testing.
19. Rotate and gzip trip logs so long drives do not create oversized files.

## 18. MVP Definition

The MVP is complete when:

```text
- Flutter app streams GPS updates.
- Flutter app keeps a valid GPS history buffer.
- Flutter app keeps last matched road and last confident road.
- Flutter app uses speed-aware gRPC stream message scheduling.
- Flutter app ignores stale backend responses using sequence_id or equivalent.
- Backend receives GPS data.
- Backend receives heading, speed, GPS history, and last_road_id.
- Backend uses cache for road geometry, candidate lookups, and connectivity.
- Backend returns top road candidates from PostGIS/cache.
- Backend applies distance, road class, heading, last-road continuity, and graph/connectivity scoring.
- Backend returns confidence and debug scoring metadata.
- Flutter displays best road and candidate list.
- Flutter map shows current GPS point.
- Frontend can show an immediate stable road result while waiting for backend.
```

The second version is complete when:

```text
- GPS history smoothing is more advanced.
- Proper graph nodes and edges replace simple geometry-nearby connectivity.
- Gyroscope-assisted turn detection works near intersections.
- Replay tooling supports recorded trip evaluation.
- Road jumping is further reduced in dense urban areas.
```
