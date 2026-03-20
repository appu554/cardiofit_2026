import 'package:freezed_annotation/freezed_annotation.dart';

part 'milestone.freezed.dart';
part 'milestone.g.dart';

enum MilestoneStatus { achieved, locked }

@freezed
class Milestone with _$Milestone {
  const factory Milestone({
    required String id,
    required String title,
    required String description,
    required MilestoneStatus status,
    String? achievedDate,
  }) = _Milestone;

  factory Milestone.fromJson(Map<String, dynamic> json) =>
      _$MilestoneFromJson(json);
}
