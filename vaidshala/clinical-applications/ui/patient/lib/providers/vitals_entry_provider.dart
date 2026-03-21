// lib/providers/vitals_entry_provider.dart
import 'dart:convert';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'database_provider.dart';

class VitalsEntryState {
  final String systolic;
  final String diastolic;
  final String glucoseValue;
  final String? glucoseContext; // fasting, post-meal, random
  final String weight;
  final Map<String, String?> errors;

  const VitalsEntryState({
    this.systolic = '',
    this.diastolic = '',
    this.glucoseValue = '',
    this.glucoseContext,
    this.weight = '',
    this.errors = const {},
  });

  VitalsEntryState copyWith({
    String? systolic,
    String? diastolic,
    String? glucoseValue,
    String? glucoseContext,
    String? weight,
    Map<String, String?>? errors,
  }) =>
      VitalsEntryState(
        systolic: systolic ?? this.systolic,
        diastolic: diastolic ?? this.diastolic,
        glucoseValue: glucoseValue ?? this.glucoseValue,
        glucoseContext: glucoseContext ?? this.glucoseContext,
        weight: weight ?? this.weight,
        errors: errors ?? this.errors,
      );
}

final vitalsEntryProvider =
    StateNotifierProvider<VitalsEntryNotifier, VitalsEntryState>(
        (ref) => VitalsEntryNotifier(ref));

class VitalsEntryNotifier extends StateNotifier<VitalsEntryState> {
  final Ref _ref;

  VitalsEntryNotifier(this._ref) : super(const VitalsEntryState());

  void setSystolic(String v) => state = state.copyWith(systolic: v);
  void setDiastolic(String v) => state = state.copyWith(diastolic: v);
  void setGlucoseValue(String v) => state = state.copyWith(glucoseValue: v);
  void setGlucoseContext(String? v) => state = state.copyWith(glucoseContext: v);
  void setWeight(String v) => state = state.copyWith(weight: v);

  bool validateBp() {
    final errors = <String, String?>{...state.errors};
    final sys = double.tryParse(state.systolic);
    final dia = double.tryParse(state.diastolic);

    if (sys == null || sys < 60 || sys > 250) {
      errors['systolic'] = 'Enter a value between 60 and 250';
    } else {
      errors.remove('systolic');
    }

    if (dia == null || dia < 40 || dia > 150) {
      errors['diastolic'] = 'Enter a value between 40 and 150';
    } else {
      errors.remove('diastolic');
    }

    if (sys != null && dia != null && sys <= dia) {
      errors['systolic'] = 'Systolic must be higher than diastolic';
    }

    state = state.copyWith(errors: errors);
    return !errors.containsKey('systolic') && !errors.containsKey('diastolic');
  }

  bool validateGlucose() {
    final errors = <String, String?>{...state.errors};
    final val = double.tryParse(state.glucoseValue);

    if (val == null || val < 20 || val > 600) {
      errors['glucose'] = 'Enter a value between 20 and 600';
    } else {
      errors.remove('glucose');
    }

    if (state.glucoseContext == null) {
      errors['glucoseContext'] = 'Select when this was measured';
    } else {
      errors.remove('glucoseContext');
    }

    state = state.copyWith(errors: errors);
    return !errors.containsKey('glucose') && !errors.containsKey('glucoseContext');
  }

  bool validateWeight() {
    final errors = <String, String?>{...state.errors};
    final val = double.tryParse(state.weight);

    if (val == null || val < 20 || val > 300) {
      errors['weight'] = 'Enter a value between 20 and 300';
    } else {
      errors.remove('weight');
    }

    state = state.copyWith(errors: errors);
    return !errors.containsKey('weight');
  }

  Future<bool> saveBp() async {
    if (!validateBp()) return false;
    try {
      final db = _ref.read(databaseProvider);
      final id = 'obs-bp-${DateTime.now().millisecondsSinceEpoch}';
      await db.insertObservation(
        id: id,
        type: 'bp',
        value: jsonEncode({
          'systolic': int.parse(state.systolic),
          'diastolic': int.parse(state.diastolic),
        }),
        unit: 'mmHg',
      );
      state = state.copyWith(systolic: '', diastolic: '');
      return true;
    } catch (_) {
      return false;
    }
  }

  Future<bool> saveGlucose() async {
    if (!validateGlucose()) return false;
    try {
      final db = _ref.read(databaseProvider);
      final id = 'obs-gluc-${DateTime.now().millisecondsSinceEpoch}';
      await db.insertObservation(
        id: id,
        type: 'glucose',
        value: jsonEncode({
          'value': double.parse(state.glucoseValue),
          'context': state.glucoseContext,
        }),
        unit: 'mg/dL',
      );
      state = state.copyWith(glucoseValue: '', glucoseContext: null);
      return true;
    } catch (_) {
      return false;
    }
  }

  Future<bool> saveWeight() async {
    if (!validateWeight()) return false;
    try {
      final db = _ref.read(databaseProvider);
      final id = 'obs-wt-${DateTime.now().millisecondsSinceEpoch}';
      await db.insertObservation(
        id: id,
        type: 'weight',
        value: jsonEncode({'value': double.parse(state.weight)}),
        unit: 'kg',
      );
      state = state.copyWith(weight: '');
      return true;
    } catch (_) {
      return false;
    }
  }

  void reset() => state = const VitalsEntryState();
}
