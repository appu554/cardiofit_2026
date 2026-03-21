// lib/widgets/medication_adherence_section.dart
import 'package:flutter/material.dart';
import '../models/medication_adherence.dart';
import '../theme.dart';
import 'adherence_ring.dart';
import 'med_streak_row.dart';

class MedicationAdherenceSection extends StatelessWidget {
  final MedicationAdherence adherence;

  const MedicationAdherenceSection({super.key, required this.adherence});

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Weekly adherence header
          Padding(
            padding: const EdgeInsets.all(16),
            child: Row(
              children: [
                AdherenceRing(percentage: adherence.weeklyPct),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text(
                        'This Week',
                        style: TextStyle(
                          fontSize: 14,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      Text(
                        '${(adherence.weeklyPct * 7 / 100).round()} of 7 days — all meds taken',
                        style: const TextStyle(
                          fontSize: 12,
                          color: AppColors.textSecondary,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),

          const Divider(height: 1),

          // Per-medication streaks
          const SizedBox(height: 8),
          ...adherence.streaks.map(
            (s) => MedStreakRow(
              name: s.medicationName,
              streakDays: s.streakDays,
            ),
          ),

          // Missed dose indicator
          if (adherence.lastMissed != null) ...[
            const Divider(height: 24),
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
              child: Row(
                children: [
                  const Icon(Icons.warning_amber,
                      color: Colors.orange, size: 18),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      'Last missed: ${adherence.lastMissed!.medicationName}, '
                      '${adherence.lastMissed!.daysAgo} day${adherence.lastMissed!.daysAgo == 1 ? "" : "s"} ago',
                      style: const TextStyle(
                        fontSize: 12,
                        color: Colors.orange,
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ] else
            const SizedBox(height: 12),
        ],
      ),
    );
  }
}
