// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'action.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$PatientActionImpl _$$PatientActionImplFromJson(Map<String, dynamic> json) =>
    _$PatientActionImpl(
      id: json['id'] as String,
      text: json['text'] as String,
      icon: json['icon'] as String,
      time: json['time'] as String,
      done: json['done'] as bool? ?? false,
      why: json['why'] as String?,
    );

Map<String, dynamic> _$$PatientActionImplToJson(_$PatientActionImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'text': instance.text,
      'icon': instance.icon,
      'time': instance.time,
      'done': instance.done,
      'why': instance.why,
    };
