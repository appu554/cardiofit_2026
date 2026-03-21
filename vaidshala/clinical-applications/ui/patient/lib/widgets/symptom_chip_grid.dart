// lib/widgets/symptom_chip_grid.dart
import 'package:flutter/material.dart';
import '../theme.dart';

/// A single symptom chip used within [SymptomChipGrid].
class SymptomChip extends StatelessWidget {
  final String label;
  final bool selected;
  final VoidCallback onTap;

  const SymptomChip({
    super.key,
    required this.label,
    required this.selected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 200),
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        decoration: BoxDecoration(
          color: selected ? AppColors.primaryTeal : Colors.grey.shade100,
          borderRadius: BorderRadius.circular(20),
          border: Border.all(
            color: selected ? AppColors.primaryTeal : Colors.grey.shade300,
          ),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (selected)
              const Padding(
                padding: EdgeInsets.only(right: 4),
                child: Icon(Icons.check, size: 16, color: Colors.white),
              ),
            Text(
              label,
              style: TextStyle(
                color: selected ? Colors.white : AppColors.textPrimary,
                fontWeight: selected ? FontWeight.w600 : FontWeight.normal,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// A Wrap of [SymptomChip] widgets for multi-select symptom entry.
class SymptomChipGrid extends StatelessWidget {
  final List<String> symptoms;
  final Set<String> selected;
  final ValueChanged<String> onToggle;

  const SymptomChipGrid({
    super.key,
    required this.symptoms,
    required this.selected,
    required this.onToggle,
  });

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: symptoms
          .map((s) => SymptomChip(
                label: s,
                selected: selected.contains(s),
                onTap: () => onToggle(s),
              ))
          .toList(),
    );
  }
}
