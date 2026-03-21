// lib/widgets/glucose_entry_card.dart
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/vitals_entry_provider.dart';
import '../theme.dart';
import 'animations/animations.dart';

class GlucoseEntryCard extends ConsumerStatefulWidget {
  final VoidCallback onSaved;

  const GlucoseEntryCard({super.key, required this.onSaved});

  @override
  ConsumerState<GlucoseEntryCard> createState() => _GlucoseEntryCardState();
}

class _GlucoseEntryCardState extends ConsumerState<GlucoseEntryCard> {
  final GlobalKey<ShakeWidgetState> _shakeKey = GlobalKey<ShakeWidgetState>();

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(vitalsEntryProvider);
    final notifier = ref.read(vitalsEntryProvider.notifier);

    return ShakeWidget(
      key: _shakeKey,
      child: Card(
        margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  const Icon(Icons.bloodtype, color: AppColors.scoreRed, size: 20),
                  const SizedBox(width: 8),
                  const Text('Blood Glucose',
                      style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
                  const Spacer(),
                  const Text('mg/dL',
                      style: TextStyle(fontSize: 12, color: AppColors.textSecondary)),
                ],
              ),
              const SizedBox(height: 12),
              Row(
                children: [
                  Expanded(
                    flex: 2,
                    child: TextFormField(
                      decoration: InputDecoration(
                        labelText: 'Glucose',
                        hintText: '178',
                        errorText: state.errors['glucose'],
                        border: const OutlineInputBorder(),
                      ),
                      keyboardType: TextInputType.number,
                      inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                      onChanged: notifier.setGlucoseValue,
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    flex: 3,
                    child: DropdownButtonFormField<String>(
                      value: state.glucoseContext,
                      decoration: InputDecoration(
                        labelText: 'Context',
                        errorText: state.errors['glucoseContext'],
                        border: const OutlineInputBorder(),
                      ),
                      items: const [
                        DropdownMenuItem(value: 'fasting', child: Text('Fasting')),
                        DropdownMenuItem(value: 'post-meal', child: Text('Post-meal')),
                        DropdownMenuItem(value: 'random', child: Text('Random')),
                      ],
                      onChanged: (v) => notifier.setGlucoseContext(v),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 12),
              SpringTapCard(
                onTap: () async {
                  final saved = await notifier.saveGlucose();
                  if (saved && context.mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(content: Text('Blood glucose saved')),
                    );
                    widget.onSaved();
                  } else {
                    _shakeKey.currentState?.shake();
                  }
                },
                child: SizedBox(
                  width: double.infinity,
                  child: FilledButton(
                    onPressed: () {},
                    child: const Text('Save'),
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
