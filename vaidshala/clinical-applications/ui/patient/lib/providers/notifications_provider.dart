// lib/providers/notifications_provider.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/app_notification.dart';
import '../services/drift_database.dart';
import 'database_provider.dart';

final notificationsProvider =
    AsyncNotifierProvider<NotificationsNotifier, List<AppNotification>>(
        NotificationsNotifier.new);

class NotificationsNotifier extends AsyncNotifier<List<AppNotification>> {
  @override
  Future<List<AppNotification>> build() async {
    try {
      final db = ref.read(databaseProvider);
      await db.seedNotifications(_seedCompanions());
      final rows = await db.allNotifications();
      return rows.map(_rowToModel).toList();
    } catch (_) {
      return _mockSeedData();
    }
  }

  Future<void> markRead(String id) async {
    try {
      final db = ref.read(databaseProvider);
      await db.markNotificationRead(id);
    } catch (_) {}
    final current = state.valueOrNull ?? [];
    state = AsyncData(
      current.map((n) => n.id == id ? n.copyWith(read: true) : n).toList(),
    );
  }

  Future<void> markAllRead() async {
    try {
      final db = ref.read(databaseProvider);
      await db.markAllNotificationsRead();
    } catch (_) {}
    final current = state.valueOrNull ?? [];
    state = AsyncData(current.map((n) => n.copyWith(read: true)).toList());
  }

  Future<void> dismiss(String id) async {
    try {
      final db = ref.read(databaseProvider);
      await db.deleteNotification(id);
    } catch (_) {}
    final current = state.valueOrNull ?? [];
    state = AsyncData(current.where((n) => n.id != id).toList());
  }

  Future<void> refresh() async {
    state = const AsyncLoading();
    state = AsyncData(await build());
  }
}

AppNotification _rowToModel(dynamic row) {
  return AppNotification(
    id: row.id as String,
    type: NotificationType.values.firstWhere(
      (t) => t.name == (row.type as String),
      orElse: () => NotificationType.coaching,
    ),
    title: row.title as String,
    body: row.body as String,
    deepLink: row.deepLink as String?,
    timestamp: DateTime.fromMillisecondsSinceEpoch(row.timestamp as int),
    read: row.read as bool,
  );
}

final unreadCountProvider = Provider<int>((ref) {
  final notifications = ref.watch(notificationsProvider).valueOrNull ?? [];
  return notifications.where((n) => !n.read).length;
});

List<AppNotification> _mockSeedData() {
  final now = DateTime.now();
  return [
    AppNotification(
      id: 'n1', type: NotificationType.coaching,
      title: 'Great progress!', body: 'Your FBG dropped 12 mg/dL this week',
      deepLink: '/home/progress',
      timestamp: DateTime(now.year, now.month, now.day, 9), read: true,
    ),
    AppNotification(
      id: 'n2', type: NotificationType.alert,
      title: 'FBG trending down', body: 'Your fasting glucose is moving toward target',
      deepLink: '/home/progress',
      timestamp: DateTime(now.year, now.month, now.day, 8), read: false,
    ),
    AppNotification(
      id: 'n3', type: NotificationType.reminder,
      title: 'Time for evening walk',
      body: 'A 15-min post-dinner walk can lower glucose by 15-20%',
      deepLink: '/home/my-day',
      timestamp: DateTime(now.year, now.month, now.day, 19), read: false,
    ),
    AppNotification(
      id: 'n4', type: NotificationType.coaching,
      title: 'Weekly progress summary', body: 'You completed 85% of actions this week',
      deepLink: '/home/progress',
      timestamp: now.subtract(const Duration(days: 1)), read: true,
    ),
    AppNotification(
      id: 'n5', type: NotificationType.milestone,
      title: 'New health tip available',
      body: "Learn about protein's role in metabolic health",
      deepLink: '/home/learn',
      timestamp: now.subtract(const Duration(days: 3)), read: true,
    ),
  ];
}

// Drift companions for seed data — only used if Drift DB is available
List<NotificationsCompanion> _seedCompanions() {
  return [];
}
