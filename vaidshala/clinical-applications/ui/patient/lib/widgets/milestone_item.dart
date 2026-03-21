import 'package:flutter/material.dart';
import '../models/milestone.dart';
import '../theme.dart';

class MilestoneItem extends StatelessWidget {
  final Milestone milestone;
  const MilestoneItem({super.key, required this.milestone});

  @override
  Widget build(BuildContext context) {
    final achieved = milestone.status == MilestoneStatus.achieved;

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
      child: Row(
        children: [
          Container(
            width: 40,
            height: 40,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: achieved ? const Color(0xFFFFC107) : Colors.grey.shade300,
            ),
            child: Icon(
              achieved ? Icons.star : Icons.lock,
              color: achieved ? Colors.white : Colors.grey.shade500,
              size: 20,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  milestone.title,
                  style: TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w600,
                    color: achieved
                        ? AppColors.textPrimary
                        : AppColors.textSecondary,
                  ),
                ),
                Text(
                  achieved
                      ? 'Achieved ${milestone.achievedDate ?? ''}'
                      : milestone.description,
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
    );
  }
}
