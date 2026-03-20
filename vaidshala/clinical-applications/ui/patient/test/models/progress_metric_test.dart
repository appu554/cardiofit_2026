import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/models/progress_metric.dart';

void main() {
  group('ProgressMetric', () {
    test('creates with required fields', () {
      const metric = ProgressMetric(
        id: 'fbg',
        name: 'Fasting Blood Glucose',
        icon: 'bloodtype',
        current: 178,
        previous: 185,
        target: 126,
        unit: 'mg/dL',
      );
      expect(metric.id, 'fbg');
      expect(metric.improving, false);
      expect(metric.sparkline, isEmpty);
    });

    test('serializes to/from JSON', () {
      const metric = ProgressMetric(
        id: 'fbg',
        name: 'FBG',
        icon: 'bloodtype',
        current: 178,
        previous: 185,
        target: 126,
        unit: 'mg/dL',
        improving: true,
        sparkline: [190, 185, 182, 178],
      );
      final json = metric.toJson();
      final restored = ProgressMetric.fromJson(json);
      expect(restored, metric);
    });
  });
}
