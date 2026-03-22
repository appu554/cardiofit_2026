import 'package:freezed_annotation/freezed_annotation.dart';

part 'health_score.freezed.dart';
part 'health_score.g.dart';

@freezed
class HealthScore with _$HealthScore {
  const factory HealthScore({
    required int score,
    required String label,
    required int delta,
    required List<int> sparkline,
    DateTime? updatedAt,
  }) = _HealthScore;

  factory HealthScore.fromJson(Map<String, dynamic> json) =>
      _$HealthScoreFromJson(json);
}
