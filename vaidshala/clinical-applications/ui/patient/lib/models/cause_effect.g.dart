// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'cause_effect.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$CauseEffectImpl _$$CauseEffectImplFromJson(Map<String, dynamic> json) =>
    _$CauseEffectImpl(
      id: json['id'] as String,
      cause: json['cause'] as String,
      effect: json['effect'] as String,
      causeIcon: json['causeIcon'] as String,
      effectIcon: json['effectIcon'] as String,
      verified: json['verified'] as bool? ?? false,
    );

Map<String, dynamic> _$$CauseEffectImplToJson(_$CauseEffectImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'cause': instance.cause,
      'effect': instance.effect,
      'causeIcon': instance.causeIcon,
      'effectIcon': instance.effectIcon,
      'verified': instance.verified,
    };
