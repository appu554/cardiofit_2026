import 'package:flutter/material.dart';
import '../models/driver.dart';
import '../theme.dart';
import 'animations/animations.dart';

class HealthDriverCard extends StatelessWidget {
  final HealthDriver driver;

  const HealthDriverCard({super.key, required this.driver});

  IconData _iconFromName(String name) {
    return switch (name) {
      'bloodtype' => Icons.bloodtype,
      'favorite' => Icons.favorite,
      'monitor_heart' => Icons.monitor_heart,
      'directions_walk' => Icons.directions_walk,
      _ => Icons.health_and_safety,
    };
  }

  @override
  Widget build(BuildContext context) {
    final progress = (driver.current / driver.target).clamp(0.0, 1.0);
    final trendColor = driver.improving ? AppColors.scoreGreen : AppColors.scoreRed;

    return SpringTapCard(
      onTap: () {},
      child: Card(
        margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              Icon(_iconFromName(driver.icon), color: AppColors.primaryTeal, size: 24),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      driver.name,
                      style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 14),
                    ),
                    const SizedBox(height: 4),
                    AnimatedProgressBar(
                      value: progress,
                      color: trendColor,
                      height: 6,
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 12),
              Column(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  Text(
                    '${driver.current.round()}',
                    style: TextStyle(
                      fontWeight: FontWeight.bold,
                      fontSize: 16,
                      color: trendColor,
                    ),
                  ),
                  Text(
                    driver.unit,
                    style: const TextStyle(fontSize: 11, color: AppColors.textSecondary),
                  ),
                ],
              ),
              const SizedBox(width: 4),
              Icon(
                driver.improving ? Icons.trending_up : Icons.trending_down,
                color: trendColor,
                size: 18,
              ),
            ],
          ),
        ),
      ),
    );
  }
}
