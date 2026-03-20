// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'timeline_entry.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$TimelineEntryImpl _$$TimelineEntryImplFromJson(Map<String, dynamic> json) =>
    _$TimelineEntryImpl(
      id: json['id'] as String,
      time: json['time'] as String,
      text: json['text'] as String,
      icon: json['icon'] as String,
      done: json['done'] as bool,
    );

Map<String, dynamic> _$$TimelineEntryImplToJson(_$TimelineEntryImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'time': instance.time,
      'text': instance.text,
      'icon': instance.icon,
      'done': instance.done,
    };
