// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'vital_entry.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$VitalEntryImpl _$$VitalEntryImplFromJson(Map<String, dynamic> json) =>
    _$VitalEntryImpl(
      id: json['id'] as String,
      type: json['type'] as String,
      value: json['value'] as String,
      unit: json['unit'] as String,
      timestamp: DateTime.parse(json['timestamp'] as String),
      synced: json['synced'] as bool? ?? false,
    );

Map<String, dynamic> _$$VitalEntryImplToJson(_$VitalEntryImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'type': instance.type,
      'value': instance.value,
      'unit': instance.unit,
      'timestamp': instance.timestamp.toIso8601String(),
      'synced': instance.synced,
    };
