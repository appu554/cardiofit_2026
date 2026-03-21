// lib/widgets/notification_item.dart
import 'package:flutter/material.dart';
import '../models/app_notification.dart';
import '../theme.dart';

class NotificationItem extends StatelessWidget {
  final AppNotification notification;
  final VoidCallback onTap;
  final VoidCallback onDismiss;

  const NotificationItem({
    super.key,
    required this.notification,
    required this.onTap,
    required this.onDismiss,
  });

  IconData get _typeIcon {
    switch (notification.type) {
      case NotificationType.coaching:
        return Icons.school;
      case NotificationType.reminder:
        return Icons.alarm;
      case NotificationType.alert:
        return Icons.warning_amber;
      case NotificationType.milestone:
        return Icons.emoji_events;
    }
  }

  Color get _typeColor {
    switch (notification.type) {
      case NotificationType.coaching:
        return AppColors.scoreGreen;
      case NotificationType.reminder:
        return AppColors.primaryTeal;
      case NotificationType.alert:
        return Colors.orange;
      case NotificationType.milestone:
        return Colors.purple;
    }
  }

  String get _timeAgo {
    final diff = DateTime.now().difference(notification.timestamp);
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    if (diff.inHours < 24) return '${diff.inHours}h ago';
    if (diff.inDays == 1) return 'Yesterday';
    return '${diff.inDays}d ago';
  }

  @override
  Widget build(BuildContext context) {
    return Dismissible(
      key: Key(notification.id),
      direction: DismissDirection.endToStart,
      onDismissed: (_) => onDismiss(),
      background: Container(
        color: AppColors.scoreRed,
        alignment: Alignment.centerRight,
        padding: const EdgeInsets.only(right: 16),
        child: const Icon(Icons.delete, color: Colors.white),
      ),
      child: ListTile(
        leading: CircleAvatar(
          backgroundColor: _typeColor.withValues(alpha: 0.15),
          child: Icon(_typeIcon, color: _typeColor, size: 20),
        ),
        title: Text(
          notification.title,
          style: TextStyle(
            fontWeight: notification.read ? FontWeight.normal : FontWeight.bold,
          ),
        ),
        subtitle: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(notification.body, maxLines: 2, overflow: TextOverflow.ellipsis),
            const SizedBox(height: 2),
            Text(_timeAgo,
                style: const TextStyle(fontSize: 11, color: AppColors.textSecondary)),
          ],
        ),
        trailing: notification.read
            ? null
            : Container(
                width: 8,
                height: 8,
                decoration: const BoxDecoration(
                  color: Colors.blue,
                  shape: BoxShape.circle,
                ),
              ),
        onTap: onTap,
      ),
    );
  }
}
