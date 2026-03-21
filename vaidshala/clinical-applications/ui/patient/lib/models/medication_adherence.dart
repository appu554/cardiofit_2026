import 'package:freezed_annotation/freezed_annotation.dart';

part 'medication_adherence.freezed.dart';
part 'medication_adherence.g.dart';

@freezed
class MedicationAdherence with _$MedicationAdherence {
  const factory MedicationAdherence({
    required int weeklyPct,
    required List<MedStreak> streaks,
    MissedDose? lastMissed,
  }) = _MedicationAdherence;

  factory MedicationAdherence.fromJson(Map<String, dynamic> json) =>
      _$MedicationAdherenceFromJson(json);
}

@freezed
class MedStreak with _$MedStreak {
  const factory MedStreak({
    required String medicationName,
    required int streakDays,
  }) = _MedStreak;

  factory MedStreak.fromJson(Map<String, dynamic> json) =>
      _$MedStreakFromJson(json);
}

@freezed
class MissedDose with _$MissedDose {
  const factory MissedDose({
    required String medicationName,
    required int daysAgo,
  }) = _MissedDose;

  factory MissedDose.fromJson(Map<String, dynamic> json) =>
      _$MissedDoseFromJson(json);
}
