import 'package:flutter/material.dart';
import '../models/progress_metric.dart';
import '../theme.dart';
import '../utils/icon_mapper.dart';

class ProgressMetricRow extends StatelessWidget {
  final ProgressMetric metric;
  const ProgressMetricRow({super.key, required this.metric});

  @override
  Widget build(BuildContext context) {
    final delta = metric.current - metric.previous;
    final deltaStr = delta >= 0 ? '+${delta.toStringAsFixed(1)}' : delta.toStringAsFixed(1);
    final isPositiveDelta = metric.improving;
    final progress = (metric.current / metric.target).clamp(0.0, 2.0);
    final barValue = progress.clamp(0.0, 1.0);

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(IconMapper.fromString(metric.icon), size: 20, color: AppColors.textSecondary),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  metric.name,
                  style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600),
                ),
              ),
              if (isPositiveDelta)
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                  decoration: BoxDecoration(
                    color: AppColors.coachingGreen,
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: const Text(
                    'Improving',
                    style: TextStyle(fontSize: 10, color: AppColors.scoreGreen),
                  ),
                ),
            ],
          ),
          const SizedBox(height: 6),
          ClipRRect(
            borderRadius: BorderRadius.circular(4),
            child: LinearProgressIndicator(
              value: barValue,
              minHeight: 8,
              backgroundColor: Colors.grey.shade200,
              valueColor: AlwaysStoppedAnimation(
                isPositiveDelta ? AppColors.scoreGreen : AppColors.scoreYellow,
              ),
            ),
          ),
          const SizedBox(height: 4),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(
                '${metric.current.toStringAsFixed(metric.current == metric.current.roundToDouble() ? 0 : 1)} ${metric.unit}',
                style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w500),
              ),
              Text(
                '$deltaStr from previous',
                style: TextStyle(
                  fontSize: 11,
                  color: isPositiveDelta ? AppColors.scoreGreen : AppColors.scoreRed,
                ),
              ),
              Text(
                'Target: ${metric.target.toStringAsFixed(metric.target == metric.target.roundToDouble() ? 0 : 1)}',
                style: const TextStyle(fontSize: 11, color: AppColors.textSecondary),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
