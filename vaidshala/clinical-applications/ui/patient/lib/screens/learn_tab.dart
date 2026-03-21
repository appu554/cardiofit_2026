import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/learn_provider.dart';
import '../theme.dart';
import '../widgets/alert_card.dart';
import '../widgets/clinical_translation_row.dart';
import '../widgets/education_tip_card.dart';
import '../widgets/animations/animations.dart';

class LearnTab extends ConsumerWidget {
  const LearnTab({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final learn = ref.watch(learnProvider);

    return SafeArea(
      child: SingleChildScrollView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.only(bottom: 80),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Header
            StaggeredItem(
              index: 0,
              keepAlive: true,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: const [
                  Padding(
                    padding: EdgeInsets.fromLTRB(16, 16, 16, 4),
                    child: Text(
                      'Learn',
                      style: TextStyle(fontSize: 24, fontWeight: FontWeight.bold),
                    ),
                  ),
                  Padding(
                    padding: EdgeInsets.symmetric(horizontal: 16),
                    child: Text(
                      'Understand your health better',
                      style:
                          TextStyle(fontSize: 14, color: AppColors.textSecondary),
                    ),
                  ),
                ],
              ),
            ),

            // Alerts (conditional)
            if (learn.alerts.isNotEmpty)
              StaggeredItem(
                index: 1,
                keepAlive: true,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const SizedBox(height: 12),
                    ...learn.alerts.map((a) => AlertCard(message: a)),
                  ],
                ),
              ),

            // Health Tips
            if (learn.tips.isNotEmpty)
              StaggeredItem(
                index: 2,
                keepAlive: true,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Padding(
                      padding: EdgeInsets.fromLTRB(16, 20, 16, 8),
                      child: Text(
                        'Health Tips',
                        style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                      ),
                    ),
                    ...learn.tips.map((t) => EducationTipCard(tip: t)),
                  ],
                ),
              ),

            // Understanding Your Reports
            if (learn.translations.isNotEmpty)
              StaggeredItem(
                index: 3,
                keepAlive: true,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Padding(
                      padding: EdgeInsets.fromLTRB(16, 20, 16, 8),
                      child: Text(
                        'Understanding Your Reports',
                        style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                      ),
                    ),
                    const Padding(
                      padding: EdgeInsets.symmetric(horizontal: 16),
                      child: Text(
                        'Tap any term to learn more',
                        style: TextStyle(
                            fontSize: 13, color: AppColors.textSecondary),
                      ),
                    ),
                    const SizedBox(height: 4),
                    Card(
                      margin: const EdgeInsets.symmetric(horizontal: 16),
                      child: Column(
                        children: learn.translations
                            .map((t) => ClinicalTranslationRow(translation: t))
                            .toList(),
                      ),
                    ),
                  ],
                ),
              ),
          ],
        ),
      ),
    );
  }
}
