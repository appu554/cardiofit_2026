import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/models/vital_entry.dart';

void main() {
  group('VitalEntry', () {
    test('creates with default synced=false', () {
      final v = VitalEntry(
        id: 'v1',
        type: 'bp',
        value: '{"systolic":156,"diastolic":98}',
        unit: 'mmHg',
        timestamp: DateTime(2026, 3, 21),
      );
      expect(v.synced, false);
      expect(v.type, 'bp');
    });

    test('serializes to/from JSON', () {
      final v = VitalEntry(
        id: 'v2',
        type: 'glucose',
        value: '{"value":178,"context":"fasting"}',
        unit: 'mg/dL',
        timestamp: DateTime(2026, 3, 21),
        synced: true,
      );
      final json = v.toJson();
      final restored = VitalEntry.fromJson(json);
      expect(restored, v);
    });
  });
}
