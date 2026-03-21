// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'symptom_entry.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$SymptomEntryImpl _$$SymptomEntryImplFromJson(Map<String, dynamic> json) =>
    _$SymptomEntryImpl(
      id: json['id'] as String,
      symptom: json['symptom'] as String,
      severity: json['severity'] as String,
      notes: json['notes'] as String?,
      timestamp: DateTime.parse(json['timestamp'] as String),
      synced: json['synced'] as bool? ?? false,
    );

Map<String, dynamic> _$$SymptomEntryImplToJson(_$SymptomEntryImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'symptom': instance.symptom,
      'severity': instance.severity,
      'notes': instance.notes,
      'timestamp': instance.timestamp.toIso8601String(),
      'synced': instance.synced,
    };
