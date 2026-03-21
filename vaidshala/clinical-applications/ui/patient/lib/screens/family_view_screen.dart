import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/family_view_provider.dart';
import '../theme.dart';
import '../utils/icon_mapper.dart';
import '../widgets/skeleton_card.dart';

class FamilyViewScreen extends ConsumerWidget {
  final String token;
  const FamilyViewScreen({super.key, required this.token});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final dataAsync = ref.watch(familyViewProvider(token));

    return Scaffold(
      body: SafeArea(
        child: dataAsync.when(
          data: (data) {
            if (data == null) {
              return const Center(
                child: Text('This link has expired or is invalid.'),
              );
            }
            return RefreshIndicator(
              onRefresh: () async => ref.invalidate(familyViewProvider(token)),
              child: SingleChildScrollView(
                physics: const AlwaysScrollableScrollPhysics(),
                padding: const EdgeInsets.all(24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    // Header
                    Text(
                      "${data.patientName}'s Health Plan Today",
                      style: const TextStyle(
                          fontSize: 22, fontWeight: FontWeight.bold),
                    ),
                    const SizedBox(height: 4),
                    const Text(
                      'Here\'s how you can help today',
                      style: TextStyle(
                          fontSize: 14, color: AppColors.textSecondary),
                    ),
                    const SizedBox(height: 20),

                    // Meal Guidance
                    if (data.mealActions.isNotEmpty) ...[
                      const _SectionLabel(icon: Icons.restaurant, text: 'Meal Guidance'),
                      const SizedBox(height: 8),
                      Card(
                        child: Column(
                          children: data.mealActions
                              .map((a) => _FamilyActionTile(
                                    text: a.text,
                                    icon: a.icon,
                                    time: a.time,
                                  ))
                              .toList(),
                        ),
                      ),
                    ],

                    const SizedBox(height: 16),

                    // Activity Reminders
                    if (data.activityActions.isNotEmpty) ...[
                      const _SectionLabel(
                          icon: Icons.directions_walk,
                          text: 'Activity Reminders'),
                      const SizedBox(height: 8),
                      Card(
                        child: Column(
                          children: data.activityActions
                              .map((a) => _FamilyActionTile(
                                    text: a.text,
                                    icon: a.icon,
                                    time: a.time,
                                  ))
                              .toList(),
                        ),
                      ),
                    ],

                    // Support Message
                    if (data.supportMessage != null) ...[
                      const SizedBox(height: 20),
                      Card(
                        color: AppColors.coachingGreen,
                        child: Padding(
                          padding: const EdgeInsets.all(16),
                          child: Row(
                            children: [
                              const Icon(Icons.favorite,
                                  color: AppColors.scoreGreen),
                              const SizedBox(width: 12),
                              Expanded(
                                child: Text(
                                  data.supportMessage!,
                                  style: const TextStyle(
                                      fontSize: 14, height: 1.4),
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                    ],

                    // Branding footer
                    const SizedBox(height: 32),
                    const Center(
                      child: Column(
                        children: [
                          Icon(Icons.health_and_safety,
                              color: AppColors.primaryTeal, size: 32),
                          SizedBox(height: 4),
                          Text(
                            'Powered by Vaidshala',
                            style: TextStyle(
                              fontSize: 12,
                              color: AppColors.textSecondary,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            );
          },
          loading: () => const SingleChildScrollView(
            child: Column(
              children: [
                SkeletonCard(height: 200),
                SkeletonCard(height: 150),
              ],
            ),
          ),
          error: (_, __) => const Center(
            child: Text('This link has expired. Ask Rajesh to share a new one.'),
          ),
        ),
      ),
    );
  }
}

class _SectionLabel extends StatelessWidget {
  final IconData icon;
  final String text;
  const _SectionLabel({required this.icon, required this.text});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Icon(icon, size: 20, color: AppColors.primaryTeal),
        const SizedBox(width: 8),
        Text(
          text,
          style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
        ),
      ],
    );
  }
}

class _FamilyActionTile extends StatelessWidget {
  final String text;
  final String icon;
  final String time;
  const _FamilyActionTile(
      {required this.text, required this.icon, required this.time});

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: Icon(IconMapper.fromString(icon), color: AppColors.primaryTeal),
      title: Text(text, style: const TextStyle(fontSize: 14)),
      trailing: Text(
        time,
        style: const TextStyle(fontSize: 12, color: AppColors.textSecondary),
      ),
    );
  }
}
