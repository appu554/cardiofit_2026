// lib/widgets/adherence_ring.dart
import 'package:flutter/material.dart';
import '../theme.dart';

/// Small circular progress indicator showing an adherence percentage.
class AdherenceRing extends StatelessWidget {
  final int percentage;
  final double size;

  const AdherenceRing({
    super.key,
    required this.percentage,
    this.size = 48,
  });

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: size,
      height: size,
      child: Stack(
        alignment: Alignment.center,
        children: [
          CircularProgressIndicator(
            value: percentage / 100,
            strokeWidth: 4,
            backgroundColor: Colors.grey.shade200,
            valueColor: const AlwaysStoppedAnimation(AppColors.scoreGreen),
          ),
          Text(
            '$percentage%',
            style: const TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.bold,
            ),
          ),
        ],
      ),
    );
  }
}
