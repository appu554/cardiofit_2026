// lib/widgets/full_sparkline_chart.dart
import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/material.dart';
import '../theme.dart';
import '../theme/motion.dart';

class FullSparklineChart extends StatefulWidget {
  final List<double> data;
  final double? targetLine;
  final double height;

  const FullSparklineChart({
    super.key,
    required this.data,
    this.targetLine,
    this.height = 120,
  });

  @override
  State<FullSparklineChart> createState() => _FullSparklineChartState();
}

class _FullSparklineChartState extends State<FullSparklineChart>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late final Animation<double> _animation;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 800),
    );
    _animation = CurvedAnimation(
      parent: _controller,
      curve: AppMotion.kDecelerate,
    );
    _controller.forward();
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (widget.data.isEmpty) return SizedBox(height: widget.height);

    final spots = widget.data
        .asMap()
        .entries
        .map((e) => FlSpot(e.key.toDouble(), e.value))
        .toList();

    return SizedBox(
      height: widget.height,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16),
        child: AnimatedBuilder(
          animation: _animation,
          builder: (context, child) => ClipRect(
            clipper: _RevealClipper(_animation.value),
            child: child,
          ),
          child: LineChart(
            LineChartData(
              minY: 0,
              maxY: 100,
              gridData: const FlGridData(show: false),
              borderData: FlBorderData(show: false),
              titlesData: FlTitlesData(
                leftTitles: const AxisTitles(
                  sideTitles: SideTitles(showTitles: false),
                ),
                rightTitles: const AxisTitles(
                  sideTitles: SideTitles(showTitles: false),
                ),
                topTitles: const AxisTitles(
                  sideTitles: SideTitles(showTitles: false),
                ),
                bottomTitles: AxisTitles(
                  sideTitles: SideTitles(
                    showTitles: true,
                    interval: 1,
                    getTitlesWidget: (value, meta) {
                      if (value.toInt() % 4 == 0) {
                        return Text(
                          'W${value.toInt() + 1}',
                          style: const TextStyle(
                            fontSize: 10,
                            color: AppColors.textSecondary,
                          ),
                        );
                      }
                      return const SizedBox.shrink();
                    },
                  ),
                ),
              ),
              extraLinesData: widget.targetLine != null
                  ? ExtraLinesData(horizontalLines: [
                      HorizontalLine(
                        y: widget.targetLine!,
                        color: AppColors.scoreGreen.withValues(alpha: 0.5),
                        strokeWidth: 1,
                        dashArray: [8, 4],
                        label: HorizontalLineLabel(
                          show: true,
                          labelResolver: (_) => 'Target',
                          style: const TextStyle(
                            fontSize: 10,
                            color: AppColors.scoreGreen,
                          ),
                        ),
                      ),
                    ])
                  : null,
              lineBarsData: [
                LineChartBarData(
                  spots: spots,
                  isCurved: true,
                  color: AppColors.primaryTeal,
                  barWidth: 2.5,
                  dotData: FlDotData(
                    show: true,
                    getDotPainter: (spot, _, __, ___) {
                      if (spot == spots.last) {
                        return FlDotCirclePainter(
                          radius: 4,
                          color: AppColors.primaryTeal,
                          strokeWidth: 2,
                          strokeColor: Colors.white,
                        );
                      }
                      return FlDotCirclePainter(radius: 0, color: Colors.transparent);
                    },
                  ),
                  belowBarData: BarAreaData(
                    show: true,
                    gradient: LinearGradient(
                      begin: Alignment.topCenter,
                      end: Alignment.bottomCenter,
                      colors: [
                        AppColors.primaryTeal.withValues(alpha: 0.3),
                        AppColors.primaryTeal.withValues(alpha: 0.05),
                      ],
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _RevealClipper extends CustomClipper<Rect> {
  final double fraction;
  _RevealClipper(this.fraction);

  @override
  Rect getClip(Size size) => Rect.fromLTWH(0, 0, size.width * fraction, size.height);

  @override
  bool shouldReclip(covariant _RevealClipper oldClipper) => fraction != oldClipper.fraction;
}
