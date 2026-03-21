import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/actions_provider.dart';
import '../providers/timeline_provider.dart';
import '../theme.dart';
import '../widgets/did_you_know_card.dart';
import '../widgets/skeleton_card.dart';
import '../widgets/timeline_entry_widget.dart';

class MyDayTab extends ConsumerWidget {
  const MyDayTab({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final myDay = ref.watch(myDayProvider);

    return SafeArea(
      child: RefreshIndicator(
        onRefresh: () => ref.read(actionsProvider.notifier).refresh(),
        child: myDay.isLoading
            ? const SingleChildScrollView(
                child: Column(
                  children: [SkeletonCard(height: 300)],
                ),
              )
            : SingleChildScrollView(
                physics: const AlwaysScrollableScrollPhysics(),
                padding: const EdgeInsets.only(bottom: 80),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    // Header
                    const Padding(
                      padding: EdgeInsets.fromLTRB(16, 16, 16, 4),
                      child: Text(
                        'My Day',
                        style: TextStyle(
                            fontSize: 24, fontWeight: FontWeight.bold),
                      ),
                    ),
                    const Padding(
                      padding: EdgeInsets.symmetric(horizontal: 16),
                      child: Text(
                        'Your daily health routine',
                        style: TextStyle(
                            fontSize: 14, color: AppColors.textSecondary),
                      ),
                    ),
                    const SizedBox(height: 16),

                    // Timeline
                    if (myDay.entries.isEmpty)
                      const Padding(
                        padding: EdgeInsets.all(32),
                        child: Center(
                          child: Text(
                            'No activities scheduled for today',
                            style: TextStyle(color: AppColors.textSecondary),
                          ),
                        ),
                      )
                    else
                      Card(
                        margin: const EdgeInsets.symmetric(horizontal: 16),
                        child: Padding(
                          padding: const EdgeInsets.symmetric(vertical: 16),
                          child: Column(
                            children: [
                              for (var i = 0; i < myDay.entries.length; i++)
                                TimelineEntryWidget(
                                  entry: myDay.entries[i],
                                  isLast: i == myDay.entries.length - 1,
                                ),
                            ],
                          ),
                        ),
                      ),

                    // Completion footer
                    if (myDay.entries.isNotEmpty &&
                        myDay.entries.every((e) => e.done))
                      const Padding(
                        padding: EdgeInsets.all(24),
                        child: Center(
                          child: Column(
                            children: [
                              Icon(Icons.celebration,
                                  size: 48, color: AppColors.scoreGreen),
                              SizedBox(height: 8),
                              Text(
                                'All done for today! Great work!',
                                style: TextStyle(
                                  fontSize: 16,
                                  fontWeight: FontWeight.w600,
                                  color: AppColors.scoreGreen,
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),

                    // Did You Know
                    if (myDay.tipOfTheDay != null)
                      DidYouKnowCard(tip: myDay.tipOfTheDay!),
                  ],
                ),
              ),
      ),
    );
  }
}
