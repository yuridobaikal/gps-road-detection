// This is a generated file - do not edit.
//
// Generated from road_matcher.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:grpc/service_api.dart' as $grpc;
import 'package:protobuf/protobuf.dart' as $pb;

import 'road_matcher.pb.dart' as $0;

export 'road_matcher.pb.dart';

@$pb.GrpcServiceName('roadmatcher.v1.RoadMatcher')
class RoadMatcherClient extends $grpc.Client {
  /// The hostname for this service.
  static const $core.String defaultHost = '';

  /// OAuth scopes needed for the client.
  static const $core.List<$core.String> oauthScopes = [
    '',
  ];

  RoadMatcherClient(super.channel, {super.options, super.interceptors});

  $grpc.ResponseStream<$0.MatchRoadResponse> streamGps(
    $async.Stream<$0.MatchRoadRequest> request, {
    $grpc.CallOptions? options,
  }) {
    return $createStreamingCall(_$streamGps, request, options: options);
  }

  $grpc.ResponseFuture<$0.MatchRoadResponse> matchRoad(
    $0.MatchRoadRequest request, {
    $grpc.CallOptions? options,
  }) {
    return $createUnaryCall(_$matchRoad, request, options: options);
  }

  // method descriptors

  static final _$streamGps =
      $grpc.ClientMethod<$0.MatchRoadRequest, $0.MatchRoadResponse>(
          '/roadmatcher.v1.RoadMatcher/StreamGps',
          ($0.MatchRoadRequest value) => value.writeToBuffer(),
          $0.MatchRoadResponse.fromBuffer);
  static final _$matchRoad =
      $grpc.ClientMethod<$0.MatchRoadRequest, $0.MatchRoadResponse>(
          '/roadmatcher.v1.RoadMatcher/MatchRoad',
          ($0.MatchRoadRequest value) => value.writeToBuffer(),
          $0.MatchRoadResponse.fromBuffer);
}

@$pb.GrpcServiceName('roadmatcher.v1.RoadMatcher')
abstract class RoadMatcherServiceBase extends $grpc.Service {
  $core.String get $name => 'roadmatcher.v1.RoadMatcher';

  RoadMatcherServiceBase() {
    $addMethod($grpc.ServiceMethod<$0.MatchRoadRequest, $0.MatchRoadResponse>(
        'StreamGps',
        streamGps,
        true,
        true,
        ($core.List<$core.int> value) => $0.MatchRoadRequest.fromBuffer(value),
        ($0.MatchRoadResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.MatchRoadRequest, $0.MatchRoadResponse>(
        'MatchRoad',
        matchRoad_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.MatchRoadRequest.fromBuffer(value),
        ($0.MatchRoadResponse value) => value.writeToBuffer()));
  }

  $async.Stream<$0.MatchRoadResponse> streamGps(
      $grpc.ServiceCall call, $async.Stream<$0.MatchRoadRequest> request);

  $async.Future<$0.MatchRoadResponse> matchRoad_Pre($grpc.ServiceCall $call,
      $async.Future<$0.MatchRoadRequest> $request) async {
    return matchRoad($call, await $request);
  }

  $async.Future<$0.MatchRoadResponse> matchRoad(
      $grpc.ServiceCall call, $0.MatchRoadRequest request);
}
