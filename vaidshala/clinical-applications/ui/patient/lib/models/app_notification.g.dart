// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'app_notification.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$AppNotificationImpl _$$AppNotificationImplFromJson(
  Map<String, dynamic> json,
) => _$AppNotificationImpl(
  id: json['id'] as String,
  type: $enumDecode(_$NotificationTypeEnumMap, json['type']),
  title: json['title'] as String,
  body: json['body'] as String,
  deepLink: json['deepLink'] as String?,
  timestamp: DateTime.parse(json['timestamp'] as String),
  read: json['read'] as bool? ?? false,
);

Map<String, dynamic> _$$AppNotificationImplToJson(
  _$AppNotificationImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'type': _$NotificationTypeEnumMap[instance.type]!,
  'title': instance.title,
  'body': instance.body,
  'deepLink': instance.deepLink,
  'timestamp': instance.timestamp.toIso8601String(),
  'read': instance.read,
};

const _$NotificationTypeEnumMap = {
  NotificationType.coaching: 'coaching',
  NotificationType.reminder: 'reminder',
  NotificationType.alert: 'alert',
  NotificationType.milestone: 'milestone',
};
