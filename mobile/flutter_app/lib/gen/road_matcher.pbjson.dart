// This is a generated file - do not edit.
//
// Generated from road_matcher.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports
// ignore_for_file: unused_import

import 'dart:convert' as $convert;
import 'dart:core' as $core;
import 'dart:typed_data' as $typed_data;

@$core.Deprecated('Use matchRoadRequestDescriptor instead')
const MatchRoadRequest$json = {
  '1': 'MatchRoadRequest',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'sequence_id', '3': 2, '4': 1, '5': 4, '10': 'sequenceId'},
    {'1': 'timestamp', '3': 3, '4': 1, '5': 9, '10': 'timestamp'},
    {'1': 'lat', '3': 4, '4': 1, '5': 1, '10': 'lat'},
    {'1': 'lon', '3': 5, '4': 1, '5': 1, '10': 'lon'},
    {'1': 'accuracy', '3': 6, '4': 1, '5': 1, '10': 'accuracy'},
    {'1': 'speed', '3': 7, '4': 1, '5': 1, '10': 'speed'},
    {'1': 'heading', '3': 8, '4': 1, '5': 1, '10': 'heading'},
    {'1': 'last_road_id', '3': 9, '4': 1, '5': 3, '10': 'lastRoadId'},
    {
      '1': 'history',
      '3': 10,
      '4': 3,
      '5': 11,
      '6': '.roadmatcher.v1.GpsHistoryPoint',
      '10': 'history'
    },
  ],
};

/// Descriptor for `MatchRoadRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List matchRoadRequestDescriptor = $convert.base64Decode(
    'ChBNYXRjaFJvYWRSZXF1ZXN0EhsKCWRldmljZV9pZBgBIAEoCVIIZGV2aWNlSWQSHwoLc2VxdW'
    'VuY2VfaWQYAiABKARSCnNlcXVlbmNlSWQSHAoJdGltZXN0YW1wGAMgASgJUgl0aW1lc3RhbXAS'
    'EAoDbGF0GAQgASgBUgNsYXQSEAoDbG9uGAUgASgBUgNsb24SGgoIYWNjdXJhY3kYBiABKAFSCG'
    'FjY3VyYWN5EhQKBXNwZWVkGAcgASgBUgVzcGVlZBIYCgdoZWFkaW5nGAggASgBUgdoZWFkaW5n'
    'EiAKDGxhc3Rfcm9hZF9pZBgJIAEoA1IKbGFzdFJvYWRJZBI5CgdoaXN0b3J5GAogAygLMh8ucm'
    '9hZG1hdGNoZXIudjEuR3BzSGlzdG9yeVBvaW50UgdoaXN0b3J5');

@$core.Deprecated('Use gpsHistoryPointDescriptor instead')
const GpsHistoryPoint$json = {
  '1': 'GpsHistoryPoint',
  '2': [
    {'1': 'timestamp', '3': 1, '4': 1, '5': 9, '10': 'timestamp'},
    {'1': 'lat', '3': 2, '4': 1, '5': 1, '10': 'lat'},
    {'1': 'lon', '3': 3, '4': 1, '5': 1, '10': 'lon'},
    {'1': 'accuracy', '3': 4, '4': 1, '5': 1, '10': 'accuracy'},
    {'1': 'speed', '3': 5, '4': 1, '5': 1, '10': 'speed'},
    {'1': 'heading', '3': 6, '4': 1, '5': 1, '10': 'heading'},
  ],
};

/// Descriptor for `GpsHistoryPoint`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List gpsHistoryPointDescriptor = $convert.base64Decode(
    'Cg9HcHNIaXN0b3J5UG9pbnQSHAoJdGltZXN0YW1wGAEgASgJUgl0aW1lc3RhbXASEAoDbGF0GA'
    'IgASgBUgNsYXQSEAoDbG9uGAMgASgBUgNsb24SGgoIYWNjdXJhY3kYBCABKAFSCGFjY3VyYWN5'
    'EhQKBXNwZWVkGAUgASgBUgVzcGVlZBIYCgdoZWFkaW5nGAYgASgBUgdoZWFkaW5n');

@$core.Deprecated('Use matchRoadResponseDescriptor instead')
const MatchRoadResponse$json = {
  '1': 'MatchRoadResponse',
  '2': [
    {'1': 'sequence_id', '3': 1, '4': 1, '5': 4, '10': 'sequenceId'},
    {'1': 'matched', '3': 2, '4': 1, '5': 8, '10': 'matched'},
    {'1': 'confidence', '3': 3, '4': 1, '5': 9, '10': 'confidence'},
    {
      '1': 'best',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.roadmatcher.v1.RoadCandidate',
      '10': 'best'
    },
    {
      '1': 'candidates',
      '3': 5,
      '4': 3,
      '5': 11,
      '6': '.roadmatcher.v1.RoadCandidate',
      '10': 'candidates'
    },
    {
      '1': 'processing_duration_ms',
      '3': 6,
      '4': 1,
      '5': 1,
      '10': 'processingDurationMs'
    },
  ],
};

/// Descriptor for `MatchRoadResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List matchRoadResponseDescriptor = $convert.base64Decode(
    'ChFNYXRjaFJvYWRSZXNwb25zZRIfCgtzZXF1ZW5jZV9pZBgBIAEoBFIKc2VxdWVuY2VJZBIYCg'
    'dtYXRjaGVkGAIgASgIUgdtYXRjaGVkEh4KCmNvbmZpZGVuY2UYAyABKAlSCmNvbmZpZGVuY2US'
    'MQoEYmVzdBgEIAEoCzIdLnJvYWRtYXRjaGVyLnYxLlJvYWRDYW5kaWRhdGVSBGJlc3QSPQoKY2'
    'FuZGlkYXRlcxgFIAMoCzIdLnJvYWRtYXRjaGVyLnYxLlJvYWRDYW5kaWRhdGVSCmNhbmRpZGF0'
    'ZXMSNAoWcHJvY2Vzc2luZ19kdXJhdGlvbl9tcxgGIAEoAVIUcHJvY2Vzc2luZ0R1cmF0aW9uTX'
    'M=');

@$core.Deprecated('Use roadCandidateDescriptor instead')
const RoadCandidate$json = {
  '1': 'RoadCandidate',
  '2': [
    {'1': 'road_id', '3': 1, '4': 1, '5': 3, '10': 'roadId'},
    {'1': 'osm_id', '3': 2, '4': 1, '5': 3, '10': 'osmId'},
    {'1': 'name', '3': 3, '4': 1, '5': 9, '10': 'name'},
    {'1': 'highway', '3': 4, '4': 1, '5': 9, '10': 'highway'},
    {'1': 'oneway', '3': 5, '4': 1, '5': 9, '10': 'oneway'},
    {'1': 'distance_meters', '3': 6, '4': 1, '5': 1, '10': 'distanceMeters'},
    {'1': 'heading_diff', '3': 7, '4': 1, '5': 1, '10': 'headingDiff'},
    {'1': 'connectivity', '3': 8, '4': 1, '5': 9, '10': 'connectivity'},
    {'1': 'distance_score', '3': 9, '4': 1, '5': 1, '10': 'distanceScore'},
    {'1': 'heading_score', '3': 10, '4': 1, '5': 1, '10': 'headingScore'},
    {
      '1': 'connectivity_score',
      '3': 11,
      '4': 1,
      '5': 1,
      '10': 'connectivityScore'
    },
    {'1': 'score', '3': 12, '4': 1, '5': 1, '10': 'score'},
    {
      '1': 'geometry',
      '3': 13,
      '4': 3,
      '5': 11,
      '6': '.roadmatcher.v1.LatLon',
      '10': 'geometry'
    },
  ],
};

/// Descriptor for `RoadCandidate`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List roadCandidateDescriptor = $convert.base64Decode(
    'Cg1Sb2FkQ2FuZGlkYXRlEhcKB3JvYWRfaWQYASABKANSBnJvYWRJZBIVCgZvc21faWQYAiABKA'
    'NSBW9zbUlkEhIKBG5hbWUYAyABKAlSBG5hbWUSGAoHaGlnaHdheRgEIAEoCVIHaGlnaHdheRIW'
    'CgZvbmV3YXkYBSABKAlSBm9uZXdheRInCg9kaXN0YW5jZV9tZXRlcnMYBiABKAFSDmRpc3Rhbm'
    'NlTWV0ZXJzEiEKDGhlYWRpbmdfZGlmZhgHIAEoAVILaGVhZGluZ0RpZmYSIgoMY29ubmVjdGl2'
    'aXR5GAggASgJUgxjb25uZWN0aXZpdHkSJQoOZGlzdGFuY2Vfc2NvcmUYCSABKAFSDWRpc3Rhbm'
    'NlU2NvcmUSIwoNaGVhZGluZ19zY29yZRgKIAEoAVIMaGVhZGluZ1Njb3JlEi0KEmNvbm5lY3Rp'
    'dml0eV9zY29yZRgLIAEoAVIRY29ubmVjdGl2aXR5U2NvcmUSFAoFc2NvcmUYDCABKAFSBXNjb3'
    'JlEjIKCGdlb21ldHJ5GA0gAygLMhYucm9hZG1hdGNoZXIudjEuTGF0TG9uUghnZW9tZXRyeQ==');

@$core.Deprecated('Use latLonDescriptor instead')
const LatLon$json = {
  '1': 'LatLon',
  '2': [
    {'1': 'lat', '3': 1, '4': 1, '5': 1, '10': 'lat'},
    {'1': 'lon', '3': 2, '4': 1, '5': 1, '10': 'lon'},
  ],
};

/// Descriptor for `LatLon`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List latLonDescriptor = $convert.base64Decode(
    'CgZMYXRMb24SEAoDbGF0GAEgASgBUgNsYXQSEAoDbG9uGAIgASgBUgNsb24=');
