import 'package:flutter/material.dart';
import '../models/insight.dart';
import '../theme.dart';

class CoachingMessageCard extends StatelessWidget {
  final String message;
  final InsightType type;

  const CoachingMessageCard({
    super.key,
    required this.message,
    required this.type,
  });

  @override
  Widget build(BuildContext context) {
    final (icon, bgColor) = switch (type) {
      InsightType.encouragement => (Icons.emoji_events, AppColors.coachingGreen),
      InsightType.reinforcement => (Icons.thumb_up, AppColors.coachingGreen),
      InsightType.problemSolving => (Icons.lightbulb, AppColors.alertAmber),
    };

    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      color: bgColor,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(icon, size: 20, color: AppColors.primaryTeal),
            const SizedBox(width: 12),
            Expanded(
              child: Text(
                message,
                style: const TextStyle(fontSize: 14, height: 1.5),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
