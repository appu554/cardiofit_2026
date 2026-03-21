// lib/screens/home_tab.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../models/insight.dart';
import '../providers/actions_provider.dart';
import '../providers/drivers_provider.dart';
import '../providers/health_score_provider.dart';
import '../providers/insights_provider.dart';
import '../theme.dart';
import '../theme/motion.dart';
import '../widgets/action_checklist_item.dart';
import '../widgets/animations/animations.dart';
import '../widgets/coaching_card.dart';
import '../widgets/driver_card.dart';
import '../widgets/score_ring.dart';
import '../widgets/skeleton_card.dart';

class HomeTab extends ConsumerWidget {
  const HomeTab({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final scoreAsync = ref.watch(healthScoreProvider);
    final actionsState = ref.watch(actionsProvider);
    final driversAsync = ref.watch(healthDriversProvider);
    final insightsAsync = ref.watch(insightsProvider);

    return SafeArea(
      child: RefreshIndicator(
        onRefresh: () async {
          ref.invalidate(healthScoreProvider);
          ref.read(actionsProvider.notifier).refresh();
          ref.invalidate(healthDriversProvider);
          ref.invalidate(insightsProvider);
        },
        child: SingleChildScrollView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.only(bottom: 80),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Greeting
              StaggeredItem(
                index: 0,
                keepAlive: true,
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
                  child: Text(
                    'Namaste, Rajesh',
                    style: Theme.of(context).textTheme.headlineLarge,
                  ),
                ),
              ),

              // Score Ring Card
              StaggeredItem(
                index: 1,
                keepAlive: true,
                child: scoreAsync.when(
                  data: (score) => _ScoreCard(score: score?.score),
                  loading: () => const SkeletonCard(height: 180),
                  error: (_, __) => const _ScoreCard(score: null),
                ),
              ),

              // Coaching Message — with animated left border (0→3px width)
              StaggeredItem(
                index: 2,
                keepAlive: true,
                child: insightsAsync.when(
                  data: (insight) {
                    if (insight.coachingMessage == null) {
                      return const SizedBox.shrink();
                    }
                    return Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 4),
                      child: TweenAnimationBuilder<double>(
                        tween: Tween(begin: 0.0, end: 3.0),
                        duration: AppMotion.kEntranceDuration,
                        builder: (context, borderWidth, child) => Container(
                          decoration: BoxDecoration(
                            border: Border(
                              left: BorderSide(
                                color: AppColors.primaryTeal,
                                width: borderWidth,
                              ),
                            ),
                          ),
                          child: child,
                        ),
                        child: CoachingMessageCard(
                          message: insight.coachingMessage!,
                          type: insight.coachingType ?? InsightType.encouragement,
                        ),
                      ),
                    );
                  },
                  loading: () => const SkeletonCard(height: 80),
                  error: (_, __) => const SizedBox.shrink(),
                ),
              ),

              // Today's Actions
              StaggeredItem(
                index: 3,
                keepAlive: true,
                child: _ActionsSection(
                  state: actionsState,
                  onToggle: (id) =>
                      ref.read(actionsProvider.notifier).toggleAction(id),
                ),
              ),

              // Health Drivers
              StaggeredItem(
                index: 4,
                keepAlive: true,
                child: driversAsync.when(
                  data: (drivers) => Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Padding(
                        padding: EdgeInsets.fromLTRB(16, 16, 16, 8),
                        child: Text(
                          'Health Drivers',
                          style: TextStyle(
                            fontSize: 18,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                      ),
                      ...drivers.map((d) => HealthDriverCard(driver: d)),
                    ],
                  ),
                  loading: () => const SkeletonCard(height: 100),
                  error: (_, __) => const SizedBox.shrink(),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ScoreCard extends StatelessWidget {
  final int? score;
  const _ScoreCard({this.score});

  @override
  Widget build(BuildContext context) {
    return SpringTapCard(
      onTap: () => context.push('/score-detail'),
      child: Card(
        margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        color: AppColors.primaryNavy,
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Row(
            children: [
              Hero(
                tag: 'score-ring',
                child: ScoreRing(score: score, size: 120),
              ),
              const SizedBox(width: 24),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text(
                      'Metabolic Health Score',
                      style: TextStyle(
                        color: Colors.white70,
                        fontSize: 12,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                    const SizedBox(height: 4),
                    if (score != null)
                      Text(
                        'This month',
                        style: TextStyle(
                          color: Colors.white.withValues(alpha: 0.5),
                          fontSize: 11,
                        ),
                      ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ActionsSection extends StatelessWidget {
  final ActionsState state;
  final ValueChanged<String> onToggle;

  const _ActionsSection({required this.state, required this.onToggle});

  @override
  Widget build(BuildContext context) {
    if (state.isLoading) {
      return const SkeletonCard(height: 200);
    }

    if (state.actions.isEmpty) {
      return const Card(
        margin: EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        child: Padding(
          padding: EdgeInsets.all(24),
          child: Center(
            child: Text(
              'Connect to load your health actions',
              style: TextStyle(color: AppColors.textSecondary),
            ),
          ),
        ),
      );
    }

    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                const Text(
                  "Today's Actions",
                  style: TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.bold,
                  ),
                ),
                Text(
                  '${state.completionPct}% complete',
                  style: const TextStyle(
                    fontSize: 12,
                    color: AppColors.textSecondary,
                  ),
                ),
              ],
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: AnimatedProgressBar(
              value: state.completionPct / 100,
              color: AppColors.scoreGreen,
              height: 6,
            ),
          ),
          const SizedBox(height: 8),
          ...state.actions.map(
            (action) => ActionChecklistItem(
              action: action,
              onToggle: onToggle,
            ),
          ),
          const SizedBox(height: 8),
        ],
      ),
    );
  }
}
