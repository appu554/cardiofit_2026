// lib/widgets/notification_date_group.dart
import 'package:flutter/material.dart';
import '../models/app_notification.dart';
import '../theme.dart';
import 'notification_item.dart';

class NotificationDateGroup extends StatelessWidget {
  final String label;
  final List<AppNotification> items;
  final ValueChanged<AppNotification> onTap;
  final ValueChanged<String> onDismiss;

  const NotificationDateGroup({
    super.key,
    required this.label,
    required this.items,
    required this.onTap,
    required this.onDismiss,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
          child: Text(
            label,
            style: const TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.w600,
              color: AppColors.textSecondary,
              letterSpacing: 1.0,
            ),
          ),
        ),
        ...items.map((n) => NotificationItem(
              notification: n,
              onTap: () => onTap(n),
              onDismiss: () => onDismiss(n.id),
            )),
      ],
    );
  }
}
