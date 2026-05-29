# GPS Road Detection Flutter App

Flutter frontend for the GPS road detection MVP.

The app collects live GPS from the device or Android emulator, keeps recent GPS state locally, and streams updates to the Go gRPC backend. The backend returns the best matched road, confidence, candidate roads, and candidate geometry for map debugging.

Road matching logic lives in the backend, not in Flutter.

Current debug UI shows:

- current GPS coordinate
- accuracy, speed, and heading
- matched road ID and road name
- confidence and score
- top candidate roads with road IDs
- map view with GPS point, best road, and candidates

Backend scoring currently uses distance, heading, last-road connectivity, and road class. Speed is used to decide whether heading is reliable.
