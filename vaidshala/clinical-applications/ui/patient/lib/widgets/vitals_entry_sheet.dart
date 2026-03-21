// lib/widgets/vitals_entry_sheet.dart
import 'package:flutter/material.dart';
import 'bp_entry_card.dart';
import 'glucose_entry_card.dart';
import 'weight_entry_card.dart';

class VitalsEntrySheet extends StatelessWidget {
  const VitalsEntrySheet({super.key});

  @override
  Widget build(BuildContext context) {
    return DraggableScrollableSheet(
      initialChildSize: 0.85,
      minChildSize: 0.5,
      maxChildSize: 0.95,
      expand: false,
      builder: (context, scrollController) {
        return Container(
          decoration: const BoxDecoration(
            color: Colors.white,
            borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
          ),
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
                padding: EdgeInsets.fromLTRB(16, 8, 16, 8),
                child: Text(
                  'Log a Reading',
                  style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold),
                ),
              ),
              Expanded(
                child: ListView(
                  controller: scrollController,
                  children: [
                    BpEntryCard(onSaved: () => Navigator.pop(context)),
                    GlucoseEntryCard(onSaved: () => Navigator.pop(context)),
                    WeightEntryCard(onSaved: () => Navigator.pop(context)),
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
}
