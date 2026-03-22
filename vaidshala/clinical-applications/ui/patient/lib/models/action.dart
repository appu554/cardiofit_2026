import 'package:freezed_annotation/freezed_annotation.dart';

part 'action.freezed.dart';
part 'action.g.dart';

@freezed
class PatientAction with _$PatientAction {
  const factory PatientAction({
    required String id,
    required String text,
    required String icon,
    required String time,
    @Default(false) bool done,
    String? why,
  }) = _PatientAction;

  factory PatientAction.fromJson(Map<String, dynamic> json) =>
      _$PatientActionFromJson(json);
}
