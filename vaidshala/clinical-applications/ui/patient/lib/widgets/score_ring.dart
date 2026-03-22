import 'dart:math';
import 'package:flutter/material.dart';
import '../theme.dart';
import 'animations/animations.dart';

class ScoreRing extends StatelessWidget {
  final int? score;
  final double size;

  const ScoreRing({super.key, this.score, this.size = 120});

  @override
  Widget build(BuildContext context) {
    final displayScore = score ?? 0;
    final color = score != null ? AppColors.scoreColor(displayScore) : Colors.grey;
    final fraction = displayScore / 100;

    return SizedBox(
      width: size,
      height: size,
      child: Stack(
        alignment: Alignment.center,
        children: [
          CustomPaint(
            size: Size(size, size),
            painter: _RingPainter(
              fraction: fraction,
              color: color,
              strokeWidth: size * 0.08,
            ),
          ),
          if (score != null)
            CountUpText(
              value: displayScore.toDouble(),
              style: TextStyle(
                fontSize: size * 0.28,
                fontWeight: FontWeight.bold,
                color: Colors.white,
              ),
            )
          else
            Text(
              '--',
              style: TextStyle(
                fontSize: size * 0.28,
                fontWeight: FontWeight.bold,
                color: Colors.white54,
              ),
            ),
        ],
      ),
    );
  }
}

class _RingPainter extends CustomPainter {
  final double fraction;
  final Color color;
  final double strokeWidth;

  _RingPainter({
    required this.fraction,
    required this.color,
    required this.strokeWidth,
  });

  @override
  void paint(Canvas canvas, Size size) {
    final center = Offset(size.width / 2, size.height / 2);
    final radius = (size.width - strokeWidth) / 2;

    // Background ring
    canvas.drawCircle(
      center,
      radius,
      Paint()
        ..style = PaintingStyle.stroke
        ..strokeWidth = strokeWidth
        ..color = Colors.white.withValues(alpha: 0.15),
    );

    // Progress arc
    if (fraction > 0) {
      canvas.drawArc(
        Rect.fromCircle(center: center, radius: radius),
        -pi / 2,
        2 * pi * fraction,
        false,
        Paint()
          ..style = PaintingStyle.stroke
          ..strokeWidth = strokeWidth
          ..strokeCap = StrokeCap.round
          ..color = color,
      );
    }
  }

  @override
  bool shouldRepaint(covariant _RingPainter oldDelegate) =>
      fraction != oldDelegate.fraction || color != oldDelegate.color;
}
