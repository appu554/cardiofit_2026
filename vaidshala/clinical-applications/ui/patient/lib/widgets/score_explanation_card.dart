// lib/widgets/score_explanation_card.dart
import 'package:flutter/material.dart';
import '../theme.dart';

class ScoreExplanationCard extends StatelessWidget {
  final int score;
  final String explanation;

  const ScoreExplanationCard({
    super.key,
    required this.score,
    required this.explanation,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.all(16),
      color: AppColors.coachingGreen,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.lightbulb_outline,
                    color: AppColors.scoreGreen, size: 20),
                const SizedBox(width: 8),
                const Text(
                  'What This Means',
                  style: TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              explanation,
              style: const TextStyle(
                fontSize: 13,
                color: AppColors.textPrimary,
                height: 1.5,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
