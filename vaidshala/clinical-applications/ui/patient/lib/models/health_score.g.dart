// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'health_score.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$HealthScoreImpl _$$HealthScoreImplFromJson(Map<String, dynamic> json) =>
    _$HealthScoreImpl(
      score: (json['score'] as num).toInt(),
      label: json['label'] as String,
      delta: (json['delta'] as num?)?.toInt() ?? 0,
      sparkline:
          (json['sparkline'] as List<dynamic>?)
              ?.map((e) => (e as num).toInt())
              .toList() ??
          const [],
      updatedAt: json['updatedAt'] == null
          ? null
          : DateTime.parse(json['updatedAt'] as String),
    );

Map<String, dynamic> _$$HealthScoreImplToJson(_$HealthScoreImpl instance) =>
    <String, dynamic>{
      'score': instance.score,
      'label': instance.label,
      'delta': instance.delta,
      'sparkline': instance.sparkline,
      'updatedAt': instance.updatedAt?.toIso8601String(),
    };
