// lib/widgets/symptom_logger_sheet.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/symptom_entry_provider.dart';
import '../theme.dart';
import 'severity_selector.dart';
import 'symptom_chip_grid.dart';
import 'animations/animations.dart';

class SymptomLoggerSheet extends ConsumerWidget {
  const SymptomLoggerSheet({super.key});

  static const _symptoms = [
    'Dizziness',
    'Nausea',
    'Fatigue',
    'Chest Pain',
    'Swelling',
    'Breathlessness',
    'Low Sugar Feeling',
  ];

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(symptomEntryProvider);
    final notifier = ref.read(symptomEntryProvider.notifier);

    return DraggableScrollableSheet(
      initialChildSize: 0.8,
      minChildSize: 0.5,
      maxChildSize: 0.95,
      expand: false,
      builder: (context, scrollController) {
        return GlassmorphicContainer(
          borderRadius: 20,
          child: Column(
            children: [
              // Drag handle
              Container(
                margin: const EdgeInsets.symmetric(vertical: 8),
                width: 40,
                height: 4,
                decoration: BoxDecoration(
                  color: Colors.grey.shade300,
                  borderRadius: BorderRadius.circular(2),
                ),
              ),
              const Padding(
                padding: EdgeInsets.fromLTRB(16, 8, 16, 16),
                child: Text(
                  'Log a Symptom',
                  style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold),
                ),
              ),
              Expanded(
                child: ListView(
                  controller: scrollController,
                  padding: const EdgeInsets.symmetric(horizontal: 16),
                  children: [
                    // Symptom chips
                    StaggeredItem(
                      index: 0,
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          const Text('What are you feeling?',
                              style: TextStyle(
                                  fontSize: 14, fontWeight: FontWeight.w600)),
                          const SizedBox(height: 8),
                          SymptomChipGrid(
                            symptoms: _symptoms,
                            selected: state.selectedSymptoms,
                            onToggle: notifier.toggleSymptom,
                          ),
                          const SizedBox(height: 20),
                        ],
                      ),
                    ),

                    // Severity
                    StaggeredItem(
                      index: 1,
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          const Text('How severe?',
                              style: TextStyle(
                                  fontSize: 14, fontWeight: FontWeight.w600)),
                          const SizedBox(height: 8),
                          SeveritySelector(
                            value: state.severity,
                            onChanged: notifier.setSeverity,
                          ),
                          const SizedBox(height: 20),
                        ],
                      ),
                    ),

                    // Free text
                    StaggeredItem(
                      index: 2,
                      child: Column(
                        children: [
                          TextFormField(
                            decoration: const InputDecoration(
                              labelText: 'Tell us more (optional)',
                              border: OutlineInputBorder(),
                              counterText: '',
                            ),
                            maxLength: 200,
                            maxLines: 3,
                            onChanged: notifier.setNotes,
                          ),
                          const SizedBox(height: 20),
                        ],
                      ),
                    ),

                    // Time selector
                    StaggeredItem(
                      index: 3,
                      child: Column(
                        children: [
                          ListTile(
                            leading: const Icon(Icons.access_time),
                            title: const Text('When did this happen?'),
                            subtitle: Text(
                              _formatTime(state.timestamp),
                              style:
                                  const TextStyle(color: AppColors.primaryTeal),
                            ),
                            onTap: () async {
                              final time = await showTimePicker(
                                context: context,
                                initialTime:
                                    TimeOfDay.fromDateTime(state.timestamp),
                              );
                              if (time != null) {
                                final now = DateTime.now();
                                notifier.setTimestamp(DateTime(
                                  now.year, now.month, now.day,
                                  time.hour, time.minute,
                                ));
                              }
                            },
                          ),
                          const SizedBox(height: 16),
                        ],
                      ),
                    ),

                    // Save button
                    StaggeredItem(
                      index: 4,
                      child: state.canSave
                          ? SpringTapCard(
                              onTap: () async {
                                final saved = await notifier.save();
                                if (saved && context.mounted) {
                                  ScaffoldMessenger.of(context).showSnackBar(
                                    const SnackBar(
                                        content: Text('Symptom logged')),
                                  );
                                  Navigator.pop(context);
                                }
                              },
                              child: SizedBox(
                                width: double.infinity,
                                child: FilledButton(
                                  onPressed: () {},
                                  child: const Text('Save'),
                                ),
                              ),
                            )
                          : SizedBox(
                              width: double.infinity,
                              child: FilledButton(
                                onPressed: null,
                                child: const Text('Save'),
                              ),
                            ),
                    ),
                    const SizedBox(height: 32),
                  ],
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  String _formatTime(DateTime dt) {
    final hour = dt.hour.toString().padLeft(2, '0');
    final min = dt.minute.toString().padLeft(2, '0');
    return '$hour:$min';
  }
}
