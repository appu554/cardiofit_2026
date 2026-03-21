import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:vaidshala_patient/models/timeline_entry.dart';
import 'package:vaidshala_patient/widgets/timeline_entry_widget.dart';

void main() {
  testWidgets('TimelineEntryWidget shows time and text', (tester) async {
    const entry = TimelineEntry(
      id: 'act-001',
      time: '06:30',
      text: 'Measure fasting blood glucose',
      icon: 'bloodtype',
      done: true,
    );

    await tester.pumpWidget(
      const MaterialApp(
          home: Scaffold(body: TimelineEntryWidget(entry: entry))),
    );

    expect(find.text('06:30'), findsOneWidget);
    expect(find.text('Measure fasting blood glucose'), findsOneWidget);
  });

  testWidgets('done entry has strikethrough text', (tester) async {
    const entry = TimelineEntry(
      id: 'act-002',
      time: '08:00',
      text: 'Take Metformin 1000mg',
      icon: 'medication',
      done: true,
    );

    await tester.pumpWidget(
      const MaterialApp(
          home: Scaffold(body: TimelineEntryWidget(entry: entry))),
    );

    final text = tester.widget<Text>(find.text('Take Metformin 1000mg'));
    expect(text.style?.decoration, TextDecoration.lineThrough);
  });
}
