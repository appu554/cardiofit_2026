// lib/providers/medication_adherence_provider.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/medication_adherence.dart';

final medicationAdherenceProvider =
    FutureProvider<MedicationAdherence>((ref) async {
  // Sprint 3: Mock data — Rajesh Kumar 14-day medication history
  // In future sprints, this reads from Drift medication_log table
  return const MedicationAdherence(
    weeklyPct: 85,
    streaks: [
      MedStreak(medicationName: 'Metformin 1000mg BD', streakDays: 12),
      MedStreak(medicationName: 'Glimepiride 2mg OD', streakDays: 8),
      MedStreak(medicationName: 'Telmisartan 40mg OD', streakDays: 14),
    ],
    lastMissed: MissedDose(medicationName: 'Metformin PM', daysAgo: 2),
  );
});
