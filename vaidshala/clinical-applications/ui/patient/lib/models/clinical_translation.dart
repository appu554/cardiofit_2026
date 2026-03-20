import 'package:freezed_annotation/freezed_annotation.dart';

part 'clinical_translation.freezed.dart';
part 'clinical_translation.g.dart';

@freezed
class ClinicalTranslation with _$ClinicalTranslation {
  const factory ClinicalTranslation({
    required String clinicalTerm,
    required String patientTerm,
    required String explanation,
  }) = _ClinicalTranslation;

  factory ClinicalTranslation.fromJson(Map<String, dynamic> json) =>
      _$ClinicalTranslationFromJson(json);
}
