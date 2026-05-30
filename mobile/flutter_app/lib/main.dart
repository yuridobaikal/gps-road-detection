import 'dart:async';
import 'dart:io' show Platform;

import 'package:fixnum/fixnum.dart';
import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:geolocator/geolocator.dart';
import 'package:grpc/grpc.dart';
import 'package:latlong2/latlong.dart';

import 'gen/road_matcher.pbgrpc.dart';

const _defaultBackendHost = String.fromEnvironment(
  'ROAD_MATCHER_HOST',
  defaultValue: '127.0.0.1',
);
const _backendPort = int.fromEnvironment(
  'ROAD_MATCHER_PORT',
  defaultValue: 50051,
);
const _deviceId = String.fromEnvironment(
  'ROAD_MATCHER_DEVICE_ID',
  defaultValue: 'dev-flutter',
);

void main() {
  runApp(const RoadDetectorApp());
}

class RoadDetectorApp extends StatelessWidget {
  const RoadDetectorApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Road Detector',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color(0xFF116A7B),
          brightness: Brightness.light,
        ),
        useMaterial3: true,
      ),
      home: const RoadDetectorScreen(),
    );
  }
}

class RoadDetectorScreen extends StatefulWidget {
  const RoadDetectorScreen({super.key});

  @override
  State<RoadDetectorScreen> createState() => _RoadDetectorScreenState();
}

class _RoadDetectorScreenState extends State<RoadDetectorScreen> {
  final List<GpsHistoryPoint> _history = [];
  StreamController<MatchRoadRequest>? _outgoing;

  ClientChannel? _channel;
  StreamSubscription<Position>? _gpsSubscription;
  StreamSubscription<MatchRoadResponse>? _matchSubscription;

  Position? _position;
  MatchRoadResponse? _lastResponse;
  RoadCandidate? _lastConfidentRoad;
  int _sequence = 0;
  int _lastAcceptedSequence = 0;
  int _lastMatchedRoadId = 0;
  final Map<int, DateTime> _sentAtBySequence = {};
  double? _lastRoundTripMs;
  DateTime? _lastAcceptedAt;
  DateTime? _lastSentAt;
  Position? _lastSentPosition;
  String _status = 'Disconnected';
  bool _tracking = false;

  String get _backendHost {
    if (_defaultBackendHost != '127.0.0.1') {
      return _defaultBackendHost;
    }
    if (Platform.isAndroid) {
      return '10.0.2.2';
    }
    return _defaultBackendHost;
  }

  @override
  void initState() {
    super.initState();
    _connectStream();
  }

  Future<void> _connectStream() async {
    await _matchSubscription?.cancel();
    await _channel?.shutdown();
    await _outgoing?.close();

    final outgoing = StreamController<MatchRoadRequest>();

    final channel = ClientChannel(
      _backendHost,
      port: _backendPort,
      options: const ChannelOptions(credentials: ChannelCredentials.insecure()),
    );

    final client = RoadMatcherClient(channel);
    final responseStream = client.streamGps(outgoing.stream);

    _matchSubscription = responseStream.listen(
      _handleMatchResponse,
      onError: (Object error) {
        if (mounted) {
          setState(() => _status = 'Stream error: $error');
        }
      },
      onDone: () {
        if (mounted) {
          setState(() => _status = 'Stream closed');
        }
      },
    );

    setState(() {
      _outgoing = outgoing;
      _channel = channel;
      _status = 'Stream opened to $_backendHost:$_backendPort';
    });
  }

  Future<void> _startTracking() async {
    final allowed = await _requestLocationPermission();
    if (!allowed) {
      setState(() => _status = 'Location permission unavailable');
      return;
    }

    const settings = LocationSettings(
      accuracy: LocationAccuracy.bestForNavigation,
      distanceFilter: 5,
    );

    await _gpsSubscription?.cancel();
    _gpsSubscription = Geolocator.getPositionStream(locationSettings: settings)
        .listen(
          _handlePosition,
          onError: (Object error) {
            if (mounted) {
              setState(() => _status = 'GPS error: $error');
            }
          },
        );

    setState(() {
      _tracking = true;
      _status = 'Tracking GPS';
    });
  }

  Future<bool> _requestLocationPermission() async {
    final serviceEnabled = await Geolocator.isLocationServiceEnabled();
    if (!serviceEnabled) {
      return false;
    }

    var permission = await Geolocator.checkPermission();
    if (permission == LocationPermission.denied) {
      permission = await Geolocator.requestPermission();
    }

    return permission == LocationPermission.always ||
        permission == LocationPermission.whileInUse;
  }

  void _handlePosition(Position position) {
    if (!_isValidPosition(position)) {
      setState(() {
        _position = position;
        _status =
            'Ignored weak GPS accuracy ${position.accuracy.toStringAsFixed(1)}m';
      });
      return;
    }

    final point = GpsHistoryPoint(
      timestamp: position.timestamp.toUtc().toIso8601String(),
      lat: position.latitude,
      lon: position.longitude,
      accuracy: position.accuracy,
      speed: position.speed,
      heading: position.heading,
    );

    _history.add(point);
    if (_history.length > 10) {
      _history.removeAt(0);
    }

    setState(() {
      _position = position;
      _status = 'GPS update received';
    });

    if (_shouldSend(position)) {
      _sendPosition(position);
    }
  }

  bool _isValidPosition(Position position) {
    return position.accuracy > 0 && position.accuracy <= 100;
  }

  bool _shouldSend(Position position) {
    final now = DateTime.now();
    final speed = position.speed < 0 ? 0.0 : position.speed;
    final interval = _intervalForSpeed(speed);
    final minDistance = _minDistanceForSpeed(speed);

    if (_lastSentAt == null || _lastSentPosition == null) {
      return true;
    }

    final elapsed = now.difference(_lastSentAt!);
    final moved = Geolocator.distanceBetween(
      _lastSentPosition!.latitude,
      _lastSentPosition!.longitude,
      position.latitude,
      position.longitude,
    );

    return elapsed >= interval || moved >= minDistance;
  }

  Duration _intervalForSpeed(double speed) {
    if (speed < 1) return const Duration(seconds: 3);
    if (speed < 2) return const Duration(milliseconds: 1500);
    if (speed < 10) return const Duration(seconds: 1);
    if (speed < 20) return const Duration(milliseconds: 500);
    return const Duration(milliseconds: 300);
  }

  double _minDistanceForSpeed(double speed) {
    if (speed < 1) return 0;
    if (speed < 2) return 5;
    if (speed < 10) return 8;
    if (speed < 20) return 10;
    return 12;
  }

  void _sendPosition(Position position) {
    final request = MatchRoadRequest(
      deviceId: _deviceId,
      sequenceId: Int64(++_sequence),
      timestamp: DateTime.now().toUtc().toIso8601String(),
      lat: position.latitude,
      lon: position.longitude,
      accuracy: position.accuracy,
      speed: position.speed < 0 ? 0 : position.speed,
      heading: position.heading < 0 ? 0 : position.heading,
      lastRoadId: Int64(_lastMatchedRoadId),
      history: _history,
    );

    final outgoing = _outgoing;
    if (outgoing == null || outgoing.isClosed) {
      setState(() => _status = 'Stream is not open');
      return;
    }

    outgoing.add(request);
    _sentAtBySequence[request.sequenceId.toInt()] = DateTime.now();

    setState(() {
      _lastSentAt = DateTime.now();
      _lastSentPosition = position;
      _status = 'Sent sequence ${request.sequenceId}';
    });
  }

  void _handleMatchResponse(MatchRoadResponse response) {
    final responseSequence = response.sequenceId.toInt();
    if (responseSequence < _lastAcceptedSequence) {
      return;
    }

    final best = response.hasBest() ? response.best : null;
    final receivedAt = DateTime.now();
    final sentAt = _sentAtBySequence.remove(responseSequence);
    final roundTripMs = sentAt == null
        ? null
        : receivedAt.difference(sentAt).inMicroseconds / 1000.0;
    _sentAtBySequence.removeWhere(
      (sequence, _) => sequence < responseSequence - 20,
    );

    setState(() {
      _lastAcceptedSequence = responseSequence;
      _lastResponse = response;
      _lastRoundTripMs = roundTripMs;
      _lastAcceptedAt = receivedAt;
      if (best != null) {
        _lastMatchedRoadId = best.roadId.toInt();
        if (response.confidence == 'high' || response.confidence == 'medium') {
          _lastConfidentRoad = best;
        }
      }
      _status = 'Accepted sequence $responseSequence';
    });
  }

  Future<void> _stopTracking() async {
    await _gpsSubscription?.cancel();
    _gpsSubscription = null;
    setState(() {
      _tracking = false;
      _status = 'GPS tracking stopped';
    });
  }

  @override
  void dispose() {
    _gpsSubscription?.cancel();
    _matchSubscription?.cancel();
    _outgoing?.close();
    _channel?.shutdown();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final best = _lastResponse?.best;
    final stableRoad = _lastConfidentRoad ?? best;

    return Scaffold(
      appBar: AppBar(
        title: const Text('Current Road'),
        actions: [
          IconButton(
            tooltip: 'Reconnect',
            onPressed: _connectStream,
            icon: const Icon(Icons.sync),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _StatusPanel(
            status: _status,
            backend: '$_backendHost:$_backendPort',
            tracking: _tracking,
            sequence: _sequence,
            acceptedSequence: _lastAcceptedSequence,
            roundTripMs: _lastRoundTripMs,
            backendProcessingMs: _lastResponse?.processingDurationMs,
            responseAge: _lastAcceptedAt == null
                ? null
                : DateTime.now().difference(_lastAcceptedAt!),
          ),
          const SizedBox(height: 12),
          _RoadPanel(road: stableRoad, response: _lastResponse),
          const SizedBox(height: 12),
          _GpsPanel(position: _position, historyCount: _history.length),
          const SizedBox(height: 12),
          _MapDebugPanel(position: _position, response: _lastResponse),
          const SizedBox(height: 12),
          _CandidateList(candidates: _lastResponse?.candidates ?? const []),
        ],
      ),
      bottomNavigationBar: SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
          child: FilledButton.icon(
            onPressed: _tracking ? _stopTracking : _startTracking,
            icon: Icon(_tracking ? Icons.stop : Icons.my_location),
            label: Text(_tracking ? 'Stop GPS' : 'Start GPS'),
          ),
        ),
      ),
    );
  }
}

class _StatusPanel extends StatelessWidget {
  const _StatusPanel({
    required this.status,
    required this.backend,
    required this.tracking,
    required this.sequence,
    required this.acceptedSequence,
    required this.roundTripMs,
    required this.backendProcessingMs,
    required this.responseAge,
  });

  final String status;
  final String backend;
  final bool tracking;
  final int sequence;
  final int acceptedSequence;
  final double? roundTripMs;
  final double? backendProcessingMs;
  final Duration? responseAge;

  @override
  Widget build(BuildContext context) {
    return _Panel(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _PanelTitle(
            icon: tracking ? Icons.sensors : Icons.sensors_off,
            text: 'Stream',
          ),
          _Metric(label: 'Status', value: status),
          _Metric(label: 'Backend', value: backend),
          _Metric(label: 'GPS', value: tracking ? 'on' : 'off'),
          _Metric(label: 'Sent', value: '$sequence'),
          _Metric(label: 'Accepted', value: '$acceptedSequence'),
          _Metric(label: 'Backend ms', value: _formatMs(backendProcessingMs)),
          _Metric(label: 'Round trip', value: _formatMs(roundTripMs)),
          _Metric(label: 'Age', value: _formatDuration(responseAge)),
        ],
      ),
    );
  }
}

String _formatMs(double? value) {
  if (value == null || value <= 0) {
    return '-';
  }
  return '${value.toStringAsFixed(1)} ms';
}

String _formatDuration(Duration? duration) {
  if (duration == null) {
    return '-';
  }
  final ms = duration.inMilliseconds;
  if (ms < 1000) {
    return '$ms ms';
  }
  return '${(ms / 1000).toStringAsFixed(1)} s';
}

class _RoadPanel extends StatelessWidget {
  const _RoadPanel({required this.road, required this.response});

  final RoadCandidate? road;
  final MatchRoadResponse? response;

  @override
  Widget build(BuildContext context) {
    final title = road == null
        ? 'No road yet'
        : '${road!.roadId}  ${road!.name.isNotEmpty ? road!.name : '(unnamed road)'}';
    final subtitle = road == null
        ? 'Waiting for backend match'
        : '${road!.highway}  confidence ${response?.confidence ?? '-'}';

    return _Panel(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const _PanelTitle(icon: Icons.route, text: 'Matched Road'),
          Text(title, style: Theme.of(context).textTheme.headlineSmall),
          const SizedBox(height: 4),
          Text(subtitle),
          if (road != null) ...[
            const SizedBox(height: 12),
            _Metric(label: 'Road ID', value: '${road!.roadId}'),
            _Metric(
              label: 'Distance',
              value: '${road!.distanceMeters.toStringAsFixed(1)} m',
            ),
            _Metric(label: 'Score', value: road!.score.toStringAsFixed(3)),
            _Metric(label: 'Connectivity', value: road!.connectivity),
          ],
        ],
      ),
    );
  }
}

class _GpsPanel extends StatelessWidget {
  const _GpsPanel({required this.position, required this.historyCount});

  final Position? position;
  final int historyCount;

  @override
  Widget build(BuildContext context) {
    return _Panel(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const _PanelTitle(icon: Icons.gps_fixed, text: 'GPS'),
          if (position == null)
            const Text('No GPS point yet')
          else ...[
            _Metric(label: 'Lat', value: position!.latitude.toStringAsFixed(6)),
            _Metric(
              label: 'Lon',
              value: position!.longitude.toStringAsFixed(6),
            ),
            _Metric(
              label: 'Accuracy',
              value: '${position!.accuracy.toStringAsFixed(1)} m',
            ),
            _Metric(
              label: 'Speed',
              value: '${(position!.speed * 3.6).toStringAsFixed(1)} km/h',
            ),
            _Metric(
              label: 'Heading',
              value: '${position!.heading.toStringAsFixed(1)} deg',
            ),
          ],
          _Metric(label: 'History', value: '$historyCount / 10'),
        ],
      ),
    );
  }
}

class _MapDebugPanel extends StatelessWidget {
  const _MapDebugPanel({required this.position, required this.response});

  static const _fallbackCenter = LatLng(21.027763, 105.834160);

  final Position? position;
  final MatchRoadResponse? response;

  @override
  Widget build(BuildContext context) {
    final currentPoint = position == null
        ? null
        : LatLng(position!.latitude, position!.longitude);
    final center = currentPoint ?? _firstGeometryPoint() ?? _fallbackCenter;
    final centerKey =
        '${center.latitude.toStringAsFixed(5)},'
        '${center.longitude.toStringAsFixed(5)},'
        '${response?.sequenceId ?? 0}';

    return _Panel(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const _PanelTitle(icon: Icons.map, text: 'Map Debug'),
          ClipRRect(
            borderRadius: BorderRadius.circular(8),
            child: SizedBox(
              height: 280,
              child: FlutterMap(
                key: ValueKey(centerKey),
                options: MapOptions(
                  initialCenter: center,
                  initialZoom: 16,
                  interactionOptions: const InteractionOptions(
                    flags: InteractiveFlag.all & ~InteractiveFlag.rotate,
                  ),
                ),
                children: [
                  TileLayer(
                    urlTemplate:
                        'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
                    userAgentPackageName: 'com.example.flutter_app',
                  ),
                  PolylineLayer(polylines: _polylines()),
                  if (currentPoint != null)
                    MarkerLayer(
                      markers: [
                        Marker(
                          point: currentPoint,
                          width: 44,
                          height: 44,
                          child: const Icon(
                            Icons.my_location,
                            color: Colors.blue,
                            size: 34,
                          ),
                        ),
                      ],
                    ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 10),
          const Row(
            children: [
              _LegendSwatch(color: Colors.blue, label: 'GPS'),
              SizedBox(width: 12),
              _LegendSwatch(color: Colors.green, label: 'Best'),
              SizedBox(width: 12),
              _LegendSwatch(color: Colors.amber, label: 'Candidate'),
            ],
          ),
        ],
      ),
    );
  }

  LatLng? _firstGeometryPoint() {
    final candidates = response?.candidates;
    if (candidates == null) {
      return null;
    }
    for (final candidate in candidates) {
      if (candidate.geometry.isNotEmpty) {
        final point = candidate.geometry.first;
        return LatLng(point.lat, point.lon);
      }
    }
    return null;
  }

  List<Polyline> _polylines() {
    final candidates = response?.candidates ?? const <RoadCandidate>[];
    final bestRoadId = response?.hasBest() == true
        ? response!.best.roadId
        : null;

    return [
      for (final candidate in candidates)
        if (candidate.geometry.length >= 2)
          Polyline(
            points: [
              for (final point in candidate.geometry)
                LatLng(point.lat, point.lon),
            ],
            color: candidate.roadId == bestRoadId ? Colors.green : Colors.amber,
            strokeWidth: candidate.roadId == bestRoadId ? 5 : 3,
          ),
    ];
  }
}

class _LegendSwatch extends StatelessWidget {
  const _LegendSwatch({required this.color, required this.label});

  final Color color;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          width: 12,
          height: 12,
          decoration: BoxDecoration(
            color: color,
            borderRadius: BorderRadius.circular(2),
          ),
        ),
        const SizedBox(width: 5),
        Text(label),
      ],
    );
  }
}

class _CandidateList extends StatelessWidget {
  const _CandidateList({required this.candidates});

  final List<RoadCandidate> candidates;

  @override
  Widget build(BuildContext context) {
    return _Panel(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const _PanelTitle(icon: Icons.alt_route, text: 'Candidates'),
          if (candidates.isEmpty)
            const Text('No candidates yet')
          else
            for (final candidate in candidates)
              Padding(
                padding: const EdgeInsets.only(top: 10),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    SizedBox(
                      width: 44,
                      child: Text(candidate.score.toStringAsFixed(2)),
                    ),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            '${candidate.roadId}  ${candidate.name.isNotEmpty ? candidate.name : '(unnamed road)'}',
                            style: const TextStyle(fontWeight: FontWeight.w600),
                          ),
                          Text(
                            '${candidate.highway}  ${candidate.distanceMeters.toStringAsFixed(1)} m  ${candidate.connectivity}',
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
        ],
      ),
    );
  }
}

class _Panel extends StatelessWidget {
  const _Panel({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: Theme.of(context).colorScheme.outlineVariant),
      ),
      child: Padding(padding: const EdgeInsets.all(14), child: child),
    );
  }
}

class _PanelTitle extends StatelessWidget {
  const _PanelTitle({required this.icon, required this.text});

  final IconData icon;
  final String text;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Row(
        children: [
          Icon(icon, size: 18),
          const SizedBox(width: 8),
          Text(text, style: Theme.of(context).textTheme.titleMedium),
        ],
      ),
    );
  }
}

class _Metric extends StatelessWidget {
  const _Metric({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(top: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 96,
            child: Text(
              label,
              style: const TextStyle(fontWeight: FontWeight.w600),
            ),
          ),
          Expanded(child: Text(value)),
        ],
      ),
    );
  }
}
