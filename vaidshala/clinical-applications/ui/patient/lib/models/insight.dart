import 'package:freezed_annotation/freezed_annotation.dart';

part 'insight.freezed.dart';
part 'insight.g.dart';

enum InsightType {
  reinforcement,
  encouragement,
  problemSolving,
}

@freezed
class Insight with _$Insight {
  const factory Insight({
    String? coachingMessage,
    InsightType? coachingType,
    @Default([]) List<String> tips,
    @Default([]) List<String> alerts,
  }) = _Insight;

  factory Insight.fromJson(Map<String, dynamic> json) =>
      _$InsightFromJson(json);
}
