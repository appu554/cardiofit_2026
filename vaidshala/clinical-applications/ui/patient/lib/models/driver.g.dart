// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'driver.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$HealthDriverImpl _$$HealthDriverImplFromJson(Map<String, dynamic> json) =>
    _$HealthDriverImpl(
      id: json['id'] as String,
      name: json['name'] as String,
      icon: json['icon'] as String,
      current: (json['current'] as num).toDouble(),
      target: (json['target'] as num).toDouble(),
      unit: json['unit'] as String,
      improving: json['improving'] as bool? ?? false,
    );

Map<String, dynamic> _$$HealthDriverImplToJson(_$HealthDriverImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'name': instance.name,
      'icon': instance.icon,
      'current': instance.current,
      'target': instance.target,
      'unit': instance.unit,
      'improving': instance.improving,
    };
