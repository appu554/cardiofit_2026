import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/models/progress_metric.dart';
import 'package:vaidshala_patient/widgets/progress_metric_row.dart';

void main() {
  testWidgets('ProgressMetricRow shows metric name and current value',
      (tester) async {
    const metric = ProgressMetric(
      id: 'fbg',
      name: 'Fasting Blood Glucose',
      icon: 'bloodtype',
      current: 178,
      previous: 185,
      target: 126,
      unit: 'mg/dL',
      improving: true,
    );

    await tester.pumpWidget(
      const MaterialApp(home: Scaffold(body: ProgressMetricRow(metric: metric))),
    );

    expect(find.text('Fasting Blood Glucose'), findsOneWidget);
    expect(find.text('178 mg/dL'), findsOneWidget);
    expect(find.text('Improving'), findsOneWidget);
    expect(find.text('Target: 126'), findsOneWidget);
  });

  testWidgets('shows no Improving badge when not improving', (tester) async {
    const metric = ProgressMetric(
      id: 'hba1c',
      name: 'HbA1c',
      icon: 'science',
      current: 8.9,
      previous: 8.5,
      target: 7.0,
      unit: '%',
      improving: false,
    );

    await tester.pumpWidget(
      const MaterialApp(home: Scaffold(body: ProgressMetricRow(metric: metric))),
    );

    expect(find.text('Improving'), findsNothing);
  });
}
