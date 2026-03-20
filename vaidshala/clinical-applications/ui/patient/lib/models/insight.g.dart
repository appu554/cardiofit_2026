// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'insight.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$InsightImpl _$$InsightImplFromJson(
  Map<String, dynamic> json,
) => _$InsightImpl(
  coachingMessage: json['coachingMessage'] as String?,
  coachingType: $enumDecodeNullable(_$InsightTypeEnumMap, json['coachingType']),
  tips:
      (json['tips'] as List<dynamic>?)?.map((e) => e as String).toList() ??
      const [],
  alerts:
      (json['alerts'] as List<dynamic>?)?.map((e) => e as String).toList() ??
      const [],
);

Map<String, dynamic> _$$InsightImplToJson(_$InsightImpl instance) =>
    <String, dynamic>{
      'coachingMessage': instance.coachingMessage,
      'coachingType': _$InsightTypeEnumMap[instance.coachingType],
      'tips': instance.tips,
      'alerts': instance.alerts,
    };

const _$InsightTypeEnumMap = {
  InsightType.reinforcement: 'reinforcement',
  InsightType.encouragement: 'encouragement',
  InsightType.problemSolving: 'problemSolving',
};
