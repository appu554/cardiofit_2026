// lib/widgets/notification_date_group.dart
import 'package:flutter/material.dart';
import '../models/app_notification.dart';
import '../theme.dart';
import 'notification_item.dart';
import 'animations/animations.dart';

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
          child: TweenAnimationBuilder<double>(
            tween: Tween(begin: 0.0, end: 1.0),
            duration: const Duration(milliseconds: 400),
            curve: const Cubic(0.25, 0.46, 0.45, 0.94),
            builder: (context, value, child) => Opacity(
              opacity: value,
              child: Transform.translate(
                offset: Offset(0, 12 * (1.0 - value)),
                child: child,
              ),
            ),
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
        ),
        ...items.asMap().entries.map((entry) => StaggeredItem(
              index: entry.key,
              child: NotificationItem(
                notification: entry.value,
                onTap: () => onTap(entry.value),
                onDismiss: () => onDismiss(entry.value.id),
              ),
            )),
      ],
    );
  }
}
