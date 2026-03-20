// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'progress_metric.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$ProgressMetricImpl _$$ProgressMetricImplFromJson(Map<String, dynamic> json) =>
    _$ProgressMetricImpl(
      id: json['id'] as String,
      name: json['name'] as String,
      icon: json['icon'] as String,
      current: (json['current'] as num).toDouble(),
      previous: (json['previous'] as num).toDouble(),
      target: (json['target'] as num).toDouble(),
      unit: json['unit'] as String,
      improving: json['improving'] as bool? ?? false,
      sparkline:
          (json['sparkline'] as List<dynamic>?)
              ?.map((e) => (e as num).toDouble())
              .toList() ??
          const [],
    );

Map<String, dynamic> _$$ProgressMetricImplToJson(
  _$ProgressMetricImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'name': instance.name,
  'icon': instance.icon,
  'current': instance.current,
  'previous': instance.previous,
  'target': instance.target,
  'unit': instance.unit,
  'improving': instance.improving,
  'sparkline': instance.sparkline,
};
