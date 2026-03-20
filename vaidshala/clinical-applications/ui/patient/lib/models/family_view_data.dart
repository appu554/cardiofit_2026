import 'package:freezed_annotation/freezed_annotation.dart';

part 'family_view_data.freezed.dart';
part 'family_view_data.g.dart';

@freezed
class FamilyViewData with _$FamilyViewData {
  const factory FamilyViewData({
    required String patientName,
    @Default([]) List<FamilyAction> mealActions,
    @Default([]) List<FamilyAction> activityActions,
    String? supportMessage,
  }) = _FamilyViewData;

  factory FamilyViewData.fromJson(Map<String, dynamic> json) =>
      _$FamilyViewDataFromJson(json);
}

@freezed
class FamilyAction with _$FamilyAction {
  const factory FamilyAction({
    required String text,
    required String icon,
    required String time,
  }) = _FamilyAction;

  factory FamilyAction.fromJson(Map<String, dynamic> json) =>
      _$FamilyActionFromJson(json);
}
