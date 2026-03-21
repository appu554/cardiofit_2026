import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/actions_provider.dart';
import '../providers/timeline_provider.dart';
import '../theme.dart';
import '../widgets/did_you_know_card.dart';
import '../widgets/skeleton_card.dart';
import '../widgets/timeline_entry_widget.dart';
import '../widgets/vitals_entry_sheet.dart';
import '../widgets/symptom_entry_sheet.dart';

class MyDayTab extends ConsumerWidget {
  const MyDayTab({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final myDay = ref.watch(myDayProvider);

    return Scaffold(
      backgroundColor: Colors.transparent,
      floatingActionButton: _SpeedDialFab(),
      body: SafeArea(
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
                              style:
                                  TextStyle(color: AppColors.textSecondary),
                            ),
                          ),
                        )
                      else
                        Card(
                          margin:
                              const EdgeInsets.symmetric(horizontal: 16),
                          child: Padding(
                            padding:
                                const EdgeInsets.symmetric(vertical: 16),
                            child: Column(
                              children: [
                                for (var i = 0;
                                    i < myDay.entries.length;
                                    i++)
                                  TimelineEntryWidget(
                                    entry: myDay.entries[i],
                                    isLast:
                                        i == myDay.entries.length - 1,
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
                                    size: 48,
                                    color: AppColors.scoreGreen),
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
      ),
    );
  }
}

class _SpeedDialFab extends StatefulWidget {
  @override
  State<_SpeedDialFab> createState() => _SpeedDialFabState();
}

class _SpeedDialFabState extends State<_SpeedDialFab>
    with SingleTickerProviderStateMixin {
  bool _isOpen = false;
  late final AnimationController _controller;
  late final Animation<double> _expandAnimation;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 250),
    );
    _expandAnimation = CurvedAnimation(
      parent: _controller,
      curve: Curves.easeOut,
    );
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _toggle() {
    setState(() => _isOpen = !_isOpen);
    if (_isOpen) {
      _controller.forward();
    } else {
      _controller.reverse();
    }
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.end,
      children: [
        // Sub-buttons
        ScaleTransition(
          scale: _expandAnimation,
          child: Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                  decoration: BoxDecoration(
                    color: Colors.black87,
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: const Text('Log Reading',
                      style: TextStyle(color: Colors.white, fontSize: 12)),
                ),
                const SizedBox(width: 8),
                FloatingActionButton.small(
                  heroTag: 'fab_vitals',
                  onPressed: () {
                    _toggle();
                    showModalBottomSheet(
                      context: context,
                      isScrollControlled: true,
                      backgroundColor: Colors.transparent,
                      builder: (_) => const VitalsEntrySheet(),
                    );
                  },
                  child: const Icon(Icons.monitor_heart, size: 20),
                ),
              ],
            ),
          ),
        ),
        ScaleTransition(
          scale: _expandAnimation,
          child: Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                  decoration: BoxDecoration(
                    color: Colors.black87,
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: const Text('Log Symptom',
                      style: TextStyle(color: Colors.white, fontSize: 12)),
                ),
                const SizedBox(width: 8),
                FloatingActionButton.small(
                  heroTag: 'fab_symptom',
                  onPressed: () {
                    _toggle();
                    showModalBottomSheet(
                      context: context,
                      isScrollControlled: true,
                      backgroundColor: Colors.transparent,
                      builder: (_) => const SymptomEntrySheet(),
                    );
                  },
                  child: const Icon(Icons.edit_note, size: 20),
                ),
              ],
            ),
          ),
        ),

        // Main FAB
        FloatingActionButton(
          heroTag: 'fab_main',
          onPressed: _toggle,
          child: AnimatedRotation(
            turns: _isOpen ? 0.125 : 0,
            duration: const Duration(milliseconds: 250),
            child: const Icon(Icons.add),
          ),
        ),
      ],
    );
  }
}
