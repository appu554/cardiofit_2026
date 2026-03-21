// lib/screens/notifications_screen.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../models/app_notification.dart';
import '../providers/notifications_provider.dart';
import '../theme.dart';
import '../widgets/notification_date_group.dart';
import '../widgets/animations/animations.dart';

class NotificationsScreen extends ConsumerWidget {
  const NotificationsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final notificationsAsync = ref.watch(notificationsProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Notifications'),
        actions: [
          TextButton(
            onPressed: () =>
                ref.read(notificationsProvider.notifier).markAllRead(),
            child: const Text('Mark all read'),
          ),
        ],
      ),
      body: notificationsAsync.when(
        data: (notifications) {
          if (notifications.isEmpty) {
            return const Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(Icons.notifications_none,
                      size: 64, color: AppColors.textSecondary),
                  SizedBox(height: 16),
                  Text(
                    'No notifications yet',
                    style: TextStyle(
                      fontSize: 16,
                      color: AppColors.textSecondary,
                    ),
                  ),
                ],
              ),
            );
          }

          final grouped = _groupByDate(notifications);
          return ListView(
            children: grouped.entries
                .toList()
                .asMap()
                .entries
                .map((mapEntry) => StaggeredItem(
                      index: mapEntry.key,
                      child: NotificationDateGroup(
                        label: mapEntry.value.key,
                        items: mapEntry.value.value,
                        onTap: (n) {
                          ref
                              .read(notificationsProvider.notifier)
                              .markRead(n.id);
                          if (n.deepLink != null) {
                            context.go(n.deepLink!);
                          }
                        },
                        onDismiss: (id) => ref
                            .read(notificationsProvider.notifier)
                            .dismiss(id),
                      ),
                    ))
                .toList(),
          );
        },
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (_, __) => const Center(child: Text('Unable to load notifications')),
      ),
    );
  }

  Map<String, List<AppNotification>> _groupByDate(
      List<AppNotification> notifications) {
    final now = DateTime.now();
    final today = DateTime(now.year, now.month, now.day);
    final yesterday = today.subtract(const Duration(days: 1));
    final weekAgo = today.subtract(const Duration(days: 7));

    final groups = <String, List<AppNotification>>{};

    for (final n in notifications) {
      final date = DateTime(n.timestamp.year, n.timestamp.month, n.timestamp.day);
      String label;
      if (date == today || date.isAfter(today)) {
        label = 'TODAY';
      } else if (date == yesterday || (date.isAfter(yesterday) && date.isBefore(today))) {
        label = 'YESTERDAY';
      } else if (date.isAfter(weekAgo)) {
        label = 'THIS WEEK';
      } else {
        label = 'EARLIER';
      }
      groups.putIfAbsent(label, () => []).add(n);
    }

    return groups;
  }
}
