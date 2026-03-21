import 'package:freezed_annotation/freezed_annotation.dart';

part 'domain_score.freezed.dart';
part 'domain_score.g.dart';

@freezed
class DomainScore with _$DomainScore {
  const factory DomainScore({
    required String name,
    required int score,
    required int target,
    required String icon,
  }) = _DomainScore;

  factory DomainScore.fromJson(Map<String, dynamic> json) =>
      _$DomainScoreFromJson(json);
}
