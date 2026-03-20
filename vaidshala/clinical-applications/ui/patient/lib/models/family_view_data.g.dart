// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'family_view_data.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$FamilyViewDataImpl _$$FamilyViewDataImplFromJson(Map<String, dynamic> json) =>
    _$FamilyViewDataImpl(
      patientName: json['patientName'] as String,
      mealActions:
          (json['mealActions'] as List<dynamic>?)
              ?.map((e) => FamilyAction.fromJson(e as Map<String, dynamic>))
              .toList() ??
          const [],
      activityActions:
          (json['activityActions'] as List<dynamic>?)
              ?.map((e) => FamilyAction.fromJson(e as Map<String, dynamic>))
              .toList() ??
          const [],
      supportMessage: json['supportMessage'] as String?,
    );

Map<String, dynamic> _$$FamilyViewDataImplToJson(
  _$FamilyViewDataImpl instance,
) => <String, dynamic>{
  'patientName': instance.patientName,
  'mealActions': instance.mealActions,
  'activityActions': instance.activityActions,
  'supportMessage': instance.supportMessage,
};

_$FamilyActionImpl _$$FamilyActionImplFromJson(Map<String, dynamic> json) =>
    _$FamilyActionImpl(
      text: json['text'] as String,
      icon: json['icon'] as String,
      time: json['time'] as String,
    );

Map<String, dynamic> _$$FamilyActionImplToJson(_$FamilyActionImpl instance) =>
    <String, dynamic>{
      'text': instance.text,
      'icon': instance.icon,
      'time': instance.time,
    };
