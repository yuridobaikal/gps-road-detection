// This is a generated file - do not edit.
//
// Generated from road_matcher.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

class MatchRoadRequest extends $pb.GeneratedMessage {
  factory MatchRoadRequest({
    $core.String? deviceId,
    $fixnum.Int64? sequenceId,
    $core.String? timestamp,
    $core.double? lat,
    $core.double? lon,
    $core.double? accuracy,
    $core.double? speed,
    $core.double? heading,
    $fixnum.Int64? lastRoadId,
    $core.Iterable<GpsHistoryPoint>? history,
  }) {
    final result = create();
    if (deviceId != null) result.deviceId = deviceId;
    if (sequenceId != null) result.sequenceId = sequenceId;
    if (timestamp != null) result.timestamp = timestamp;
    if (lat != null) result.lat = lat;
    if (lon != null) result.lon = lon;
    if (accuracy != null) result.accuracy = accuracy;
    if (speed != null) result.speed = speed;
    if (heading != null) result.heading = heading;
    if (lastRoadId != null) result.lastRoadId = lastRoadId;
    if (history != null) result.history.addAll(history);
    return result;
  }

  MatchRoadRequest._();

  factory MatchRoadRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory MatchRoadRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'MatchRoadRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'roadmatcher.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'deviceId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'sequenceId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(3, _omitFieldNames ? '' : 'timestamp')
    ..aD(4, _omitFieldNames ? '' : 'lat')
    ..aD(5, _omitFieldNames ? '' : 'lon')
    ..aD(6, _omitFieldNames ? '' : 'accuracy')
    ..aD(7, _omitFieldNames ? '' : 'speed')
    ..aD(8, _omitFieldNames ? '' : 'heading')
    ..aInt64(9, _omitFieldNames ? '' : 'lastRoadId')
    ..pPM<GpsHistoryPoint>(10, _omitFieldNames ? '' : 'history',
        subBuilder: GpsHistoryPoint.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  MatchRoadRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  MatchRoadRequest copyWith(void Function(MatchRoadRequest) updates) =>
      super.copyWith((message) => updates(message as MatchRoadRequest))
          as MatchRoadRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static MatchRoadRequest create() => MatchRoadRequest._();
  @$core.override
  MatchRoadRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static MatchRoadRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<MatchRoadRequest>(create);
  static MatchRoadRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get deviceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set deviceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDeviceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDeviceId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get sequenceId => $_getI64(1);
  @$pb.TagNumber(2)
  set sequenceId($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSequenceId() => $_has(1);
  @$pb.TagNumber(2)
  void clearSequenceId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get timestamp => $_getSZ(2);
  @$pb.TagNumber(3)
  set timestamp($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasTimestamp() => $_has(2);
  @$pb.TagNumber(3)
  void clearTimestamp() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.double get lat => $_getN(3);
  @$pb.TagNumber(4)
  set lat($core.double value) => $_setDouble(3, value);
  @$pb.TagNumber(4)
  $core.bool hasLat() => $_has(3);
  @$pb.TagNumber(4)
  void clearLat() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.double get lon => $_getN(4);
  @$pb.TagNumber(5)
  set lon($core.double value) => $_setDouble(4, value);
  @$pb.TagNumber(5)
  $core.bool hasLon() => $_has(4);
  @$pb.TagNumber(5)
  void clearLon() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.double get accuracy => $_getN(5);
  @$pb.TagNumber(6)
  set accuracy($core.double value) => $_setDouble(5, value);
  @$pb.TagNumber(6)
  $core.bool hasAccuracy() => $_has(5);
  @$pb.TagNumber(6)
  void clearAccuracy() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.double get speed => $_getN(6);
  @$pb.TagNumber(7)
  set speed($core.double value) => $_setDouble(6, value);
  @$pb.TagNumber(7)
  $core.bool hasSpeed() => $_has(6);
  @$pb.TagNumber(7)
  void clearSpeed() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.double get heading => $_getN(7);
  @$pb.TagNumber(8)
  set heading($core.double value) => $_setDouble(7, value);
  @$pb.TagNumber(8)
  $core.bool hasHeading() => $_has(7);
  @$pb.TagNumber(8)
  void clearHeading() => $_clearField(8);

  @$pb.TagNumber(9)
  $fixnum.Int64 get lastRoadId => $_getI64(8);
  @$pb.TagNumber(9)
  set lastRoadId($fixnum.Int64 value) => $_setInt64(8, value);
  @$pb.TagNumber(9)
  $core.bool hasLastRoadId() => $_has(8);
  @$pb.TagNumber(9)
  void clearLastRoadId() => $_clearField(9);

  @$pb.TagNumber(10)
  $pb.PbList<GpsHistoryPoint> get history => $_getList(9);
}

class GpsHistoryPoint extends $pb.GeneratedMessage {
  factory GpsHistoryPoint({
    $core.String? timestamp,
    $core.double? lat,
    $core.double? lon,
    $core.double? accuracy,
    $core.double? speed,
    $core.double? heading,
  }) {
    final result = create();
    if (timestamp != null) result.timestamp = timestamp;
    if (lat != null) result.lat = lat;
    if (lon != null) result.lon = lon;
    if (accuracy != null) result.accuracy = accuracy;
    if (speed != null) result.speed = speed;
    if (heading != null) result.heading = heading;
    return result;
  }

  GpsHistoryPoint._();

  factory GpsHistoryPoint.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GpsHistoryPoint.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GpsHistoryPoint',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'roadmatcher.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'timestamp')
    ..aD(2, _omitFieldNames ? '' : 'lat')
    ..aD(3, _omitFieldNames ? '' : 'lon')
    ..aD(4, _omitFieldNames ? '' : 'accuracy')
    ..aD(5, _omitFieldNames ? '' : 'speed')
    ..aD(6, _omitFieldNames ? '' : 'heading')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GpsHistoryPoint clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GpsHistoryPoint copyWith(void Function(GpsHistoryPoint) updates) =>
      super.copyWith((message) => updates(message as GpsHistoryPoint))
          as GpsHistoryPoint;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GpsHistoryPoint create() => GpsHistoryPoint._();
  @$core.override
  GpsHistoryPoint createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GpsHistoryPoint getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GpsHistoryPoint>(create);
  static GpsHistoryPoint? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get timestamp => $_getSZ(0);
  @$pb.TagNumber(1)
  set timestamp($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTimestamp() => $_has(0);
  @$pb.TagNumber(1)
  void clearTimestamp() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.double get lat => $_getN(1);
  @$pb.TagNumber(2)
  set lat($core.double value) => $_setDouble(1, value);
  @$pb.TagNumber(2)
  $core.bool hasLat() => $_has(1);
  @$pb.TagNumber(2)
  void clearLat() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.double get lon => $_getN(2);
  @$pb.TagNumber(3)
  set lon($core.double value) => $_setDouble(2, value);
  @$pb.TagNumber(3)
  $core.bool hasLon() => $_has(2);
  @$pb.TagNumber(3)
  void clearLon() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.double get accuracy => $_getN(3);
  @$pb.TagNumber(4)
  set accuracy($core.double value) => $_setDouble(3, value);
  @$pb.TagNumber(4)
  $core.bool hasAccuracy() => $_has(3);
  @$pb.TagNumber(4)
  void clearAccuracy() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.double get speed => $_getN(4);
  @$pb.TagNumber(5)
  set speed($core.double value) => $_setDouble(4, value);
  @$pb.TagNumber(5)
  $core.bool hasSpeed() => $_has(4);
  @$pb.TagNumber(5)
  void clearSpeed() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.double get heading => $_getN(5);
  @$pb.TagNumber(6)
  set heading($core.double value) => $_setDouble(5, value);
  @$pb.TagNumber(6)
  $core.bool hasHeading() => $_has(5);
  @$pb.TagNumber(6)
  void clearHeading() => $_clearField(6);
}

class MatchRoadResponse extends $pb.GeneratedMessage {
  factory MatchRoadResponse({
    $fixnum.Int64? sequenceId,
    $core.bool? matched,
    $core.String? confidence,
    RoadCandidate? best,
    $core.Iterable<RoadCandidate>? candidates,
    $core.double? processingDurationMs,
  }) {
    final result = create();
    if (sequenceId != null) result.sequenceId = sequenceId;
    if (matched != null) result.matched = matched;
    if (confidence != null) result.confidence = confidence;
    if (best != null) result.best = best;
    if (candidates != null) result.candidates.addAll(candidates);
    if (processingDurationMs != null)
      result.processingDurationMs = processingDurationMs;
    return result;
  }

  MatchRoadResponse._();

  factory MatchRoadResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory MatchRoadResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'MatchRoadResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'roadmatcher.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'sequenceId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOB(2, _omitFieldNames ? '' : 'matched')
    ..aOS(3, _omitFieldNames ? '' : 'confidence')
    ..aOM<RoadCandidate>(4, _omitFieldNames ? '' : 'best',
        subBuilder: RoadCandidate.create)
    ..pPM<RoadCandidate>(5, _omitFieldNames ? '' : 'candidates',
        subBuilder: RoadCandidate.create)
    ..aD(6, _omitFieldNames ? '' : 'processingDurationMs')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  MatchRoadResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  MatchRoadResponse copyWith(void Function(MatchRoadResponse) updates) =>
      super.copyWith((message) => updates(message as MatchRoadResponse))
          as MatchRoadResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static MatchRoadResponse create() => MatchRoadResponse._();
  @$core.override
  MatchRoadResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static MatchRoadResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<MatchRoadResponse>(create);
  static MatchRoadResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get sequenceId => $_getI64(0);
  @$pb.TagNumber(1)
  set sequenceId($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSequenceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSequenceId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get matched => $_getBF(1);
  @$pb.TagNumber(2)
  set matched($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMatched() => $_has(1);
  @$pb.TagNumber(2)
  void clearMatched() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get confidence => $_getSZ(2);
  @$pb.TagNumber(3)
  set confidence($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasConfidence() => $_has(2);
  @$pb.TagNumber(3)
  void clearConfidence() => $_clearField(3);

  @$pb.TagNumber(4)
  RoadCandidate get best => $_getN(3);
  @$pb.TagNumber(4)
  set best(RoadCandidate value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasBest() => $_has(3);
  @$pb.TagNumber(4)
  void clearBest() => $_clearField(4);
  @$pb.TagNumber(4)
  RoadCandidate ensureBest() => $_ensure(3);

  @$pb.TagNumber(5)
  $pb.PbList<RoadCandidate> get candidates => $_getList(4);

  @$pb.TagNumber(6)
  $core.double get processingDurationMs => $_getN(5);
  @$pb.TagNumber(6)
  set processingDurationMs($core.double value) => $_setDouble(5, value);
  @$pb.TagNumber(6)
  $core.bool hasProcessingDurationMs() => $_has(5);
  @$pb.TagNumber(6)
  void clearProcessingDurationMs() => $_clearField(6);
}

class RoadCandidate extends $pb.GeneratedMessage {
  factory RoadCandidate({
    $fixnum.Int64? roadId,
    $fixnum.Int64? osmId,
    $core.String? name,
    $core.String? highway,
    $core.String? oneway,
    $core.double? distanceMeters,
    $core.double? headingDiff,
    $core.String? connectivity,
    $core.double? distanceScore,
    $core.double? headingScore,
    $core.double? connectivityScore,
    $core.double? score,
    $core.Iterable<LatLon>? geometry,
  }) {
    final result = create();
    if (roadId != null) result.roadId = roadId;
    if (osmId != null) result.osmId = osmId;
    if (name != null) result.name = name;
    if (highway != null) result.highway = highway;
    if (oneway != null) result.oneway = oneway;
    if (distanceMeters != null) result.distanceMeters = distanceMeters;
    if (headingDiff != null) result.headingDiff = headingDiff;
    if (connectivity != null) result.connectivity = connectivity;
    if (distanceScore != null) result.distanceScore = distanceScore;
    if (headingScore != null) result.headingScore = headingScore;
    if (connectivityScore != null) result.connectivityScore = connectivityScore;
    if (score != null) result.score = score;
    if (geometry != null) result.geometry.addAll(geometry);
    return result;
  }

  RoadCandidate._();

  factory RoadCandidate.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RoadCandidate.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RoadCandidate',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'roadmatcher.v1'),
      createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'roadId')
    ..aInt64(2, _omitFieldNames ? '' : 'osmId')
    ..aOS(3, _omitFieldNames ? '' : 'name')
    ..aOS(4, _omitFieldNames ? '' : 'highway')
    ..aOS(5, _omitFieldNames ? '' : 'oneway')
    ..aD(6, _omitFieldNames ? '' : 'distanceMeters')
    ..aD(7, _omitFieldNames ? '' : 'headingDiff')
    ..aOS(8, _omitFieldNames ? '' : 'connectivity')
    ..aD(9, _omitFieldNames ? '' : 'distanceScore')
    ..aD(10, _omitFieldNames ? '' : 'headingScore')
    ..aD(11, _omitFieldNames ? '' : 'connectivityScore')
    ..aD(12, _omitFieldNames ? '' : 'score')
    ..pPM<LatLon>(13, _omitFieldNames ? '' : 'geometry',
        subBuilder: LatLon.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RoadCandidate clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RoadCandidate copyWith(void Function(RoadCandidate) updates) =>
      super.copyWith((message) => updates(message as RoadCandidate))
          as RoadCandidate;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RoadCandidate create() => RoadCandidate._();
  @$core.override
  RoadCandidate createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RoadCandidate getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RoadCandidate>(create);
  static RoadCandidate? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get roadId => $_getI64(0);
  @$pb.TagNumber(1)
  set roadId($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRoadId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRoadId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get osmId => $_getI64(1);
  @$pb.TagNumber(2)
  set osmId($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOsmId() => $_has(1);
  @$pb.TagNumber(2)
  void clearOsmId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get name => $_getSZ(2);
  @$pb.TagNumber(3)
  set name($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasName() => $_has(2);
  @$pb.TagNumber(3)
  void clearName() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get highway => $_getSZ(3);
  @$pb.TagNumber(4)
  set highway($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasHighway() => $_has(3);
  @$pb.TagNumber(4)
  void clearHighway() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get oneway => $_getSZ(4);
  @$pb.TagNumber(5)
  set oneway($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasOneway() => $_has(4);
  @$pb.TagNumber(5)
  void clearOneway() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.double get distanceMeters => $_getN(5);
  @$pb.TagNumber(6)
  set distanceMeters($core.double value) => $_setDouble(5, value);
  @$pb.TagNumber(6)
  $core.bool hasDistanceMeters() => $_has(5);
  @$pb.TagNumber(6)
  void clearDistanceMeters() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.double get headingDiff => $_getN(6);
  @$pb.TagNumber(7)
  set headingDiff($core.double value) => $_setDouble(6, value);
  @$pb.TagNumber(7)
  $core.bool hasHeadingDiff() => $_has(6);
  @$pb.TagNumber(7)
  void clearHeadingDiff() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get connectivity => $_getSZ(7);
  @$pb.TagNumber(8)
  set connectivity($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasConnectivity() => $_has(7);
  @$pb.TagNumber(8)
  void clearConnectivity() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.double get distanceScore => $_getN(8);
  @$pb.TagNumber(9)
  set distanceScore($core.double value) => $_setDouble(8, value);
  @$pb.TagNumber(9)
  $core.bool hasDistanceScore() => $_has(8);
  @$pb.TagNumber(9)
  void clearDistanceScore() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.double get headingScore => $_getN(9);
  @$pb.TagNumber(10)
  set headingScore($core.double value) => $_setDouble(9, value);
  @$pb.TagNumber(10)
  $core.bool hasHeadingScore() => $_has(9);
  @$pb.TagNumber(10)
  void clearHeadingScore() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.double get connectivityScore => $_getN(10);
  @$pb.TagNumber(11)
  set connectivityScore($core.double value) => $_setDouble(10, value);
  @$pb.TagNumber(11)
  $core.bool hasConnectivityScore() => $_has(10);
  @$pb.TagNumber(11)
  void clearConnectivityScore() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.double get score => $_getN(11);
  @$pb.TagNumber(12)
  set score($core.double value) => $_setDouble(11, value);
  @$pb.TagNumber(12)
  $core.bool hasScore() => $_has(11);
  @$pb.TagNumber(12)
  void clearScore() => $_clearField(12);

  @$pb.TagNumber(13)
  $pb.PbList<LatLon> get geometry => $_getList(12);
}

class LatLon extends $pb.GeneratedMessage {
  factory LatLon({
    $core.double? lat,
    $core.double? lon,
  }) {
    final result = create();
    if (lat != null) result.lat = lat;
    if (lon != null) result.lon = lon;
    return result;
  }

  LatLon._();

  factory LatLon.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory LatLon.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'LatLon',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'roadmatcher.v1'),
      createEmptyInstance: create)
    ..aD(1, _omitFieldNames ? '' : 'lat')
    ..aD(2, _omitFieldNames ? '' : 'lon')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LatLon clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LatLon copyWith(void Function(LatLon) updates) =>
      super.copyWith((message) => updates(message as LatLon)) as LatLon;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static LatLon create() => LatLon._();
  @$core.override
  LatLon createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static LatLon getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<LatLon>(create);
  static LatLon? _defaultInstance;

  @$pb.TagNumber(1)
  $core.double get lat => $_getN(0);
  @$pb.TagNumber(1)
  set lat($core.double value) => $_setDouble(0, value);
  @$pb.TagNumber(1)
  $core.bool hasLat() => $_has(0);
  @$pb.TagNumber(1)
  void clearLat() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.double get lon => $_getN(1);
  @$pb.TagNumber(2)
  set lon($core.double value) => $_setDouble(1, value);
  @$pb.TagNumber(2)
  $core.bool hasLon() => $_has(1);
  @$pb.TagNumber(2)
  void clearLon() => $_clearField(2);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
