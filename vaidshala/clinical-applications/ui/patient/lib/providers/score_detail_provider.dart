// lib/providers/score_detail_provider.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/domain_score.dart';
import '../models/health_score.dart';
import 'health_score_provider.dart';

class ScoreDetailState {
  final int? score;
  final String label;
  final List<double> scoreHistory;
  final List<DomainScore> domains;
  final String explanation;

  const ScoreDetailState({
    this.score,
    this.label = '',
    this.scoreHistory = const [],
    this.domains = const [],
    this.explanation = '',
  });
}

final scoreDetailProvider = Provider<ScoreDetailState>((ref) {
  final healthScore = ref.watch(healthScoreProvider).valueOrNull;

  if (healthScore == null) return const ScoreDetailState();

  // Extend sparkline to 12 weeks with mock historical data
  final sparkline = healthScore.sparkline;
  final history = <double>[
    // Pad to 12 weeks if shorter
    ...List.generate(
      (12 - sparkline.length).clamp(0, 12),
      (i) => (30 - i).toDouble(),
    ),
    ...sparkline.map((s) => s.toDouble()),
  ];

  return ScoreDetailState(
    score: healthScore.score,
    label: _scoreLabel(healthScore.score),
    scoreHistory: history.length > 12 ? history.sublist(history.length - 12) : history,
    domains: const [
      DomainScore(name: 'Blood Sugar', score: 35, target: 60, icon: 'bloodtype'),
      DomainScore(name: 'Activity', score: 22, target: 50, icon: 'directions_walk'),
      DomainScore(name: 'Body Health', score: 58, target: 70, icon: 'monitor_weight'),
      DomainScore(name: 'Heart Health', score: 72, target: 80, icon: 'favorite'),
    ],
    explanation:
        "Your metabolic health score reflects how well your blood sugar, activity "
        "levels, body composition, and heart health markers are tracking against "
        "clinical targets. Focus on the areas with the biggest gaps to see the "
        "most improvement.",
  );
});

String _scoreLabel(int score) {
  if (score >= 80) return 'Excellent';
  if (score >= 60) return 'Good';
  if (score >= 40) return 'Improving';
  return 'Needs Attention';
}
