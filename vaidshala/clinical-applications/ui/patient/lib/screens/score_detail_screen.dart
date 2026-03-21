// lib/screens/score_detail_screen.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/score_detail_provider.dart';
import '../theme.dart';
import '../utils/icon_mapper.dart';
import '../widgets/domain_breakdown_bar.dart';
import '../widgets/full_sparkline_chart.dart';
import '../widgets/score_explanation_card.dart';
import '../widgets/score_ring.dart';
import '../widgets/animations/animations.dart';

IconData mapIcon(String name) => IconMapper.fromString(name);

class ScoreDetailScreen extends ConsumerWidget {
  const ScoreDetailScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final detail = ref.watch(scoreDetailProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('Your Health Score')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.only(bottom: 32),
        child: Column(
          children: [
            // Hero ScoreRing (large)
            StaggeredItem(
              index: 0,
              keepAlive: true,
              child: Padding(
                padding: const EdgeInsets.symmetric(vertical: 24),
                child: Center(
                  child: Hero(
                    tag: 'score-ring',
                    child: ScoreRing(score: detail.score, size: 180),
                  ),
                ),
              ),
            ),

            // 12-week sparkline
            if (detail.scoreHistory.isNotEmpty)
              StaggeredItem(
                index: 1,
                keepAlive: true,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Padding(
                      padding: EdgeInsets.fromLTRB(16, 8, 16, 4),
                      child: Text(
                        '12-Week Trend',
                        style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
                      ),
                    ),
                    FullSparklineChart(
                      data: detail.scoreHistory,
                      targetLine: 60,
                      height: 120,
                    ),
                    const SizedBox(height: 16),
                  ],
                ),
              ),

            // Domain breakdown
            if (detail.domains.isNotEmpty)
              StaggeredItem(
                index: 2,
                keepAlive: true,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Padding(
                      padding: EdgeInsets.fromLTRB(16, 16, 16, 8),
                      child: Text(
                        'Score Breakdown',
                        style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
                      ),
                    ),
                    ...detail.domains.map(
                      (d) => DomainBreakdownBar(
                        label: d.name,
                        score: d.score,
                        target: d.target,
                        icon: mapIcon(d.icon),
                        color: AppColors.scoreColor(d.score),
                      ),
                    ),
                  ],
                ),
              ),

            // Explanation card
            if (detail.explanation.isNotEmpty)
              StaggeredItem(
                index: 3,
                keepAlive: true,
                child: ScoreExplanationCard(
                  score: detail.score ?? 0,
                  explanation: detail.explanation,
                ),
              ),
          ],
        ),
      ),
    );
  }
}
