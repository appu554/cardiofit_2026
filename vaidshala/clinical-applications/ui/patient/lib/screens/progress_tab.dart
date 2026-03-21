// lib/screens/progress_tab.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/medication_adherence_provider.dart';
import '../providers/progress_provider.dart';
import '../theme.dart';
import '../widgets/animations/animations.dart';
import '../widgets/cause_effect_card.dart';
import '../widgets/medication_adherence_section.dart';
import '../widgets/milestone_item.dart';
import '../widgets/progress_metric_row.dart';
import '../widgets/skeleton_card.dart';

class ProgressTab extends ConsumerWidget {
  const ProgressTab({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final progressAsync = ref.watch(progressProvider);

    return SafeArea(
      child: RefreshIndicator(
        onRefresh: () => ref.read(progressProvider.notifier).refresh(),
        child: progressAsync.when(
          data: (state) => _ProgressContent(state: state),
          loading: () => const SingleChildScrollView(
            child: Column(
              children: [
                SkeletonCard(height: 200),
                SkeletonCard(height: 150),
                SkeletonCard(height: 120),
              ],
            ),
          ),
          error: (_, __) => const Center(
            child: Text('Unable to load progress data'),
          ),
        ),
      ),
    );
  }
}

class _ProgressContent extends ConsumerWidget {
  final ProgressState state;
  const _ProgressContent({required this.state});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return SingleChildScrollView(
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.only(bottom: 80),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Header — index 0
          StaggeredItem(
            index: 0,
            keepAlive: true,
            child: const Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Padding(
                  padding: EdgeInsets.fromLTRB(16, 16, 16, 4),
                  child: Text(
                    'Your Progress',
                    style: TextStyle(fontSize: 24, fontWeight: FontWeight.bold),
                  ),
                ),
                Padding(
                  padding: EdgeInsets.symmetric(horizontal: 16),
                  child: Text(
                    'Track how your health is changing over time',
                    style: TextStyle(fontSize: 14, color: AppColors.textSecondary),
                  ),
                ),
              ],
            ),
          ),

          // Key Metrics Card — index 1
          if (state.metrics.isNotEmpty)
            StaggeredItem(
              index: 1,
              keepAlive: true,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Padding(
                    padding: EdgeInsets.fromLTRB(16, 20, 16, 8),
                    child: Text(
                      'Key Metrics',
                      style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                    ),
                  ),
                  Card(
                    margin: const EdgeInsets.symmetric(horizontal: 16),
                    child: Column(
                      children: state.metrics
                          .map((m) => ProgressMetricRow(metric: m))
                          .toList(),
                    ),
                  ),
                ],
              ),
            ),

          // Cause & Effect Section — index 2
          if (state.causeEffects.isNotEmpty)
            StaggeredItem(
              index: 2,
              keepAlive: true,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Padding(
                    padding: EdgeInsets.fromLTRB(16, 20, 16, 8),
                    child: Text(
                      'How Your Actions Help',
                      style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                    ),
                  ),
                  ...state.causeEffects
                      .map((ce) => CauseEffectCard(causeEffect: ce)),
                ],
              ),
            ),

          // Milestones Section — index 3
          if (state.milestones.isNotEmpty)
            StaggeredItem(
              index: 3,
              keepAlive: true,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Padding(
                    padding: EdgeInsets.fromLTRB(16, 20, 16, 8),
                    child: Text(
                      'Milestones',
                      style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                    ),
                  ),
                  ...state.milestones.map((m) => MilestoneItem(milestone: m)),
                ],
              ),
            ),

          // Medication Adherence Section — index 4
          StaggeredItem(
            index: 4,
            keepAlive: true,
            child: ref.watch(medicationAdherenceProvider).when(
              data: (adherence) => Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Padding(
                    padding: EdgeInsets.fromLTRB(16, 20, 16, 8),
                    child: Text(
                      'Medication Adherence',
                      style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                    ),
                  ),
                  MedicationAdherenceSection(adherence: adherence),
                ],
              ),
              loading: () => const SizedBox.shrink(),
              error: (_, __) => const SizedBox.shrink(),
            ),
          ),
        ],
      ),
    );
  }
}
