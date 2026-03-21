// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'medication_adherence.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$MedicationAdherenceImpl _$$MedicationAdherenceImplFromJson(
  Map<String, dynamic> json,
) => _$MedicationAdherenceImpl(
  weeklyPct: (json['weeklyPct'] as num).toInt(),
  streaks: (json['streaks'] as List<dynamic>)
      .map((e) => MedStreak.fromJson(e as Map<String, dynamic>))
      .toList(),
  lastMissed: json['lastMissed'] == null
      ? null
      : MissedDose.fromJson(json['lastMissed'] as Map<String, dynamic>),
);

Map<String, dynamic> _$$MedicationAdherenceImplToJson(
  _$MedicationAdherenceImpl instance,
) => <String, dynamic>{
  'weeklyPct': instance.weeklyPct,
  'streaks': instance.streaks,
  'lastMissed': instance.lastMissed,
};

_$MedStreakImpl _$$MedStreakImplFromJson(Map<String, dynamic> json) =>
    _$MedStreakImpl(
      medicationName: json['medicationName'] as String,
      streakDays: (json['streakDays'] as num).toInt(),
    );

Map<String, dynamic> _$$MedStreakImplToJson(_$MedStreakImpl instance) =>
    <String, dynamic>{
      'medicationName': instance.medicationName,
      'streakDays': instance.streakDays,
    };

_$MissedDoseImpl _$$MissedDoseImplFromJson(Map<String, dynamic> json) =>
    _$MissedDoseImpl(
      medicationName: json['medicationName'] as String,
      daysAgo: (json['daysAgo'] as num).toInt(),
    );

Map<String, dynamic> _$$MissedDoseImplToJson(_$MissedDoseImpl instance) =>
    <String, dynamic>{
      'medicationName': instance.medicationName,
      'daysAgo': instance.daysAgo,
    };
