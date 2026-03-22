import 'package:flutter/material.dart';
import '../models/action.dart';
import '../theme.dart';
import 'animations/animations.dart';

class ActionChecklistItem extends StatelessWidget {
  final PatientAction action;
  final ValueChanged<String> onToggle;

  const ActionChecklistItem({
    super.key,
    required this.action,
    required this.onToggle,
  });

  IconData _iconFromName(String name) {
    return switch (name) {
      'medication' => Icons.medication,
      'bloodtype' => Icons.bloodtype,
      'directions_walk' => Icons.directions_walk,
      'favorite' => Icons.favorite,
      _ => Icons.check_circle_outline,
    };
  }

  @override
  Widget build(BuildContext context) {
    return SpringTapCard(
      onTap: () => onToggle(action.id),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
        child: Row(
          children: [
            AnimatedSwitcher(
              duration: const Duration(milliseconds: 250),
              child: action.done
                  ? const Icon(Icons.check_circle, color: AppColors.scoreGreen,
                      size: 22, key: ValueKey('done'))
                  : Icon(Icons.radio_button_unchecked,
                      color: AppColors.textSecondary, size: 22,
                      key: const ValueKey('undone')),
            ),
            const SizedBox(width: 12),
            Icon(_iconFromName(action.icon), size: 18, color: AppColors.primaryTeal),
            const SizedBox(width: 8),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    action.text,
                    style: TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w500,
                      decoration: action.done ? TextDecoration.lineThrough : null,
                      color: action.done ? AppColors.textSecondary : AppColors.textPrimary,
                    ),
                  ),
                  if (action.why != null)
                    Text(
                      action.why!,
                      style: const TextStyle(fontSize: 11, color: AppColors.textSecondary),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                ],
              ),
            ),
            Text(
              action.time,
              style: const TextStyle(fontSize: 11, color: AppColors.textSecondary),
            ),
          ],
        ),
      ),
    );
  }
}
