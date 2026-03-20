// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'milestone.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$MilestoneImpl _$$MilestoneImplFromJson(Map<String, dynamic> json) =>
    _$MilestoneImpl(
      id: json['id'] as String,
      title: json['title'] as String,
      description: json['description'] as String,
      status: $enumDecode(_$MilestoneStatusEnumMap, json['status']),
      achievedDate: json['achievedDate'] as String?,
    );

Map<String, dynamic> _$$MilestoneImplToJson(_$MilestoneImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'title': instance.title,
      'description': instance.description,
      'status': _$MilestoneStatusEnumMap[instance.status]!,
      'achievedDate': instance.achievedDate,
    };

const _$MilestoneStatusEnumMap = {
  MilestoneStatus.achieved: 'achieved',
  MilestoneStatus.locked: 'locked',
};
