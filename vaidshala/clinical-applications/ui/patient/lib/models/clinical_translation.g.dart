// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'clinical_translation.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$ClinicalTranslationImpl _$$ClinicalTranslationImplFromJson(
  Map<String, dynamic> json,
) => _$ClinicalTranslationImpl(
  clinicalTerm: json['clinicalTerm'] as String,
  patientTerm: json['patientTerm'] as String,
  explanation: json['explanation'] as String,
);

Map<String, dynamic> _$$ClinicalTranslationImplToJson(
  _$ClinicalTranslationImpl instance,
) => <String, dynamic>{
  'clinicalTerm': instance.clinicalTerm,
  'patientTerm': instance.patientTerm,
  'explanation': instance.explanation,
};
