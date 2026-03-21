// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'domain_score.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$DomainScoreImpl _$$DomainScoreImplFromJson(Map<String, dynamic> json) =>
    _$DomainScoreImpl(
      name: json['name'] as String,
      score: (json['score'] as num).toInt(),
      target: (json['target'] as num).toInt(),
      icon: json['icon'] as String,
    );

Map<String, dynamic> _$$DomainScoreImplToJson(_$DomainScoreImpl instance) =>
    <String, dynamic>{
      'name': instance.name,
      'score': instance.score,
      'target': instance.target,
      'icon': instance.icon,
    };
