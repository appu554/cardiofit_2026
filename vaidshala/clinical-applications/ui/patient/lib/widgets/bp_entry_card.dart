// lib/widgets/bp_entry_card.dart
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/vitals_entry_provider.dart';
import '../theme.dart';
import 'animations/animations.dart';

class BpEntryCard extends ConsumerStatefulWidget {
  final VoidCallback onSaved;

  const BpEntryCard({super.key, required this.onSaved});

  @override
  ConsumerState<BpEntryCard> createState() => _BpEntryCardState();
}

class _BpEntryCardState extends ConsumerState<BpEntryCard> {
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
                  const Icon(Icons.favorite, color: AppColors.scoreRed, size: 20),
                  const SizedBox(width: 8),
                  const Text('Blood Pressure',
                      style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
                  const Spacer(),
                  const Text('mmHg',
                      style: TextStyle(fontSize: 12, color: AppColors.textSecondary)),
                ],
              ),
              const SizedBox(height: 12),
              Row(
                children: [
                  Expanded(
                    child: TextFormField(
                      decoration: InputDecoration(
                        labelText: 'Systolic',
                        hintText: '156',
                        errorText: state.errors['systolic'],
                        border: const OutlineInputBorder(),
                      ),
                      keyboardType: TextInputType.number,
                      inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                      onChanged: notifier.setSystolic,
                    ),
                  ),
                  const Padding(
                    padding: EdgeInsets.symmetric(horizontal: 8),
                    child: Text('/', style: TextStyle(fontSize: 20)),
                  ),
                  Expanded(
                    child: TextFormField(
                      decoration: InputDecoration(
                        labelText: 'Diastolic',
                        hintText: '98',
                        errorText: state.errors['diastolic'],
                        border: const OutlineInputBorder(),
                      ),
                      keyboardType: TextInputType.number,
                      inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                      onChanged: notifier.setDiastolic,
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 12),
              SpringTapCard(
                onTap: () async {
                  final saved = await notifier.saveBp();
                  if (saved && context.mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(content: Text('Blood pressure saved')),
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
