// lib/models/progress_metric.dart
import 'package:freezed_annotation/freezed_annotation.dart';

part 'progress_metric.freezed.dart';
part 'progress_metric.g.dart';

@freezed
class ProgressMetric with _$ProgressMetric {
  const factory ProgressMetric({
    required String id,
    required String name,
    required String icon,
    required double current,
    required double previous,
    required double target,
    required String unit,
    @Default(false) bool improving,
    @Default([]) List<double> sparkline,
  }) = _ProgressMetric;

  factory ProgressMetric.fromJson(Map<String, dynamic> json) =>
      _$ProgressMetricFromJson(json);
}
