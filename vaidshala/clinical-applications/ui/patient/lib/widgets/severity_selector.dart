// lib/widgets/severity_selector.dart
import 'package:flutter/material.dart';
import '../theme.dart';

class SeveritySelector extends StatelessWidget {
  final String? value;
  final ValueChanged<String> onChanged;

  const SeveritySelector({
    super.key,
    required this.value,
    required this.onChanged,
  });

  static const _options = [
    ('mild', 'Mild', AppColors.scoreGreen),
    ('moderate', 'Moderate', AppColors.scoreYellow),
    ('severe', 'Severe', AppColors.scoreRed),
  ];

  @override
  Widget build(BuildContext context) {
    return Row(
      children: _options.map((option) {
        final isSelected = value == option.$1;
        return Expanded(
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 4),
            child: GestureDetector(
              onTap: () => onChanged(option.$1),
              child: Container(
                padding: const EdgeInsets.symmetric(vertical: 10),
                decoration: BoxDecoration(
                  color: isSelected
                      ? option.$3.withValues(alpha: 0.15)
                      : Colors.grey.shade100,
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(
                    color: isSelected ? option.$3 : Colors.grey.shade300,
                    width: isSelected ? 2 : 1,
                  ),
                ),
                child: Center(
                  child: Text(
                    option.$2,
                    style: TextStyle(
                      color: isSelected ? option.$3 : AppColors.textSecondary,
                      fontWeight:
                          isSelected ? FontWeight.bold : FontWeight.normal,
                    ),
                  ),
                ),
              ),
            ),
          ),
        );
      }).toList(),
    );
  }
}
