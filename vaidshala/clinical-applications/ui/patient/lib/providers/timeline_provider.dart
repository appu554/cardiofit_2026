import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/timeline_entry.dart';
import 'actions_provider.dart';
import 'insights_provider.dart';

class MyDayState {
  final List<TimelineEntry> entries;
  final String? tipOfTheDay;
  final bool isLoading;

  const MyDayState({
    this.entries = const [],
    this.tipOfTheDay,
    this.isLoading = false,
  });
}

final myDayProvider = Provider<MyDayState>((ref) {
  final actionsState = ref.watch(actionsProvider);
  final insightsAsync = ref.watch(insightsProvider);

  if (actionsState.isLoading) {
    return const MyDayState(isLoading: true);
  }

  // Convert actions to timeline entries, sorted by time
  final entries = actionsState.actions.map((a) => TimelineEntry(
        id: a.id,
        time: a.time,
        text: a.text,
        icon: a.icon,
        done: a.done,
      )).toList()
    ..sort((a, b) => a.time.compareTo(b.time));

  // Rotate tip based on day of year
  String? tip;
  insightsAsync.whenData((insight) {
    if (insight.tips.isNotEmpty) {
      final dayOfYear = DateTime.now().difference(DateTime(DateTime.now().year)).inDays;
      tip = insight.tips[dayOfYear % insight.tips.length];
    }
  });

  return MyDayState(entries: entries, tipOfTheDay: tip);
});
