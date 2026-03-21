// lib/widgets/full_sparkline_chart.dart
import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/material.dart';
import '../theme.dart';

class FullSparklineChart extends StatelessWidget {
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
  Widget build(BuildContext context) {
    if (data.isEmpty) return SizedBox(height: height);

    final spots = data
        .asMap()
        .entries
        .map((e) => FlSpot(e.key.toDouble(), e.value))
        .toList();

    return SizedBox(
      height: height,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16),
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
            extraLinesData: targetLine != null
                ? ExtraLinesData(horizontalLines: [
                    HorizontalLine(
                      y: targetLine!,
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
    );
  }
}
