import 'package:flutter/material.dart';
import '../models/cause_effect.dart';
import '../theme.dart';
import '../utils/icon_mapper.dart';

class CauseEffectCard extends StatelessWidget {
  final CauseEffect causeEffect;
  const CauseEffectCard({super.key, required this.causeEffect});

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      child: Container(
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(12),
          gradient: const LinearGradient(
            colors: [AppColors.coachingGreen, Colors.white],
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
          ),
        ),
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Icon(IconMapper.fromString(causeEffect.causeIcon),
                          size: 18, color: AppColors.scoreGreen),
                      const SizedBox(width: 6),
                      Expanded(
                        child: Text(
                          causeEffect.cause,
                          style: const TextStyle(
                            fontSize: 13,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ),
                    ],
                  ),
                  const Padding(
                    padding: EdgeInsets.only(left: 9),
                    child: Icon(Icons.arrow_downward,
                        size: 16, color: AppColors.textSecondary),
                  ),
                  Row(
                    children: [
                      Icon(IconMapper.fromString(causeEffect.effectIcon),
                          size: 18, color: AppColors.primaryTeal),
                      const SizedBox(width: 6),
                      Expanded(
                        child: Text(
                          causeEffect.effect,
                          style: const TextStyle(fontSize: 13),
                        ),
                      ),
                    ],
                  ),
                ],
              ),
            ),
            if (causeEffect.verified)
              const Padding(
                padding: EdgeInsets.only(left: 8),
                child: Icon(Icons.verified, color: AppColors.scoreGreen, size: 24),
              ),
          ],
        ),
      ),
    );
  }
}
