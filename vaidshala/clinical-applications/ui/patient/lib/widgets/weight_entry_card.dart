// lib/widgets/weight_entry_card.dart
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/vitals_entry_provider.dart';
import '../theme.dart';

class WeightEntryCard extends ConsumerWidget {
  final VoidCallback onSaved;

  const WeightEntryCard({super.key, required this.onSaved});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(vitalsEntryProvider);
    final notifier = ref.read(vitalsEntryProvider.notifier);

    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.monitor_weight, color: AppColors.primaryTeal, size: 20),
                const SizedBox(width: 8),
                const Text('Weight',
                    style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
                const Spacer(),
                const Text('kg',
                    style: TextStyle(fontSize: 12, color: AppColors.textSecondary)),
              ],
            ),
            const SizedBox(height: 12),
            TextFormField(
              decoration: InputDecoration(
                labelText: 'Weight',
                hintText: '82.5',
                errorText: state.errors['weight'],
                border: const OutlineInputBorder(),
              ),
              keyboardType: const TextInputType.numberWithOptions(decimal: true),
              inputFormatters: [
                FilteringTextInputFormatter.allow(RegExp(r'[\d.]')),
              ],
              onChanged: notifier.setWeight,
            ),
            const SizedBox(height: 12),
            SizedBox(
              width: double.infinity,
              child: FilledButton(
                onPressed: () async {
                  final saved = await notifier.saveWeight();
                  if (saved && context.mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(content: Text('Weight saved')),
                    );
                    onSaved();
                  }
                },
                child: const Text('Save'),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
