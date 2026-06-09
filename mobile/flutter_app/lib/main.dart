import 'dart:async';
import 'dart:convert';
import 'dart:io' show Platform;

import 'package:fixnum/fixnum.dart';
import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:geolocator/geolocator.dart';
import 'package:grpc/grpc.dart';
import 'package:latlong2/latlong.dart';
import 'package:shared_preferences/shared_preferences.dart';

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
  WidgetsFlutterBinding.ensureInitialized();
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

class BackendConfig {
  const BackendConfig({
    required this.host,
    required this.port,
    required this.deviceId,
  });

  final String host;
  final int port;
  final String deviceId;

  Map<String, Object?> toJson() {
    return {'host': host, 'port': port, 'deviceId': deviceId};
  }

  static BackendConfig? fromJson(Object? value) {
    if (value is! Map) {
      return null;
    }
    final host = value['host'];
    final port = value['port'];
    final deviceId = value['deviceId'];
    if (host is! String || host.trim().isEmpty) {
      return null;
    }
    if (port is! int || port <= 0 || port > 65535) {
      return null;
    }
    if (deviceId is! String || deviceId.trim().isEmpty) {
      return null;
    }
    return BackendConfig(
      host: host.trim(),
      port: port,
      deviceId: deviceId.trim(),
    );
  }
}

class AppSettings {
  const AppSettings({required this.backendConfig, required this.sendPolicy});

  final BackendConfig backendConfig;
  final SendPolicy sendPolicy;

  Map<String, Object?> toJson() {
    return {
      'backendConfig': backendConfig.toJson(),
      'sendPolicy': sendPolicy.toJson(),
    };
  }

  static AppSettings? fromJson(Object? value) {
    if (value is! Map) {
      return null;
    }
    final backendConfig = BackendConfig.fromJson(value['backendConfig']);
    final sendPolicy = SendPolicy.fromJson(value['sendPolicy']);
    if (backendConfig == null || sendPolicy == null) {
      return null;
    }
    return AppSettings(backendConfig: backendConfig, sendPolicy: sendPolicy);
  }
}

class SendPolicy {
  const SendPolicy({required this.rules});

  final List<SendPolicyRule> rules;

  Map<String, Object?> toJson() {
    return {
      'rules': [for (final rule in rules) rule.toJson()],
    };
  }

  static SendPolicy? fromJson(Object? value) {
    if (value is! Map) {
      return null;
    }
    final rawRules = value['rules'];
    if (rawRules is! List || rawRules.isEmpty) {
      return null;
    }

    final rules = <SendPolicyRule>[];
    for (final rawRule in rawRules) {
      final rule = SendPolicyRule.fromJson(rawRule);
      if (rule == null) {
        return null;
      }
      rules.add(rule);
    }
    if (rules.last.maxSpeedMetersPerSecond != null) {
      return null;
    }
    for (var i = 1; i < rules.length; i++) {
      final previous = rules[i - 1].maxSpeedMetersPerSecond;
      final current = rules[i].maxSpeedMetersPerSecond;
      if (previous != null && current != null && current <= previous) {
        return null;
      }
    }
    return SendPolicy(rules: rules);
  }

  factory SendPolicy.defaults() {
    return const SendPolicy(
      rules: [
        SendPolicyRule(
          maxSpeedMetersPerSecond: 1,
          intervalMs: 3000,
          minDistanceMeters: 0,
        ),
        SendPolicyRule(
          maxSpeedMetersPerSecond: 2,
          intervalMs: 1500,
          minDistanceMeters: 5,
        ),
        SendPolicyRule(
          maxSpeedMetersPerSecond: 10,
          intervalMs: 1000,
          minDistanceMeters: 8,
        ),
        SendPolicyRule(
          maxSpeedMetersPerSecond: 20,
          intervalMs: 500,
          minDistanceMeters: 10,
        ),
        SendPolicyRule(
          maxSpeedMetersPerSecond: null,
          intervalMs: 300,
          minDistanceMeters: 12,
        ),
      ],
    );
  }

  SendPolicyRule ruleForSpeed(double speedMetersPerSecond) {
    for (final rule in rules) {
      final maxSpeed = rule.maxSpeedMetersPerSecond;
      if (maxSpeed == null || speedMetersPerSecond < maxSpeed) {
        return rule;
      }
    }
    return rules.last;
  }
}

class SendPolicyRule {
  const SendPolicyRule({
    required this.maxSpeedMetersPerSecond,
    required this.intervalMs,
    required this.minDistanceMeters,
  });

  final double? maxSpeedMetersPerSecond;
  final int intervalMs;
  final double minDistanceMeters;

  Duration get interval => Duration(milliseconds: intervalMs);

  Map<String, Object?> toJson() {
    return {
      'maxSpeedMetersPerSecond': maxSpeedMetersPerSecond,
      'intervalMs': intervalMs,
      'minDistanceMeters': minDistanceMeters,
    };
  }

  static SendPolicyRule? fromJson(Object? value) {
    if (value is! Map) {
      return null;
    }
    final maxSpeed = value['maxSpeedMetersPerSecond'];
    final intervalMs = value['intervalMs'];
    final minDistance = value['minDistanceMeters'];
    if (maxSpeed != null && (maxSpeed is! num || maxSpeed <= 0)) {
      return null;
    }
    if (intervalMs is! int || intervalMs <= 0) {
      return null;
    }
    if (minDistance is! num || minDistance < 0) {
      return null;
    }
    return SendPolicyRule(
      maxSpeedMetersPerSecond: maxSpeed?.toDouble(),
      intervalMs: intervalMs,
      minDistanceMeters: minDistance.toDouble(),
    );
  }
}

class SettingsStore {
  static const _settingsKey = 'road_detector.settings.v1';

  static Future<AppSettings?> load() async {
    final prefs = await SharedPreferences.getInstance();
    final raw = prefs.getString(_settingsKey);
    if (raw == null || raw.isEmpty) {
      return null;
    }

    try {
      return AppSettings.fromJson(jsonDecode(raw));
    } catch (error) {
      debugPrint('Failed to load settings: $error');
      return null;
    }
  }

  static Future<void> save(AppSettings settings) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_settingsKey, jsonEncode(settings.toJson()));
  }
}

class _RoadDetectorScreenState extends State<RoadDetectorScreen> {
  final List<GpsHistoryPoint> _history = [];
  late BackendConfig _backendConfig;
  SendPolicy _sendPolicy = SendPolicy.defaults();
  StreamController<MatchRoadRequest>? _outgoing;

  ClientChannel? _channel;
  StreamSubscription<Position>? _gpsSubscription;
  StreamSubscription<MatchRoadResponse>? _matchSubscription;
  Timer? _reconnectTimer;
  Timer? _responseWatchdogTimer;

  Position? _position;
  MatchRoadResponse? _lastResponse;
  RoadCandidate? _lastConfidentRoad;
  int _sequence = 0;
  int _lastAcceptedSequence = 0;
  int _receivedResponses = 0;
  int _lastMatchedRoadId = 0;
  final Map<int, DateTime> _sentAtBySequence = {};
  double? _lastRoundTripMs;
  DateTime? _lastAcceptedAt;
  DateTime? _lastSentAt;
  Position? _lastSentPosition;
  String _status = 'Disconnected';
  String _backendStatus = 'Disconnected';
  bool _tracking = false;
  bool _backgroundTrackingLimited = false;
  bool _connectingBackend = false;
  int _reconnectAttempt = 0;
  int _streamToken = 0;
  String? _connectedBackend;

  Future<void> _closeBackendResources() async {
    final matchSubscription = _matchSubscription;
    final outgoing = _outgoing;
    final channel = _channel;

    _streamToken++;
    _matchSubscription = null;
    _outgoing = null;
    _channel = null;
    _connectedBackend = null;
    _backendStatus = 'Disconnected';
    _sentAtBySequence.clear();
    _responseWatchdogTimer?.cancel();
    _responseWatchdogTimer = null;

    await _bestEffortClose(
      matchSubscription?.cancel(),
      label: 'match subscription',
    );
    await _bestEffortClose(outgoing?.close(), label: 'outgoing stream');
    await _bestEffortClose(channel?.shutdown(), label: 'gRPC channel');
  }

  Future<void> _closeActiveResources() async {
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
    _responseWatchdogTimer?.cancel();
    _responseWatchdogTimer = null;

    final gpsSubscription = _gpsSubscription;
    _gpsSubscription = null;

    await _bestEffortClose(
      gpsSubscription?.cancel(),
      label: 'GPS subscription',
    );
    await _closeBackendResources();
  }

  Future<void> _bestEffortClose(
    Future<void>? close, {
    required String label,
  }) async {
    if (close == null) {
      return;
    }
    try {
      await close.timeout(const Duration(seconds: 1));
    } on TimeoutException {
      debugPrint('Timed out closing $label');
    } catch (error) {
      debugPrint('Failed closing $label: $error');
    }
  }

  @override
  void initState() {
    super.initState();
    _backendConfig = BackendConfig(
      host: _initialBackendHost(),
      port: _backendPort,
      deviceId: _deviceId,
    );
    unawaited(_loadSavedSettings());
  }

  Future<void> _loadSavedSettings() async {
    final saved = await SettingsStore.load();
    if (saved == null || !mounted || _tracking) {
      return;
    }

    setState(() {
      _backendConfig = saved.backendConfig;
      _sendPolicy = saved.sendPolicy;
      _status = 'Settings loaded';
    });
  }

  String _initialBackendHost() {
    if (_defaultBackendHost != '127.0.0.1') {
      return _defaultBackendHost;
    }
    if (Platform.isAndroid) {
      return '10.0.2.2';
    }
    return _defaultBackendHost;
  }

  Future<bool> _connectStream({bool reconnecting = false}) async {
    if (_connectingBackend) {
      return _outgoing != null && !_outgoing!.isClosed;
    }

    _connectingBackend = true;
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
    try {
      await _closeBackendResources();

      final outgoing = StreamController<MatchRoadRequest>();

      final channel = ClientChannel(
        _backendConfig.host,
        port: _backendConfig.port,
        options: const ChannelOptions(
          credentials: ChannelCredentials.insecure(),
        ),
      );

      final client = RoadMatcherClient(channel);
      final responseStream = client.streamGps(outgoing.stream);
      final streamToken = ++_streamToken;

      _matchSubscription = responseStream.listen(
        _handleMatchResponse,
        onError: (Object error) {
          _handleBackendDisconnected(streamToken, 'Stream error: $error');
        },
        onDone: () {
          _handleBackendDisconnected(streamToken, 'Stream closed');
        },
      );

      setState(() {
        _outgoing = outgoing;
        _channel = channel;
        _connectedBackend = '${_backendConfig.host}:${_backendConfig.port}';
        _backendStatus = reconnecting ? 'Reconnected' : 'Stream opened';
        _status = reconnecting
            ? 'Reconnected to $_connectedBackend'
            : 'Stream opened to $_connectedBackend';
      });
      return true;
    } catch (error) {
      if (mounted) {
        setState(() => _status = 'Connect failed: $error');
      }
      return false;
    } finally {
      _connectingBackend = false;
    }
  }

  void _handleBackendDisconnected(int streamToken, String status) {
    if (streamToken != _streamToken) {
      return;
    }

    unawaited(_closeBackendResources());
    if (!mounted) {
      return;
    }

    setState(() {
      _backendStatus = status;
      _status = status;
    });

    if (_tracking) {
      _scheduleReconnect();
    }
  }

  void _scheduleReconnect() {
    if (!_tracking || _reconnectTimer != null) {
      return;
    }

    final delay = _nextReconnectDelay();
    setState(() {
      final reconnectStatus = 'Reconnecting in ${_formatDuration(delay)}';
      _backendStatus = reconnectStatus;
      _status = 'Backend disconnected; $reconnectStatus';
    });

    _reconnectTimer = Timer(delay, () {
      _reconnectTimer = null;
      if (!_tracking || !mounted) {
        return;
      }
      setState(() {
        _backendStatus = 'Reconnecting';
        _status = 'Reconnecting backend';
      });
      unawaited(_reconnectBackend());
    });
  }

  Duration _nextReconnectDelay() {
    const delays = [
      Duration(seconds: 1),
      Duration(seconds: 2),
      Duration(seconds: 5),
      Duration(seconds: 10),
      Duration(seconds: 30),
    ];
    final index = _reconnectAttempt.clamp(0, delays.length - 1);
    _reconnectAttempt++;
    return delays[index];
  }

  Future<void> _reconnectBackend() async {
    final connected = await _connectStream(reconnecting: true);
    if (!connected || !_tracking) {
      if (mounted && _tracking) {
        _scheduleReconnect();
      }
      return;
    }

    final latestPosition = _position;
    if (latestPosition != null && _isValidPosition(latestPosition)) {
      _sendPosition(latestPosition, force: true);
    }
  }

  Future<void> _startTracking() async {
    final connected = await _connectStream();
    if (!connected) {
      return;
    }

    final allowed = await _requestLocationPermission();
    if (!allowed) {
      setState(() => _status = 'Location permission unavailable');
      return;
    }

    await _gpsSubscription?.cancel();
    _gpsSubscription =
        Geolocator.getPositionStream(
          locationSettings: _locationSettings(),
        ).listen(
          _handlePosition,
          onError: (Object error) {
            if (mounted) {
              setState(() => _status = 'GPS error: $error');
            }
          },
        );

    setState(() {
      _tracking = true;
      _status = _backgroundTrackingLimited
          ? 'Tracking GPS; iOS screen-off needs Location Always'
          : 'Tracking GPS';
    });
  }

  LocationSettings _locationSettings() {
    if (Platform.isAndroid) {
      return AndroidSettings(
        accuracy: LocationAccuracy.bestForNavigation,
        distanceFilter: 5,
        intervalDuration: const Duration(milliseconds: 300),
        foregroundNotificationConfig: const ForegroundNotificationConfig(
          notificationTitle: 'Road Detector is tracking',
          notificationText: 'GPS road matching is active for this trip.',
          notificationChannelName: 'Road Detector tracking',
          enableWakeLock: true,
          setOngoing: true,
        ),
      );
    }

    if (Platform.isIOS) {
      return AppleSettings(
        accuracy: LocationAccuracy.bestForNavigation,
        distanceFilter: 5,
        activityType: ActivityType.automotiveNavigation,
        pauseLocationUpdatesAutomatically: false,
        showBackgroundLocationIndicator: true,
        allowBackgroundLocationUpdates: true,
      );
    }

    return const LocationSettings(
      accuracy: LocationAccuracy.bestForNavigation,
      distanceFilter: 5,
    );
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

    if (Platform.isIOS && permission == LocationPermission.whileInUse) {
      permission = await Geolocator.requestPermission();
    }

    _backgroundTrackingLimited =
        Platform.isIOS && permission == LocationPermission.whileInUse;

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
    final rule = _sendPolicy.ruleForSpeed(speed);

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

    return elapsed >= rule.interval || moved >= rule.minDistanceMeters;
  }

  void _sendPosition(Position position, {bool force = false}) {
    final outgoing = _outgoing;
    if (outgoing == null || outgoing.isClosed) {
      if (_tracking) {
        _scheduleReconnect();
      } else {
        setState(() {
          _backendStatus = 'Disconnected';
          _status = 'Backend disconnected';
        });
      }
      return;
    }

    final request = MatchRoadRequest(
      deviceId: _backendConfig.deviceId,
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

    try {
      outgoing.add(request);
    } on StateError {
      if (_tracking) {
        _scheduleReconnect();
      } else {
        setState(() {
          _backendStatus = 'Stream closed';
          _status = 'Backend stream closed';
        });
      }
      return;
    } catch (error) {
      if (_tracking) {
        _scheduleReconnect();
      } else {
        setState(() {
          _backendStatus = 'Send failed';
          _status = 'Send failed: $error';
        });
      }
      return;
    }

    _sentAtBySequence[request.sequenceId.toInt()] = DateTime.now();
    _scheduleResponseWatchdog(request.sequenceId.toInt());

    setState(() {
      _lastSentAt = DateTime.now();
      _lastSentPosition = position;
      _backendStatus = 'Sent sequence ${request.sequenceId}';
      _status = force
          ? 'Sent latest sequence ${request.sequenceId} after reconnect'
          : 'Sent sequence ${request.sequenceId}';
    });
  }

  void _scheduleResponseWatchdog(int sequence) {
    _responseWatchdogTimer?.cancel();
    _responseWatchdogTimer = Timer(const Duration(seconds: 10), () {
      if (!_tracking ||
          _outgoing == null ||
          _lastAcceptedSequence >= sequence) {
        return;
      }
      _handleBackendDisconnected(_streamToken, 'No backend response for 10s');
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
    if (_sentAtBySequence.isEmpty) {
      _responseWatchdogTimer?.cancel();
      _responseWatchdogTimer = null;
    }

    setState(() {
      _reconnectAttempt = 0;
      _lastAcceptedSequence = responseSequence;
      _receivedResponses++;
      _lastResponse = response;
      _lastRoundTripMs = roundTripMs;
      _lastAcceptedAt = receivedAt;
      _backendStatus = 'Received sequence $responseSequence';
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
    setState(() {
      _tracking = false;
      _connectedBackend = null;
      _backendStatus = 'Disconnected';
      _status = 'GPS tracking stopped';
    });
    unawaited(_closeActiveResources());
  }

  @override
  void dispose() {
    unawaited(_closeActiveResources());
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
            onPressed: _tracking ? null : _connectStream,
            icon: const Icon(Icons.sync),
          ),
          IconButton(
            tooltip: 'Settings',
            onPressed: _tracking ? null : _openSettingsScreen,
            icon: const Icon(Icons.tune),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _StatusPanel(
            status: _status,
            backend:
                _connectedBackend ??
                '${_backendConfig.host}:${_backendConfig.port}',
            tracking: _tracking,
            sequence: _sequence,
            acceptedSequence: _lastAcceptedSequence,
            receivedResponses: _receivedResponses,
            pendingResponses: _sentAtBySequence.length,
            backendStatus: _backendStatus,
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
          _MapDebugPanel(position: _position),
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

  Future<void> _openSettingsScreen() async {
    final updated = await Navigator.of(context).push<AppSettings>(
      MaterialPageRoute(
        builder: (_) => SettingsScreen(
          backendConfig: _backendConfig,
          sendPolicy: _sendPolicy,
        ),
      ),
    );
    if (updated == null || !mounted) {
      return;
    }
    await SettingsStore.save(updated);
    if (!mounted) {
      return;
    }
    setState(() {
      _backendConfig = updated.backendConfig;
      _sendPolicy = updated.sendPolicy;
      _connectedBackend = null;
      _status = 'Settings updated';
    });
  }
}

class _StatusPanel extends StatelessWidget {
  const _StatusPanel({
    required this.status,
    required this.backend,
    required this.backendStatus,
    required this.tracking,
    required this.sequence,
    required this.acceptedSequence,
    required this.receivedResponses,
    required this.pendingResponses,
    required this.roundTripMs,
    required this.backendProcessingMs,
    required this.responseAge,
  });

  final String status;
  final String backend;
  final String backendStatus;
  final bool tracking;
  final int sequence;
  final int acceptedSequence;
  final int receivedResponses;
  final int pendingResponses;
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
          _Metric(label: 'Backend state', value: backendStatus),
          _Metric(label: 'GPS', value: tracking ? 'on' : 'off'),
          _Metric(label: 'Sent', value: '$sequence'),
          _Metric(label: 'Received', value: '$receivedResponses'),
          _Metric(label: 'Last accepted', value: '$acceptedSequence'),
          _Metric(label: 'Pending', value: '$pendingResponses'),
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

class SettingsScreen extends StatefulWidget {
  const SettingsScreen({
    super.key,
    required this.backendConfig,
    required this.sendPolicy,
  });

  final BackendConfig backendConfig;
  final SendPolicy sendPolicy;

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  late final TextEditingController _hostController;
  late final TextEditingController _portController;
  late final TextEditingController _deviceIdController;
  late final List<_SendPolicyRuleControllers> _sendPolicyControllers;
  String? _error;

  @override
  void initState() {
    super.initState();
    _hostController = TextEditingController(text: widget.backendConfig.host);
    _portController = TextEditingController(
      text: '${widget.backendConfig.port}',
    );
    _deviceIdController = TextEditingController(
      text: widget.backendConfig.deviceId,
    );
    _sendPolicyControllers = [
      for (final rule in widget.sendPolicy.rules)
        _SendPolicyRuleControllers.fromRule(rule),
    ];
  }

  @override
  void dispose() {
    _hostController.dispose();
    _portController.dispose();
    _deviceIdController.dispose();
    for (final controller in _sendPolicyControllers) {
      controller.dispose();
    }
    super.dispose();
  }

  void _save() {
    final host = _hostController.text.trim();
    final port = int.tryParse(_portController.text.trim());
    final deviceId = _deviceIdController.text.trim();
    if (host.isEmpty) {
      setState(() => _error = 'Backend host is required');
      return;
    }
    if (port == null || port <= 0 || port > 65535) {
      setState(() => _error = 'Backend port must be 1-65535');
      return;
    }
    if (deviceId.isEmpty) {
      setState(() => _error = 'Device ID is required');
      return;
    }

    final rules = <SendPolicyRule>[];
    for (var i = 0; i < _sendPolicyControllers.length; i++) {
      final controller = _sendPolicyControllers[i];
      final maxSpeed = i == _sendPolicyControllers.length - 1
          ? null
          : double.tryParse(controller.maxSpeed.text.trim());
      final intervalMs = int.tryParse(controller.intervalMs.text.trim());
      final minDistance = double.tryParse(controller.minDistance.text.trim());

      if (i != _sendPolicyControllers.length - 1 &&
          (maxSpeed == null || maxSpeed <= 0)) {
        setState(() => _error = 'Invalid max speed in band ${i + 1}');
        return;
      }
      if (intervalMs == null || intervalMs <= 0) {
        setState(() => _error = 'Invalid interval in band ${i + 1}');
        return;
      }
      if (minDistance == null || minDistance < 0) {
        setState(() => _error = 'Invalid min distance in band ${i + 1}');
        return;
      }
      if (rules.isNotEmpty &&
          maxSpeed != null &&
          rules.last.maxSpeedMetersPerSecond != null &&
          maxSpeed <= rules.last.maxSpeedMetersPerSecond!) {
        setState(() => _error = 'Speed bands must increase');
        return;
      }

      rules.add(
        SendPolicyRule(
          maxSpeedMetersPerSecond: maxSpeed,
          intervalMs: intervalMs,
          minDistanceMeters: minDistance,
        ),
      );
    }

    Navigator.of(context).pop(
      AppSettings(
        backendConfig: BackendConfig(
          host: host,
          port: port,
          deviceId: deviceId,
        ),
        sendPolicy: SendPolicy(rules: rules),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Settings'),
        actions: [TextButton(onPressed: _save, child: const Text('Save'))],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          if (_error != null) ...[
            Text(
              _error!,
              style: TextStyle(color: Theme.of(context).colorScheme.error),
            ),
            const SizedBox(height: 12),
          ],
          _BackendSettingsEditor(
            hostController: _hostController,
            portController: _portController,
            deviceIdController: _deviceIdController,
          ),
          const SizedBox(height: 12),
          const _PanelTitle(icon: Icons.speed, text: 'Send Policy'),
          for (var i = 0; i < _sendPolicyControllers.length; i++) ...[
            _SendPolicyRuleEditor(
              index: i,
              isLast: i == _sendPolicyControllers.length - 1,
              controllers: _sendPolicyControllers[i],
            ),
            if (i != _sendPolicyControllers.length - 1)
              const SizedBox(height: 12),
          ],
        ],
      ),
    );
  }
}

class _BackendSettingsEditor extends StatelessWidget {
  const _BackendSettingsEditor({
    required this.hostController,
    required this.portController,
    required this.deviceIdController,
  });

  final TextEditingController hostController;
  final TextEditingController portController;
  final TextEditingController deviceIdController;

  @override
  Widget build(BuildContext context) {
    return _Panel(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const _PanelTitle(icon: Icons.dns, text: 'Backend'),
          TextField(
            controller: hostController,
            decoration: const InputDecoration(
              labelText: 'Host',
              border: OutlineInputBorder(),
              isDense: true,
            ),
            keyboardType: TextInputType.url,
            textInputAction: TextInputAction.next,
          ),
          const SizedBox(height: 10),
          Row(
            children: [
              Expanded(
                child: TextField(
                  controller: portController,
                  decoration: const InputDecoration(
                    labelText: 'Port',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                  keyboardType: TextInputType.number,
                  textInputAction: TextInputAction.next,
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                flex: 2,
                child: TextField(
                  controller: deviceIdController,
                  decoration: const InputDecoration(
                    labelText: 'Device ID',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                  textInputAction: TextInputAction.done,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _SendPolicyRuleControllers {
  _SendPolicyRuleControllers({
    required this.maxSpeed,
    required this.intervalMs,
    required this.minDistance,
  });

  factory _SendPolicyRuleControllers.fromRule(SendPolicyRule rule) {
    return _SendPolicyRuleControllers(
      maxSpeed: TextEditingController(
        text: rule.maxSpeedMetersPerSecond?.toStringAsFixed(1) ?? '',
      ),
      intervalMs: TextEditingController(text: '${rule.intervalMs}'),
      minDistance: TextEditingController(
        text: rule.minDistanceMeters.toStringAsFixed(1),
      ),
    );
  }

  final TextEditingController maxSpeed;
  final TextEditingController intervalMs;
  final TextEditingController minDistance;

  void dispose() {
    maxSpeed.dispose();
    intervalMs.dispose();
    minDistance.dispose();
  }
}

class _SendPolicyRuleEditor extends StatelessWidget {
  const _SendPolicyRuleEditor({
    required this.index,
    required this.isLast,
    required this.controllers,
  });

  final int index;
  final bool isLast;
  final _SendPolicyRuleControllers controllers;

  @override
  Widget build(BuildContext context) {
    return _Panel(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _PanelTitle(icon: Icons.speed, text: 'Band ${index + 1}'),
          TextField(
            controller: controllers.maxSpeed,
            enabled: !isLast,
            decoration: InputDecoration(
              labelText: isLast ? 'Speed' : 'Max speed m/s',
              hintText: isLast ? 'Any faster speed' : null,
              border: const OutlineInputBorder(),
              isDense: true,
            ),
            keyboardType: TextInputType.number,
            textInputAction: TextInputAction.next,
          ),
          const SizedBox(height: 10),
          Row(
            children: [
              Expanded(
                child: TextField(
                  controller: controllers.intervalMs,
                  decoration: const InputDecoration(
                    labelText: 'Interval ms',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                  keyboardType: TextInputType.number,
                  textInputAction: TextInputAction.next,
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: TextField(
                  controller: controllers.minDistance,
                  decoration: const InputDecoration(
                    labelText: 'Min distance m',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                  keyboardType: TextInputType.number,
                  textInputAction: TextInputAction.next,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
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
        : 'confidence ${response?.confidence ?? '-'}';

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
            _Metric(label: 'Score', value: road!.score.toStringAsFixed(3)),
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
  const _MapDebugPanel({required this.position});

  static const _fallbackCenter = LatLng(21.027763, 105.834160);

  final Position? position;

  @override
  Widget build(BuildContext context) {
    final currentPoint = position == null
        ? null
        : LatLng(position!.latitude, position!.longitude);
    final center = currentPoint ?? _fallbackCenter;
    final centerKey =
        '${center.latitude.toStringAsFixed(5)},'
        '${center.longitude.toStringAsFixed(5)}';

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
        ],
      ),
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
