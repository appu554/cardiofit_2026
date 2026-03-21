import 'package:flutter/material.dart';
import '../models/timeline_entry.dart' as model;
import '../theme.dart';
import '../utils/icon_mapper.dart';

class TimelineEntryWidget extends StatelessWidget {
  final model.TimelineEntry entry;
  final bool isLast;

  const TimelineEntryWidget({
    super.key,
    required this.entry,
    this.isLast = false,
  });

  @override
  Widget build(BuildContext context) {
    final done = entry.done;

    return IntrinsicHeight(
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Time column
          SizedBox(
            width: 50,
            child: Padding(
              padding: const EdgeInsets.only(top: 2),
              child: Text(
                entry.time,
                style: TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                  color: done ? AppColors.textSecondary : AppColors.textPrimary,
                ),
                textAlign: TextAlign.right,
              ),
            ),
          ),
          const SizedBox(width: 12),
          // Connector column
          Column(
            children: [
              Container(
                width: 12,
                height: 12,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: done ? AppColors.scoreGreen : Colors.grey.shade400,
                  border: Border.all(
                    color: done ? AppColors.scoreGreen : Colors.grey.shade400,
                    width: 2,
                  ),
                ),
                child: done
                    ? const Icon(Icons.check, size: 8, color: Colors.white)
                    : null,
              ),
              if (!isLast)
                Expanded(
                  child: Container(
                    width: 2,
                    color: done ? AppColors.scoreGreen : Colors.grey.shade300,
                  ),
                ),
            ],
          ),
          const SizedBox(width: 12),
          // Content column
          Expanded(
            child: Padding(
              padding: const EdgeInsets.only(bottom: 20),
              child: Row(
                children: [
                  Icon(
                    IconMapper.fromString(entry.icon),
                    size: 20,
                    color: done ? AppColors.textSecondary : AppColors.primaryTeal,
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      entry.text,
                      style: TextStyle(
                        fontSize: 14,
                        decoration: done ? TextDecoration.lineThrough : null,
                        color: done
                            ? AppColors.textSecondary
                            : AppColors.textPrimary,
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}
