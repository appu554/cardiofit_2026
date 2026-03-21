// lib/widgets/symptom_entry_sheet.dart
// Thin wrapper — the actual implementation lives in symptom_logger_sheet.dart
import 'package:flutter/material.dart';
import 'symptom_logger_sheet.dart';

/// SpeedDial FAB alias for the Symptom Logger bottom sheet.
class SymptomEntrySheet extends StatelessWidget {
  const SymptomEntrySheet({super.key});

  @override
  Widget build(BuildContext context) => const SymptomLoggerSheet();
}
