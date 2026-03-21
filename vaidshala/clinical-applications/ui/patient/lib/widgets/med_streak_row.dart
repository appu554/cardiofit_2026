// lib/widgets/med_streak_row.dart
import 'package:flutter/material.dart';
import '../theme.dart';

class MedStreakRow extends StatelessWidget {
  final String name;
  final int streakDays;

  const MedStreakRow({
    super.key,
    required this.name,
    required this.streakDays,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      child: Row(
        children: [
          const Icon(Icons.check_circle, color: AppColors.scoreGreen, size: 20),
          const SizedBox(width: 8),
          Expanded(
            child: Text(name,
                style: const TextStyle(fontSize: 14)),
          ),
          Text(
            '$streakDays-day streak',
            style: const TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.w600,
              color: AppColors.scoreGreen,
            ),
          ),
        ],
      ),
    );
  }
}
