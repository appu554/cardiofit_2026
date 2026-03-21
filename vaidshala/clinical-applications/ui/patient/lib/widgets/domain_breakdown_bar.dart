// lib/widgets/domain_breakdown_bar.dart
import 'package:flutter/material.dart';
import '../theme.dart';
import 'animations/animations.dart';

class DomainBreakdownBar extends StatelessWidget {
  final String label;
  final int score;
  final int target;
  final IconData icon;
  final Color color;

  const DomainBreakdownBar({
    super.key,
    required this.label,
    required this.score,
    required this.target,
    required this.icon,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(icon, size: 18, color: color),
              const SizedBox(width: 8),
              Text(label,
                  style: const TextStyle(
                      fontSize: 14, fontWeight: FontWeight.w500)),
              const Spacer(),
              Text('$score',
                  style: TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.bold,
                      color: color)),
              Text(' / $target',
                  style: const TextStyle(
                      fontSize: 12, color: AppColors.textSecondary)),
            ],
          ),
          const SizedBox(height: 4),
          Stack(
            children: [
              // Background
              Container(
                height: 8,
                decoration: BoxDecoration(
                  color: Colors.grey.shade200,
                  borderRadius: BorderRadius.circular(4),
                ),
              ),
              // Target marker
              FractionallySizedBox(
                widthFactor: (target / 100).clamp(0, 1),
                child: Container(
                  height: 8,
                  alignment: Alignment.centerRight,
                  child: Container(
                    width: 2,
                    height: 8,
                    color: AppColors.textSecondary,
                  ),
                ),
              ),
              // Score bar (animated)
              AnimatedProgressBar(
                value: score / 100,
                color: color,
                height: 8,
                borderRadius: 4,
              ),
            ],
          ),
        ],
      ),
    );
  }
}
