// lib/providers/symptom_entry_provider.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'database_provider.dart';

class SymptomEntryState {
  final Set<String> selectedSymptoms;
  final String? severity;  // mild, moderate, severe
  final String notes;
  final DateTime timestamp;

  SymptomEntryState({
    this.selectedSymptoms = const {},
    this.severity,
    this.notes = '',
    DateTime? timestamp,
  }) : timestamp = timestamp ?? DateTime.now();

  bool get canSave => selectedSymptoms.isNotEmpty && severity != null;

  SymptomEntryState copyWith({
    Set<String>? selectedSymptoms,
    String? severity,
    String? notes,
    DateTime? timestamp,
  }) =>
      SymptomEntryState(
        selectedSymptoms: selectedSymptoms ?? this.selectedSymptoms,
        severity: severity ?? this.severity,
        notes: notes ?? this.notes,
        timestamp: timestamp ?? this.timestamp,
      );
}

final symptomEntryProvider =
    StateNotifierProvider<SymptomEntryNotifier, SymptomEntryState>(
        (ref) => SymptomEntryNotifier(ref));

class SymptomEntryNotifier extends StateNotifier<SymptomEntryState> {
  final Ref _ref;

  SymptomEntryNotifier(this._ref) : super(SymptomEntryState(timestamp: DateTime.now()));

  void toggleSymptom(String symptom) {
    final current = Set<String>.from(state.selectedSymptoms);
    if (current.contains(symptom)) {
      current.remove(symptom);
    } else {
      current.add(symptom);
    }
    state = state.copyWith(selectedSymptoms: current);
  }

  void setSeverity(String severity) =>
      state = state.copyWith(severity: severity);

  void setNotes(String notes) =>
      state = state.copyWith(notes: notes);

  void setTimestamp(DateTime timestamp) =>
      state = state.copyWith(timestamp: timestamp);

  Future<bool> save() async {
    if (!state.canSave) return false;
    try {
      final db = _ref.read(databaseProvider);
      final id = 'sym-${DateTime.now().millisecondsSinceEpoch}';
      await db.insertSymptom(
        id: id,
        symptom: state.selectedSymptoms.join(','),
        severity: state.severity!,
        notes: state.notes.isEmpty ? null : state.notes,
      );
      reset();
      return true;
    } catch (_) {
      return false;
    }
  }

  void reset() =>
      state = SymptomEntryState(timestamp: DateTime.now());
}
